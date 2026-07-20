package repository

import (
	"context"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
)

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
	return r.db.WithContext(ctx).Model(&models.Sanction{}).Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]interface{}{"revoked_at": now, "revoked_by": actorID, "revoke_reason": reason}).Error
}
