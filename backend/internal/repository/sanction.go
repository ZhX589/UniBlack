package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var ErrSanctionNotFound = errors.New("sanction not found or already revoked")

type SanctionRepository struct{ db *gorm.DB }

func NewSanctionRepository(db *gorm.DB) *SanctionRepository { return &SanctionRepository{db: db} }

func (r *SanctionRepository) Create(ctx context.Context, sanction *models.Sanction) error {
	return r.db.WithContext(ctx).Create(sanction).Error
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
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Sanction{}).Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]interface{}{"revoked_at": now, "revoked_by": actorID, "revoke_reason": reason})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSanctionNotFound
	}
	return nil
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

func (r *SanctionRepository) CreateAppeal(ctx context.Context, appeal *models.SanctionAppeal) error {
	return r.db.WithContext(ctx).Create(appeal).Error
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
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.SanctionAppeal{}).
		Where("id = ? AND status = ?", appealID, "pending").
		Updates(map[string]interface{}{
			"status": status, "reviewed_by": reviewerID, "review_notes": notes, "reviewed_at": now, "updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("sanction appeal not found or already resolved")
	}
	return nil
}

func (r *SanctionRepository) GetAppealByID(ctx context.Context, id string) (*models.SanctionAppeal, error) {
	var row models.SanctionAppeal
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("sanction appeal not found")
	}
	return &row, err
}
