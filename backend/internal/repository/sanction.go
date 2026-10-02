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
	ErrSanctionInvalidRequest        = errors.New("invalid sanction request")
	ErrSanctionForbidden             = errors.New("can only appeal own sanctions")
	ErrSanctionRevoked               = errors.New("sanction already revoked")
	ErrSanctionAlreadyAppealed       = errors.New("sanction already appealed")
	ErrSanctionNotFound              = errors.New("sanction not found or already revoked")
	ErrSanctionAppealNotFound        = errors.New("sanction appeal not found")
	ErrSanctionAppealAlreadyResolved = errors.New("sanction appeal already resolved")
)

type SanctionRepository struct{ db *gorm.DB }

func NewSanctionRepository(db *gorm.DB) *SanctionRepository { return &SanctionRepository{db: db} }

func (r *SanctionRepository) Create(ctx context.Context, sanction *models.Sanction) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(sanction).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{
			UserID: &sanction.ImposedBy, Action: "create", ResourceType: "sanction", ResourceID: &sanction.ID,
			Changes: map[string]interface{}{"type": sanction.Type, "user_id": sanction.UserID, "reason": sanction.Reason},
		}).Error
	})
}

func (r *SanctionRepository) HasActiveSubmissionRestriction(ctx context.Context, userID string) (bool, error) {
	var count int64
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&models.Sanction{}).
		Where("user_id = ? AND revoked_at IS NULL AND starts_at <= ?", userID, now).
		Where("type = ? OR (type = ? AND ends_at > ?)", "submission_ban", "submission_suspension", now).
		Count(&count).Error
	return count > 0, err
}

func (r *SanctionRepository) Revoke(ctx context.Context, id, actorID, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sanction models.Sanction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&sanction).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSanctionNotFound
			}
			return err
		}
		if sanction.RevokedAt != nil {
			return ErrSanctionNotFound
		}
		return revokeSanction(tx, id, actorID, reason, time.Now())
	})
}

// revokeSanction runs only while the sanction row is locked by the caller.
func revokeSanction(tx *gorm.DB, id, actorID, reason string, now time.Time) error {
	result := tx.Model(&models.Sanction{}).Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]interface{}{"revoked_at": now, "revoked_by": actorID, "revoke_reason": reason})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrSanctionNotFound
	}
	return tx.Create(&models.AuditLog{
		UserID: &actorID, Action: "revoke", ResourceType: "sanction", ResourceID: &id,
		Changes: map[string]interface{}{"reason": reason},
	}).Error
}

func (r *SanctionRepository) List(ctx context.Context, offset, limit int, userID string, activeOnly bool) ([]models.Sanction, int64, error) {
	var rows []models.Sanction
	var total int64
	query := r.db.WithContext(ctx).Model(&models.Sanction{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if activeOnly {
		now := time.Now()
		query = query.Where("revoked_at IS NULL AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)", now, now)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&rows).Error
	return rows, total, err
}

func (r *SanctionRepository) GetByID(ctx context.Context, id string) (*models.Sanction, error) {
	var row models.Sanction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSanctionNotFound
		}
		return nil, err
	}
	return &row, nil
}

// CreateAppeal serializes submission with revocation and other submissions.
// Ownership and eligibility are checked under the sanction lock, and no appeal
// can commit without its audit. Lock ordering matches resolution: we never lock
// an existing appeal here (only read it while holding the sanction lock).
func (r *SanctionRepository) CreateAppeal(ctx context.Context, appeal *models.SanctionAppeal) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sanction models.Sanction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", appeal.SanctionID).First(&sanction).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSanctionNotFound
			}
			return err
		}
		if sanction.UserID != appeal.SubmittedBy {
			return ErrSanctionForbidden
		}
		if sanction.RevokedAt != nil {
			return ErrSanctionRevoked
		}
		var count int64
		if err := tx.Model(&models.SanctionAppeal{}).Where("sanction_id = ?", appeal.SanctionID).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return ErrSanctionAlreadyAppealed
		}
		// A caller cannot submit a pre-resolved appeal.
		appeal.Status = "pending"
		appeal.ReviewedBy, appeal.ReviewNotes, appeal.ReviewedAt = nil, nil, nil
		if err := tx.Create(appeal).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{
			UserID: &appeal.SubmittedBy, Action: "appeal", ResourceType: "sanction", ResourceID: &appeal.SanctionID,
			Changes: map[string]interface{}{"appeal_id": appeal.ID, "reason": appeal.Reason},
		}).Error
	})
}

func (r *SanctionRepository) GetAppealBySanctionID(ctx context.Context, sanctionID string) (*models.SanctionAppeal, error) {
	var row models.SanctionAppeal
	err := r.db.WithContext(ctx).Where("sanction_id = ?", sanctionID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *SanctionRepository) ResolveAppeal(ctx context.Context, appealID, reviewerID, status, notes string) error {
	_, err := r.ResolveAppealWithConsequences(ctx, appealID, reviewerID, status, notes)
	return err
}

// ResolveAppealWithConsequences commits the terminal decision, any necessary
// revocation, and their audits together. Never hide a missing sanction as an
// already-revoked one: only a locked, existing revoked row is idempotent here.
func (r *SanctionRepository) ResolveAppealWithConsequences(ctx context.Context, appealID, reviewerID, status, notes string) (*models.SanctionAppeal, error) {
	if status != "approved" && status != "rejected" {
		return nil, errors.New("status must be approved or rejected")
	}
	var appeal models.SanctionAppeal
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", appealID).First(&appeal).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSanctionAppealNotFound
			}
			return err
		}
		if appeal.Status != "pending" || appeal.ReviewedAt != nil {
			return ErrSanctionAppealAlreadyResolved
		}
		if status == "approved" {
			var sanction models.Sanction
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", appeal.SanctionID).First(&sanction).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrSanctionNotFound
				}
				return err
			}
			if sanction.RevokedAt == nil {
				if err := revokeSanction(tx, sanction.ID, reviewerID, "sanction appeal approved: "+notes, time.Now()); err != nil {
					return err
				}
			}
		}
		now := time.Now()
		result := tx.Model(&models.SanctionAppeal{}).Where("id = ? AND status = ?", appealID, "pending").Updates(map[string]interface{}{
			"status": status, "reviewed_by": reviewerID, "review_notes": notes, "reviewed_at": now, "updated_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrSanctionAppealAlreadyResolved
		}
		if err := tx.Create(&models.AuditLog{
			UserID: &reviewerID, Action: "resolve_appeal", ResourceType: "sanction_appeal", ResourceID: &appealID,
			Changes: map[string]interface{}{"status": status, "notes": notes, "sanction_id": appeal.SanctionID},
		}).Error; err != nil {
			return err
		}
		appeal.Status, appeal.ReviewedBy, appeal.ReviewNotes = status, &reviewerID, &notes
		appeal.ReviewedAt, appeal.UpdatedAt = &now, now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &appeal, nil
}

func (r *SanctionRepository) GetAppealByID(ctx context.Context, id string) (*models.SanctionAppeal, error) {
	var row models.SanctionAppeal
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSanctionAppealNotFound
	}
	return &row, err
}
