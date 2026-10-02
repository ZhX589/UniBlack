package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/auth"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

type sessionUserStore struct {
	AuthUserStore
	user  *models.User
	err   error
	reads int
}

func (s *sessionUserStore) GetUserByID(context.Context, string) (*models.User, error) {
	s.reads++
	return s.user, s.err
}

func TestSessionReloadsCurrentIdentity(t *testing.T) {
	ctx := context.Background()
	provider := auth.NewJWTProvider(auth.JWTConfig{Secret: "access-secret", RefreshSecret: "refresh-secret"})
	original := &auth.SubjectIdentity{UserID: "user-1", Username: "old-name", Email: "old@example.test", Roles: []string{"admin"}}
	tokens, err := provider.GenerateTokens(ctx, original)
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionUserStore{user: &models.User{ID: "user-1", Username: "current-name", Email: "current@example.test", IsActive: true, Roles: []models.Role{{Name: "user"}}}}
	svc := &AuthService{userRepo: store, provider: provider}
	identity, err := svc.ValidateToken(ctx, tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Username != store.user.Username || identity.Email != store.user.Email || !reflect.DeepEqual(identity.Roles, []string{"user"}) {
		t.Fatalf("stale identity: %+v", identity)
	}
	renewed, err := svc.RefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	identity, err = provider.ValidateToken(ctx, renewed.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Username != store.user.Username || identity.Email != store.user.Email || !reflect.DeepEqual(identity.Roles, []string{"user"}) {
		t.Fatalf("incomplete refreshed identity: %+v", identity)
	}
	if store.reads != 2 {
		t.Fatalf("account reads = %d, want 2", store.reads)
	}
}

func TestSessionRejectsInactiveMissingOrUnavailableAccounts(t *testing.T) {
	ctx := context.Background()
	provider := auth.NewJWTProvider(auth.JWTConfig{Secret: "access-secret", RefreshSecret: "refresh-secret"})
	tokens, err := provider.GenerateTokens(ctx, &auth.SubjectIdentity{UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		user *models.User
		err  error
	}{
		{name: "inactive", user: &models.User{ID: "user-1", IsActive: false}},
		{name: "missing", err: repository.ErrUserNotFound},
		{name: "database failure", err: errors.New("database unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &AuthService{provider: provider, userRepo: &sessionUserStore{user: tc.user, err: tc.err}}
			if _, err := svc.ValidateToken(ctx, tokens.AccessToken); err == nil {
				t.Fatal("access accepted")
			}
			if _, err := svc.RefreshToken(ctx, tokens.RefreshToken); err == nil {
				t.Fatal("refresh accepted")
			}
		})
	}
}

func TestSessionInvalidCredentialsDoNotReadAccount(t *testing.T) {
	store := &sessionUserStore{}
	svc := &AuthService{provider: auth.NewJWTProvider(auth.JWTConfig{Secret: "access", RefreshSecret: "refresh"}), userRepo: store}
	if _, err := svc.ValidateToken(context.Background(), "invalid"); err == nil {
		t.Fatal("invalid access accepted")
	}
	if _, err := svc.RefreshToken(context.Background(), "invalid"); err == nil {
		t.Fatal("invalid refresh accepted")
	}
	if store.reads != 0 {
		t.Fatalf("invalid credentials performed %d account reads", store.reads)
	}
}
