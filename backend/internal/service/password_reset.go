package service

import (
	"context"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/mailer"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

// PasswordResetStore commits a credential only if its delivery succeeds.
type PasswordResetStore interface {
	CreateTokenAndDeliver(context.Context, string, func(string) error) error
	ResetPassword(context.Context, string, string) error
	AdminResetPassword(context.Context, string) (string, error)
}

type PasswordResetUserStore interface {
	GetUserByEmail(context.Context, string) (*models.User, error)
}

// PasswordResetService handles password recovery flows.
type PasswordResetService struct {
	repo PasswordResetStore
	user PasswordResetUserStore
}

func NewPasswordResetService(repo PasswordResetStore, user PasswordResetUserStore) *PasswordResetService {
	return &PasswordResetService{repo: repo, user: user}
}

// SendResetToken delivers a one-time credential before it becomes usable.
// Unknown accounts silently succeed; the caller must conceal other failures too.
func (s *PasswordResetService) SendResetToken(ctx context.Context, email string, m mailer.Mailer) error {
	user, err := s.user.GetUserByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.repo.CreateTokenAndDeliver(ctx, user.ID, func(token string) error {
		return m.Send(ctx, mailer.Message{
			To: []string{user.Email}, Subject: "密码重置",
			Body: "您的密码重置令牌：" + token + "\n\n请在 1 小时内使用。",
		})
	})
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
