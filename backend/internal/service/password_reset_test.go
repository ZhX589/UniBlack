package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/mailer"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

type resetTestUserStore struct {
	user  *models.User
	err   error
	calls int
}

func (s *resetTestUserStore) GetUserByEmail(context.Context, string) (*models.User, error) {
	s.calls++
	return s.user, s.err
}

type resetTestStore struct {
	calls     int
	committed bool
}

func (s *resetTestStore) CreateTokenAndDeliver(_ context.Context, _ string, deliver func(string) error) error {
	s.calls++
	if err := deliver("secret-reset-token"); err != nil {
		return err
	}
	s.committed = true
	return nil
}
func (*resetTestStore) ResetPassword(context.Context, string, string) error        { return nil }
func (*resetTestStore) AdminResetPassword(context.Context, string) (string, error) { return "", nil }

type resetTestMailer struct {
	err     error
	message mailer.Message
}

func (m *resetTestMailer) Send(_ context.Context, message mailer.Message) error {
	m.message = message
	return m.err
}

type resetTestSettings struct{ host string }

func (s resetTestSettings) GetSettingValue(_ context.Context, key string, dest interface{}) error {
	if key == "security.smtp_host" {
		*dest.(*string) = s.host
	}
	return nil
}

func TestForgotPasswordProductionRequiresSMTPBeforeCredentials(t *testing.T) {
	for _, env := range []string{"production", "staging"} {
		for _, host := range []string{"", "  "} {
			t.Run(env+"/"+host, func(t *testing.T) {
				t.Setenv("GO_ENV", env)
				users := &resetTestUserStore{user: &models.User{ID: "user", Email: "user@example.test"}}
				store := &resetTestStore{}
				s := NewAuthService(nil, resetTestSettings{host: host}, nil, nil, nil, NewPasswordResetService(store, users))
				if err := s.ForgotPassword(context.Background(), "user@example.test"); !errors.Is(err, ErrSMTPRequired) {
					t.Fatalf("ForgotPassword() = %v, want ErrSMTPRequired", err)
				}
				if users.calls != 0 || store.calls != 0 {
					t.Fatal("unconfigured SMTP reached account/token stores")
				}
			})
		}
	}
}

type resetChangingSettings struct{ reads int }

func (s *resetChangingSettings) GetSettingValue(_ context.Context, key string, dest interface{}) error {
	if key == "security.smtp_host" {
		s.reads++
		if s.reads == 1 {
			*dest.(*string) = "smtp.example.test"
		}
	}
	return nil
}

func TestForgotPasswordProductionRejectsLogMailerAfterConfigChange(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	users := &resetTestUserStore{user: &models.User{ID: "user", Email: "user@example.test"}}
	store := &resetTestStore{}
	s := NewAuthService(nil, &resetChangingSettings{}, nil, nil, nil, NewPasswordResetService(store, users))
	if err := s.ForgotPassword(context.Background(), "user@example.test"); !errors.Is(err, ErrSMTPRequired) {
		t.Fatalf("ForgotPassword() = %v, want ErrSMTPRequired", err)
	}
	if users.calls != 0 || store.calls != 0 {
		t.Fatal("configuration change allowed LogMailer to receive credentials")
	}
}

func TestForgotPasswordDeliveryFailureDoesNotRevealAccount(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	for _, lookupErr := range []error{nil, repository.ErrUserNotFound, errors.New("database unavailable")} {
		users := &resetTestUserStore{user: &models.User{ID: "user", Email: "user@example.test"}, err: lookupErr}
		store := &resetTestStore{}
		// A configured host but no from/username causes the real SMTPMailer to
		// fail before dialing. This is not a LogMailer or a fake success.
		s := NewAuthService(nil, resetTestSettings{host: "smtp.example.test"}, nil, nil, nil, NewPasswordResetService(store, users))
		if err := s.ForgotPassword(context.Background(), "user@example.test"); err != nil {
			t.Fatalf("public response disclosed failure: %v", err)
		}
		if store.committed {
			t.Fatal("failed delivery committed a usable token")
		}
		if lookupErr != nil && store.calls != 0 {
			t.Fatal("failed account lookup created a token")
		}
	}
}

func TestPasswordResetDeliveryPropagatesFailureToTransaction(t *testing.T) {
	failure := errors.New("delivery rejected")
	for _, sendErr := range []error{nil, failure} {
		users := &resetTestUserStore{user: &models.User{ID: "user", Email: "user@example.test"}}
		store := &resetTestStore{}
		m := &resetTestMailer{err: sendErr}
		s := NewPasswordResetService(store, users)
		err := s.SendResetToken(context.Background(), "user@example.test", m)
		if !errors.Is(err, sendErr) {
			t.Fatalf("SendResetToken() = %v, want %v", err, sendErr)
		}
		if store.committed != (sendErr == nil) {
			t.Fatalf("committed = %v", store.committed)
		}
		if len(m.message.To) != 1 || m.message.To[0] != users.user.Email {
			t.Fatalf("recipient = %v", m.message.To)
		}
	}
}
