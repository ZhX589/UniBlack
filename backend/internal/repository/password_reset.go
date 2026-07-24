package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var (
	ErrPasswordResetTokenExpired = errors.New("password reset token expired or already used")
	ErrPasswordResetTokenInvalid = errors.New("invalid password reset token")
)

// PasswordResetRepository manages password reset tokens.
type PasswordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// CreateToken creates a one-time reset token for the user. Returns the raw token.
func (r *PasswordResetRepository) CreateToken(ctx context.Context, userID string) (string, error) {
	raw, err := generateToken()
	if err != nil {
		return "", err
	}
	token := &models.PasswordResetToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return "", err
	}
	return raw, nil
}

// ResetPassword validates the token, updates the user's password, and marks the token used.
func (r *PasswordResetRepository) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	hashed := hashToken(rawToken)

	var token models.PasswordResetToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hashed).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPasswordResetTokenInvalid
		}
		return err
	}
	if token.UsedAt != nil || time.Now().After(token.ExpiresAt) {
		return ErrPasswordResetTokenExpired
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&token).Update("used_at", time.Now()).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", token.UserID).Update("password_hash", string(passwordHash)).Error
	})
}

// AdminResetPassword resets a user's password directly (admin tool). Returns the new plain-text password.
func (r *PasswordResetRepository) AdminResetPassword(ctx context.Context, userID string) (string, error) {
	newPassword, err := generateToken()
	if err != nil {
		return "", err
	}
	// Use first 16 hex chars as the temporary password.
	pw := newPassword[:16]

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("password_hash", string(passwordHash)).Error; err != nil {
		return "", err
	}
	return pw, nil
}
