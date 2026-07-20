package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
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

	query := r.db.WithContext(ctx).Model(&models.Submission{})
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

// DeleteSubmission deletes a submission
func (r *SubmissionRepository) DeleteSubmission(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Submission{})
	if result.RowsAffected == 0 {
		return ErrSubmissionNotFound
	}
	return result.Error
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
