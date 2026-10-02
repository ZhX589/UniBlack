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
	"gorm.io/gorm/clause"

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

// CreateTokenAndDeliver commits a one-time token only after successful delivery.
// Mail cannot be atomically committed with SQL: a commit failure after successful
// delivery may leave an unusable emailed token; a delivery error rolls SQL back.
// Transport errors after SMTP acceptance can be ambiguous: this is not an
// exactly-once email guarantee.
func (r *PasswordResetRepository) CreateTokenAndDeliver(ctx context.Context, userID string, deliver func(string) error) error {
	raw, err := generateToken()
	if err != nil {
		return err
	}
	token := &models.PasswordResetToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(token).Error; err != nil {
			return err
		}
		return deliver(raw)
	})
}

// ResetPassword validates the token, updates the user's password, and marks the token used.
func (r *PasswordResetRepository) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	// Hash before taking locks so bcrypt does not extend the critical section.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token models.PasswordResetToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", hashToken(rawToken)).First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPasswordResetTokenInvalid
			}
			return err
		}
		now := time.Now()
		if token.UsedAt != nil || !token.ExpiresAt.After(now) {
			return ErrPasswordResetTokenExpired
		}
		consumed := tx.Model(&token).Where("used_at IS NULL AND expires_at > ?", now).Update("used_at", now)
		if consumed.Error != nil {
			return consumed.Error
		}
		if consumed.RowsAffected != 1 {
			return ErrPasswordResetTokenExpired
		}
		updated := tx.Model(&models.User{}).Where("id = ?", token.UserID).Updates(map[string]interface{}{"password_hash": string(passwordHash), "auth_version": gorm.Expr("auth_version + 1")})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrUserNotFound // Roll back consumption if the account is missing/deleted.
		}
		return nil
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

	result := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{"password_hash": string(passwordHash), "auth_version": gorm.Expr("auth_version + 1")})
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected != 1 {
		return "", ErrUserNotFound
	}
	return pw, nil
}
