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
	auth := NewAuthService(nil, nil, failingAccessList{}, nil, nil, nil)
	_, err := auth.Register(context.Background(), RegisterRequest{Username: "new-user", Email: "new@example.com", Password: "password123"}, "192.0.2.1")
	if !errors.Is(err, ErrAccessControlUnavailable) {
		t.Fatalf("Register() error = %v, want %v", err, ErrAccessControlUnavailable)
	}
}

func TestRegisterRejectsInvalidInputBeforeDependencies(t *testing.T) {
	valid := RegisterRequest{Username: "new-user", Email: "new@example.test", Password: "password123"}
	for _, tc := range []struct {
		name   string
		change func(*RegisterRequest)
	}{
		{"blank username", func(r *RegisterRequest) { r.Username = "  " }},
		{"invalid email", func(r *RegisterRequest) { r.Email = "not-an-email" }},
		{"display-name email", func(r *RegisterRequest) { r.Email = "User <new@example.test>" }},
		{"empty password", func(r *RegisterRequest) { r.Password = "" }},
		{"short password", func(r *RegisterRequest) { r.Password = "1234567" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := valid
			tc.change(&req)
			// Nil dependencies ensure validation cannot accidentally reach DB,
			// captcha or verification-code consumption.
			s := NewAuthService(nil, nil, nil, nil, nil, nil)
			if _, err := s.Register(context.Background(), req, "192.0.2.1"); err == nil || errors.Is(err, ErrAccessControlUnavailable) {
				t.Fatalf("input validation error = %v", err)
			}
		})
	}
}

type failingAccessList struct{}

func (failingAccessList) IsListed(context.Context, string, string, string) (bool, error) {
	return false, errors.New("database unavailable")
}
