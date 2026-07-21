package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
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
