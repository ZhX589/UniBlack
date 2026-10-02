package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestPasswordResetDeliveryRollback(t *testing.T) {
	database := setupIntegrationDatabase(t)
	repo := NewPasswordResetRepository(database)
	user := integrationAdmin()
	if err := database.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	failure := errors.New("SMTP rejected delivery")
	var raw string
	err := repo.CreateTokenAndDeliver(context.Background(), user.ID, func(token string) error {
		raw = token
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("delivery error = %v", err)
	}
	var count int64
	if err := database.Model(&models.PasswordResetToken{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed delivery persisted %d tokens", count)
	}
	if err := repo.ResetPassword(context.Background(), raw, "newpassword123"); !errors.Is(err, ErrPasswordResetTokenInvalid) {
		t.Fatalf("undelivered token accepted: %v", err)
	}
}

func TestPasswordResetConcurrentSingleConsumption(t *testing.T) {
	database := setupIntegrationDatabase(t)
	repo := NewPasswordResetRepository(database)
	user := integrationAdmin()
	if err := database.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := repo.CreateTokenAndDeliver(context.Background(), user.ID, func(token string) error { raw = token; return nil }); err != nil {
		t.Fatal(err)
	}
	type result struct {
		password string
		err      error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, password := range []string{"firstpassword123", "secondpassword123"} {
		wg.Add(1)
		go func(password string) {
			defer wg.Done()
			<-start
			results <- result{password, repo.ResetPassword(context.Background(), raw, password)}
		}(password)
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	winningPassword := ""
	for result := range results {
		if result.err == nil {
			winners++
			winningPassword = result.password
		} else if !errors.Is(result.err, ErrPasswordResetTokenExpired) {
			t.Fatalf("unexpected concurrent error: %v", result.err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful consumptions = %d, want 1", winners)
	}
	if err := database.First(user, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.AuthVersion != 1 {
		t.Fatalf("auth version = %d, want one committed increment", user.AuthVersion)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(winningPassword)); err != nil {
		t.Fatalf("winner's password not persisted: %v", err)
	}
	if err := repo.ResetPassword(context.Background(), raw, "thirdpassword123"); !errors.Is(err, ErrPasswordResetTokenExpired) {
		t.Fatalf("replay = %v", err)
	}
}

func TestPasswordResetPasswordFailureRollsBackConsumption(t *testing.T) {
	database := setupIntegrationDatabase(t)
	repo := NewPasswordResetRepository(database)
	user := integrationAdmin()
	if err := database.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := repo.CreateTokenAndDeliver(context.Background(), user.ID, func(token string) error { raw = token; return nil }); err != nil {
		t.Fatal(err)
	}
	// Force a genuine SQL password update failure after token CAS succeeds.
	if err := database.Exec("ALTER TABLE users ADD CONSTRAINT reset_test_password CHECK (password_hash = 'integration-test')").Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.ResetPassword(context.Background(), raw, "newpassword123"); err == nil {
		t.Fatal("expected password update failure")
	}
	var token models.PasswordResetToken
	if err := database.First(&token, "token_hash = ?", hashToken(raw)).Error; err != nil {
		t.Fatal(err)
	}
	if token.UsedAt != nil {
		t.Fatal("password update failure consumed token")
	}
	if err := database.First(user, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.AuthVersion != 0 {
		t.Fatal("failed password update revoked sessions outside transaction")
	}
	if err := database.Exec("ALTER TABLE users DROP CONSTRAINT reset_test_password").Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.ResetPassword(context.Background(), raw, "newpassword123"); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
}

func TestPasswordResetExpiredTokenDoesNotChangePassword(t *testing.T) {
	database := setupIntegrationDatabase(t)
	repo := NewPasswordResetRepository(database)
	user := integrationAdmin()
	if err := database.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.PasswordResetToken{UserID: user.ID, TokenHash: hashToken("expired"), ExpiresAt: time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.ResetPassword(context.Background(), "expired", "newpassword123"); !errors.Is(err, ErrPasswordResetTokenExpired) {
		t.Fatalf("expired token = %v", err)
	}
	if err := database.First(user, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.PasswordHash != "integration-test" {
		t.Fatal("expired token changed password")
	}
}
