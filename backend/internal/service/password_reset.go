package service

import (
	"context"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

// PasswordResetService handles password recovery flows.
type PasswordResetService struct {
	repo *repository.PasswordResetRepository
	user *repository.UserRepository
}

func NewPasswordResetService(
	repo *repository.PasswordResetRepository,
	user *repository.UserRepository,
) *PasswordResetService {
	return &PasswordResetService{repo: repo, user: user}
}

// CreateResetToken finds the user by email and creates a one-time reset token.
// Returns the raw token string. Returns nil if no user found (don't reveal existence).
func (s *PasswordResetService) CreateResetToken(ctx context.Context, email string) (string, *string, error) {
	user, err := s.user.GetUserByEmail(ctx, email)
	if err != nil {
		return "", nil, nil // don't reveal whether email exists
	}

	token, err := s.repo.CreateToken(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}
	return token, &user.Email, nil
}

// ResetPassword validates the token and sets a new password.
func (s *PasswordResetService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return s.repo.ResetPassword(ctx, rawToken, newPassword)
}

// AdminResetPassword directly resets a user's password (admin action).
func (s *PasswordResetService) AdminResetPassword(ctx context.Context, userID string) (string, error) {
	return s.repo.AdminResetPassword(ctx, userID)
}
