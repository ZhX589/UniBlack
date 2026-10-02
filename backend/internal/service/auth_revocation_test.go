package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/auth"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (s *sessionUserStore) GetUserByUsername(context.Context, string) (*models.User, error) {
	return s.user, s.err
}
func (*sessionUserStore) UpdateLastLogin(context.Context, string) error { return nil }

func TestPasswordVersionRevokesBothCredentialTypesAndNewLoginWorks(t *testing.T) {
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionUserStore{user: &models.User{ID: "user-1", Username: "user", Email: "user@example.test", IsActive: true, PasswordHash: string(hash)}}
	provider := auth.NewJWTProvider(auth.JWTConfig{Secret: "access", RefreshSecret: "refresh"})
	svc := NewAuthService(store, nil, nil, nil, provider, nil)
	old, err := svc.Login(ctx, LoginRequest{Username: "user", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	store.user.AuthVersion++ // password reset and admin reset increment in the same SQL update
	if _, err := svc.ValidateToken(ctx, old.AccessToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("old access error=%v", err)
	}
	if _, err := svc.RefreshToken(ctx, old.RefreshToken); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("old refresh error=%v", err)
	}
	current, err := svc.Login(ctx, LoginRequest{Username: "user", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := svc.ValidateToken(ctx, current.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if identity.AuthVersion != 1 {
		t.Fatalf("version=%d", identity.AuthVersion)
	}
	refreshed, err := svc.RefreshToken(ctx, current.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ValidateToken(ctx, refreshed.AccessToken); err != nil {
		t.Fatal(err)
	}
}
