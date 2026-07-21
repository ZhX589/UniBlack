package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var ErrAlreadyInitialized = errors.New("system already initialized")

// SetupStore provides the atomic persistence boundary for first-run setup.
type SetupStore interface {
	Initialize(ctx context.Context, admin *models.User, siteName string) (bool, error)
}

type SetupService struct{ store SetupStore }

func NewSetupService(store SetupStore) *SetupService { return &SetupService{store: store} }

func (s *SetupService) Initialize(ctx context.Context, password, siteName string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	created, err := s.store.Initialize(ctx, &models.User{
		Username:      "admin",
		Email:         "admin@uniblack.local",
		PasswordHash:  string(hash),
		AuthProvider:  "local",
		IsActive:      true,
		EmailVerified: true,
	}, siteName)
	if err != nil {
		return err
	}
	if !created {
		return ErrAlreadyInitialized
	}
	return nil
}

// SeedDevelopmentAdmin uses the same atomic setup boundary as user setup.
func (s *SetupService) SeedDevelopmentAdmin(ctx context.Context, password string) error {
	return s.Initialize(ctx, password, "")
}
