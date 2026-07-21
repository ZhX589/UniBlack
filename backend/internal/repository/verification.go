package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var ErrVerificationNotFound = errors.New("verification code not found")

// VerificationRepository stores short-lived email codes.
type VerificationRepository struct {
	db *gorm.DB
}

func NewVerificationRepository(db *gorm.DB) *VerificationRepository {
	return &VerificationRepository{db: db}
}

// Create inserts a verification code.
func (r *VerificationRepository) Create(ctx context.Context, code *models.VerificationCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

// Consume validates and marks a code as used if it is still valid.
func (r *VerificationRepository) Consume(ctx context.Context, email, purpose, code string) error {
	var row models.VerificationCode
	err := r.db.WithContext(ctx).
		Where("email = ? AND purpose = ? AND code = ? AND used_at IS NULL AND expires_at > ?",
			email, purpose, code, time.Now()).
		Order("created_at DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVerificationNotFound
		}
		return err
	}
	now := time.Now()
	return r.db.WithContext(ctx).Model(&row).Update("used_at", now).Error
}

// InvalidateEmail marks prior unused codes for email+purpose as used.
func (r *VerificationRepository) InvalidateEmail(ctx context.Context, email, purpose string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.VerificationCode{}).
		Where("email = ? AND purpose = ? AND used_at IS NULL", email, purpose).
		Update("used_at", now).Error
}

// LatestCreatedAt returns the created_at of the newest code for email+purpose, if any.
func (r *VerificationRepository) LatestCreatedAt(ctx context.Context, email, purpose string) (time.Time, bool, error) {
	var row models.VerificationCode
	err := r.db.WithContext(ctx).
		Select("created_at").
		Where("email = ? AND purpose = ?", email, purpose).
		Order("created_at DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return row.CreatedAt, true, nil
}
