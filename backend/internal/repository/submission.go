package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
)

// SubmissionRepository handles submission database operations
type SubmissionRepository struct {
	db *gorm.DB
}

// NewSubmissionRepository creates a new submission repository
func NewSubmissionRepository(db *gorm.DB) *SubmissionRepository {
	return &SubmissionRepository{db: db}
}

// CreateSubmission creates a new submission
func (r *SubmissionRepository) CreateSubmission(ctx context.Context, submission *models.Submission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

// GetSubmissionByID retrieves a submission by ID
func (r *SubmissionRepository) GetSubmissionByID(ctx context.Context, id string) (*models.Submission, error) {
	var submission models.Submission
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&submission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	return &submission, nil
}

// ListSubmissions lists submissions with pagination and filters
func (r *SubmissionRepository) ListSubmissions(ctx context.Context, offset, limit int, status string, submittedBy string) ([]models.Submission, int64, error) {
	var submissions []models.Submission
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Submission{}).Where("deleted_at IS NULL")
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
		Find(&submissions).Error

	return submissions, total, err
}

// UpdateSubmission updates a submission
func (r *SubmissionRepository) UpdateSubmission(ctx context.Context, submission *models.Submission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}

// DeleteSubmission preserves submission history and only hides it from active reads.
// Retirement and the delete audit must commit together so active rows never hide without history.
func (r *SubmissionRepository) DeleteSubmission(ctx context.Context, id, deletedBy string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Submission{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrSubmissionNotFound
		}
		return tx.Create(&models.AuditLog{
			UserID:       &deletedBy,
			Action:       "delete",
			ResourceType: "submission",
			ResourceID:   &id,
			Changes:      map[string]interface{}{"retired": true},
		}).Error
	})
}

// ReviewSubmission updates submission status after review
func (r *SubmissionRepository) ReviewSubmission(ctx context.Context, id, reviewedBy, status, reviewNotes string, caseID *string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       status,
		"reviewed_by":  reviewedBy,
		"review_notes": reviewNotes,
		"reviewed_at":  now,
	}
	if caseID != nil {
		updates["case_id"] = caseID
	}

	return r.db.WithContext(ctx).
		Model(&models.Submission{}).
		Where("id = ?", id).
		Updates(updates).Error
}
