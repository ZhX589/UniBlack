package repository

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/db"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/setting"
	"gorm.io/gorm"
)

func TestSetupRepositoryRollsBackAndCreatesOnlyOneAdmin(t *testing.T) {
	database := setupIntegrationDatabase(t)
	ctx := context.Background()
	repo := NewSetupRepository(database)

	if err := database.WithContext(ctx).Exec("UPDATE roles SET name = ? WHERE name = ?", "admin-setup-test", "admin").Error; err != nil {
		t.Fatalf("hide admin role: %v", err)
	}
	created, err := repo.Initialize(ctx, integrationAdmin(), "")
	if err == nil || created {
		t.Fatalf("Initialize() = (%t, %v), want rollback error", created, err)
	}
	assertAdminCount(t, database, 0)
	if err := database.WithContext(ctx).Exec("UPDATE roles SET name = ? WHERE name = ?", "admin", "admin-setup-test").Error; err != nil {
		t.Fatalf("restore admin role: %v", err)
	}

	results := make(chan bool, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := repo.Initialize(ctx, integrationAdmin(), "")
			results <- created
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Initialize() error = %v", err)
		}
	}
	createdCount := 0
	for created := range results {
		if created {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("successful initializations = %d, want 1", createdCount)
	}
	assertAdminCount(t, database, 1)
}

func integrationAdmin() *models.User {
	return &models.User{Username: "admin", Email: "admin@uniblack.local", PasswordHash: "integration-test", AuthProvider: "local", IsActive: true, EmailVerified: true}
}

func setupIntegrationDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for setup integration test")
	}
	base, err := db.Connect(databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	schema := fmt.Sprintf("setup_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = base.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE").Error })

	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	database, err := db.Connect(u.String())
	if err != nil {
		t.Fatalf("connect isolated schema: %v", err)
	}
	_, file, _, _ := runtime.Caller(0)
	t.Setenv("MIGRATIONS_PATH", filepath.Join(filepath.Dir(file), "..", "migrations"))
	if err := db.RunMigrations(database); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}
	// Ensure initialized flag starts false for setup tests.
	if err := database.Exec("UPDATE system_settings SET value = 'false' WHERE key = ?", setting.KeySystemInitialized).Error; err != nil {
		t.Fatalf("reset initialized setting: %v", err)
	}
	return database
}

func assertAdminCount(t *testing.T, database *gorm.DB, want int64) {
	t.Helper()
	var count int64
	if err := database.Raw("SELECT count(*) FROM users WHERE username = ?", "admin").Scan(&count).Error; err != nil {
		t.Fatalf("count admins: %v", err)
	}
	if count != want {
		t.Fatalf("admin count = %d, want %d", count, want)
	}
}
