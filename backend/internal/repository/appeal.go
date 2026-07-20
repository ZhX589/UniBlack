package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrAppealNotFound = errors.New("appeal not found")
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

// ListAppeals lists appeals with pagination
func (r *AppealRepository) ListAppeals(ctx context.Context, offset, limit int, status string, submittedBy string) ([]models.Appeal, int64, error) {
	var appeals []models.Appeal
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Appeal{})
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

// ReviewAppeal updates appeal status after review
func (r *AppealRepository) ReviewAppeal(ctx context.Context, id, reviewedBy, status, reviewNotes string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       status,
		"reviewed_by":  reviewedBy,
		"review_notes": reviewNotes,
		"reviewed_at":  now,
	}

	return r.db.WithContext(ctx).
		Model(&models.Appeal{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// DeleteAppeal deletes an appeal
func (r *AppealRepository) DeleteAppeal(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Appeal{})
	if result.RowsAffected == 0 {
		return ErrAppealNotFound
	}
	return result.Error
}
