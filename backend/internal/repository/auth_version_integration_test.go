package repository

import (
	"context"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func TestAdminResetVersionAndStaleProfileCannotRestoreCredentials(t *testing.T) {
	db := setupIntegrationDatabase(t)
	ctx := context.Background()
	user := integrationAdmin()
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	stale := *user
	pw, err := NewPasswordResetRepository(db).AdminResetPassword(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	stale.Username = "changed-profile"
	if err := NewUserRepository(db).UpdateUser(ctx, &stale); err != nil {
		t.Fatal(err)
	}
	var current models.User
	if err := db.First(&current, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.AuthVersion != 1 {
		t.Fatalf("profile restored auth version %d", current.AuthVersion)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(pw)); err != nil {
		t.Fatal("profile restored previous password")
	}
	if current.Username != stale.Username {
		t.Fatal("profile change did not persist")
	}
}
