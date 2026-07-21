package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestSetupServiceReturnsAlreadyInitializedAfterFirstInitialization(t *testing.T) {
	store := &fakeSetupStore{}
	setup := NewSetupService(store)

	if err := setup.Initialize(context.Background(), "password123", "UniBlack"); err != nil {
		t.Fatalf("first Initialize() error = %v", err)
	}
	if err := setup.Initialize(context.Background(), "password123", "UniBlack"); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("second Initialize() error = %v, want %v", err, ErrAlreadyInitialized)
	}
	if store.calls != 2 {
		t.Fatalf("store calls = %d, want 2", store.calls)
	}
}

func TestDevelopmentSeedMakesSetupInitialized(t *testing.T) {
	store := &fakeSetupStore{}
	setup := NewSetupService(store)

	if err := setup.SeedDevelopmentAdmin(context.Background(), "admin123"); err != nil {
		t.Fatalf("SeedDevelopmentAdmin() error = %v", err)
	}
	if err := setup.Initialize(context.Background(), "password123", "UniBlack"); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("Initialize() after seed error = %v, want %v", err, ErrAlreadyInitialized)
	}
}

type fakeSetupStore struct {
	initialized bool
	calls       int
}

func (s *fakeSetupStore) Initialize(_ context.Context, _ *models.User, _ string) (bool, error) {
	s.calls++
	if s.initialized {
		return false, nil
	}
	s.initialized = true
	return true, nil
}
