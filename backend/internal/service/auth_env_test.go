package service

import (
	"context"
	"errors"
	"testing"
)

func TestAppEnvUsesGOEnvOnly(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	t.Setenv("APP_ENV", "development")
	if got := appEnv(); got != "production" {
		t.Fatalf("appEnv() = %q, want production", got)
	}
}

func TestRegisterFailsClosedWhenBlacklistLookupFails(t *testing.T) {
	auth := NewAuthService(nil, nil, failingAccessList{}, nil, nil)
	_, err := auth.Register(context.Background(), RegisterRequest{Username: "new-user", Email: "new@example.com", Password: "password123"}, "192.0.2.1")
	if !errors.Is(err, ErrAccessControlUnavailable) {
		t.Fatalf("Register() error = %v, want %v", err, ErrAccessControlUnavailable)
	}
}

type failingAccessList struct{}

func (failingAccessList) IsListed(context.Context, string, string, string) (bool, error) {
	return false, errors.New("database unavailable")
}
