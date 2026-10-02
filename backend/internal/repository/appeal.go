package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var (
	ErrAppealNotFound        = errors.New("appeal not found")
	ErrAppealForbidden       = errors.New("not permitted to retire this appeal")
	ErrAppealAlreadyResolved = errors.New("appeal already resolved")
	ErrAppealInvalidReview   = errors.New("review status must be approved or rejected")
	ErrAppealResolveRequired = errors.New("event appeals require the formal resolve endpoint")
	ErrAppealInvalidTarget   = errors.New("legacy review requires an existing case target")
)

// AppealRepository handles appeal database operations
type AppealRepository struct {
	db *gorm.DB
}

// NewAppealRepository creates a new appeal repository
func NewAppealRepository(db *gorm.DB) *AppealRepository {
	return &AppealRepository{db: db}
}

// CreateAppeal creates a new appeal
func (r *AppealRepository) CreateAppeal(ctx context.Context, appeal *models.Appeal) error {
	return r.db.WithContext(ctx).Create(appeal).Error
}

// GetAppealByID retrieves an appeal by ID
func (r *AppealRepository) GetAppealByID(ctx context.Context, id string) (*models.Appeal, error) {
	var appeal models.Appeal
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&appeal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppealNotFound
		}
		return nil, err
	}
	return &appeal, nil
}

// GetAppealsByCaseID retrieves all appeals for a case
func (r *AppealRepository) GetAppealsByCaseID(ctx context.Context, caseID string) ([]models.Appeal, error) {
	var appeals []models.Appeal
	err := r.db.WithContext(ctx).
		Where("case_id = ?", caseID).
		Order("created_at DESC").
		Find(&appeals).Error
	return appeals, err
}

func (r *AppealRepository) GetAppealsByEventID(ctx context.Context, eventID string) ([]models.Appeal, error) {
	var appeals []models.Appeal
	err := r.db.WithContext(ctx).Where("event_id = ?", eventID).Order("created_at DESC").Find(&appeals).Error
	return appeals, err
}

// ListAppeals lists appeals with pagination
func (r *AppealRepository) ListAppeals(ctx context.Context, offset, limit int, status string, submittedBy string) ([]models.Appeal, int64, error) {
	var appeals []models.Appeal
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Appeal{}).Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if submittedBy != "" {
		query = query.Where("submitted_by = ?", submittedBy)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&appeals).Error

	return appeals, total, err
}

// ReviewAppeal preserves Case-only legacy decisions without inventing a canonical
// outcome. Event decisions and their consequences belong to ResolveWithConsequences.
func (r *AppealRepository) ReviewAppeal(ctx context.Context, id, reviewedBy, status, reviewNotes string) error {
	if status != "approved" && status != "rejected" {
		return ErrAppealInvalidReview
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var appeal models.Appeal
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&appeal).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAppealNotFound
			}
			return err
		}
		if appeal.EventID != nil {
			return ErrAppealResolveRequired
		}
		if appeal.Status != "pending" || appeal.Outcome != nil || appeal.ReviewedAt != nil || appeal.ReviewedBy != nil || appeal.ResolutionReason != nil {
			return ErrAppealAlreadyResolved
		}
		if appeal.CaseID == nil {
			return ErrAppealInvalidTarget
		}
		// Read the complete Case under lock; never Save an ID-only model, which
		// clears unrelated fields. Legacy approval has no defined Case mutation
		// or canonical outcome, so its data remains unchanged.
		var caseObj models.Case
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *appeal.CaseID).First(&caseObj).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAppealInvalidTarget
			}
			return err
		}
		if err := tx.Model(&appeal).Updates(map[string]interface{}{
			"status": status, "reviewed_by": reviewedBy, "review_notes": reviewNotes, "reviewed_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{UserID: &reviewedBy, Action: "review", ResourceType: "appeal", ResourceID: &appeal.ID,
			Changes: map[string]interface{}{"old_status": "pending", "status": status, "review_notes": reviewNotes, "case_id": caseObj.ID},
		}).Error
	})
}

// ResolveAppeal persists the Phase 13 outcome alongside the legacy review state.
func (r *AppealRepository) ResolveAppeal(ctx context.Context, id, reviewedBy, status, outcome, reason string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Appeal{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": status, "outcome": outcome, "resolution_reason": reason,
		"reviewed_by": reviewedBy, "review_notes": reason, "reviewed_at": now,
	})
	if result.RowsAffected == 0 {
		return ErrAppealNotFound
	}
	return result.Error
}

// ResolveWithConsequences atomically records adjudication, event status, and the
// required warning/audits for malicious submissions.
func (r *AppealRepository) ResolveWithConsequences(ctx context.Context, id, reviewedBy, status, outcome, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var appeal models.Appeal
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&appeal).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAppealNotFound
			}
			return err
		}
		now := time.Now()
		if appeal.Status != "pending" {
			return errors.New("appeal already resolved")
		}
		if err := tx.Model(&appeal).Updates(map[string]interface{}{"status": status, "outcome": outcome, "resolution_reason": reason, "reviewed_by": reviewedBy, "review_notes": reason, "reviewed_at": now}).Error; err != nil {
			return err
		}
		if appeal.EventID != nil {
			var event models.Event
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *appeal.EventID).First(&event).Error; err != nil {
				return err
			}
			newStatus := ""
			if outcome == "corrected" {
				newStatus = "corrected"
			}
			if outcome == "withdrawn" {
				newStatus = "withdrawn"
			}
			if newStatus != "" {
				if err := tx.Model(&models.Event{}).Where("id = ?", *appeal.EventID).Updates(map[string]interface{}{"status": newStatus, "correction_note": reason}).Error; err != nil {
					return err
				}
				event.Status = newStatus
				event.CorrectionNote = &reason
				if err := recordEventVersion(tx, &event, "appeal_"+outcome, &reviewedBy); err != nil {
					return err
				}
			}
		} else if outcome == "withdrawn" && appeal.CaseID != nil {
			if err := tx.Model(&models.Case{}).Where("id = ?", *appeal.CaseID).Updates(map[string]interface{}{"status": "closed", "reviewed_by": reviewedBy, "reviewed_at": now, "verdict": reason}).Error; err != nil {
				return err
			}
		}
		appealAudit := &models.AuditLog{UserID: &reviewedBy, Action: "resolve", ResourceType: "appeal", ResourceID: &appeal.ID, Changes: map[string]interface{}{"outcome": outcome, "reason": reason}}
		if err := tx.Create(appealAudit).Error; err != nil {
			return err
		}
		if outcome == "malicious_submission" {
			if appeal.SubmittedBy == nil {
				return errors.New("appeal submitter required for malicious_submission")
			}
			warning := &models.Sanction{UserID: *appeal.SubmittedBy, Type: "warning", Reason: reason, RelatedEventID: appeal.EventID, RelatedAppealID: &appeal.ID, StartsAt: now, ImposedBy: reviewedBy}
			if err := tx.Create(warning).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.AuditLog{UserID: &reviewedBy, Action: "create", ResourceType: "sanction", ResourceID: &warning.ID, Changes: map[string]interface{}{"type": "warning", "appeal_id": appeal.ID, "user_id": warning.UserID}}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteAppeal preserves appeal history and only hides it from active reads.
// Retirement and the delete audit must commit together so active rows never hide without history.
func (r *AppealRepository) DeleteAppeal(ctx context.Context, id, deletedBy string, governance bool) error {
	if deletedBy == "" {
		return ErrAppealForbidden
	}
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var appeal models.Appeal
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&appeal).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAppealNotFound
			}
			return err
		}
		if !governance {
			if appeal.SubmittedBy == nil || *appeal.SubmittedBy != deletedBy {
				return ErrAppealForbidden
			}
			if appeal.Status != "pending" || appeal.Outcome != nil || appeal.ReviewedAt != nil {
				return ErrAppealAlreadyResolved
			}
		}
		result := tx.Model(&models.Appeal{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAppealNotFound
		}
		return tx.Create(&models.AuditLog{
			UserID:       &deletedBy,
			Action:       "delete",
			ResourceType: "appeal",
			ResourceID:   &id,
			Changes:      map[string]interface{}{"retired": true},
		}).Error
	})
}
