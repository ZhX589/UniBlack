package service

import (
	"context"
	"testing"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

type memVerifyStore struct {
	rows []models.VerificationCode
}

func (m *memVerifyStore) Create(_ context.Context, code *models.VerificationCode) error {
	cp := *code
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	m.rows = append(m.rows, cp)
	return nil
}

func (m *memVerifyStore) Consume(context.Context, string, string, string) error {
	return nil
}

func (m *memVerifyStore) InvalidateEmail(_ context.Context, email, purpose string) error {
	now := time.Now()
	for i := range m.rows {
		if m.rows[i].Email == email && m.rows[i].Purpose == purpose && m.rows[i].UsedAt == nil {
			m.rows[i].UsedAt = &now
		}
	}
	return nil
}

func (m *memVerifyStore) LatestCreatedAt(_ context.Context, email, purpose string) (time.Time, bool, error) {
	var latest time.Time
	found := false
	for _, row := range m.rows {
		if row.Email == email && row.Purpose == purpose {
			if !found || row.CreatedAt.After(latest) {
				latest = row.CreatedAt
				found = true
			}
		}
	}
	return latest, found, nil
}

func TestSendVerificationCodeRateLimited(t *testing.T) {
	t.Setenv("GO_ENV", "development")
	store := &memVerifyStore{}
	svc := &AuthService{
		verifyRepo: store,
		settings:   fixedSettings{emailOn: true},
	}
	ctx := context.Background()
	if err := svc.SendVerificationCodeForPurpose(ctx, "user@example.com", "register"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if err := svc.SendVerificationCodeForPurpose(ctx, "user@example.com", "register"); err != ErrVerificationRateLimited {
		t.Fatalf("second send want rate limited, got %v", err)
	}
}

func TestSendVerificationCodeAllowsAfterInterval(t *testing.T) {
	t.Setenv("GO_ENV", "development")
	store := &memVerifyStore{
		rows: []models.VerificationCode{{
			Email:     "user@example.com",
			Code:      "old",
			Purpose:   "register",
			ExpiresAt: time.Now().Add(time.Minute),
			CreatedAt: time.Now().Add(-2 * time.Minute),
		}},
	}
	svc := &AuthService{
		verifyRepo: store,
		settings:   fixedSettings{emailOn: true},
	}
	if err := svc.SendVerificationCodeForPurpose(context.Background(), "user@example.com", "register"); err != nil {
		t.Fatalf("send after interval: %v", err)
	}
}

type fixedSettings struct {
	emailOn bool
}

func (f fixedSettings) GetSettingValue(_ context.Context, key string, dest interface{}) error {
	if key == "security.email_verification" {
		if p, ok := dest.(*bool); ok {
			*p = f.emailOn
		}
	}
	return nil
}
