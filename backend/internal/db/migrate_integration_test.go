package db

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"gorm.io/gorm"
)

func TestRunMigrations(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for migration integration test")
	}
	base, err := Connect(databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	schema := fmt.Sprintf("migration_%d", time.Now().UnixNano())
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
	database, err := Connect(u.String())
	if err != nil {
		t.Fatalf("connect isolated schema: %v", err)
	}
	_, file, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(file), "..", "migrations")
	t.Setenv("MIGRATIONS_PATH", migrationsPath)
	if err := RunMigrations(database); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	userID := insertReturningID(t, database, "INSERT INTO users (username, email, password_hash) VALUES ('migration-user', 'migration@example.test', 'test') RETURNING id")
	subjectID := insertReturningID(t, database, "INSERT INTO subjects (public_id, display_name, status, created_by) VALUES ('UBS_MIGRATION_TEST', 'migration', 'active', ?) RETURNING id", userID)
	eventID := insertReturningID(t, database, "INSERT INTO events (subject_id, title, details, status, submitted_by) VALUES (?, 'event', 'details', 'published', ?) RETURNING id", subjectID, userID)
	appealID := insertReturningID(t, database, "INSERT INTO appeals (event_id, reason, status, submitted_by) VALUES (?, 'event-only', 'pending', ?) RETURNING id", eventID, userID)

	m := migrationInstance(t, database, migrationsPath)
	err = m.Steps(-1)
	if err == nil || !strings.Contains(err.Error(), "cannot downgrade event governance while event-only appeals exist") {
		t.Fatalf("event-only downgrade error = %v", err)
	}
	assertColumnNullable(t, database, "appeals", "case_id", "YES")
	assertColumnExists(t, database, "appeals", "deleted_at")
	assertConstraintExists(t, database, "appeals_target_check")
	if err := m.Force(11); err != nil {
		t.Fatalf("restore migration version after expected failure: %v", err)
	}
	if err := database.Exec("DELETE FROM appeals WHERE id = ?", appealID).Error; err != nil {
		t.Fatalf("remove incompatible appeal: %v", err)
	}
	if err := m.Steps(-1); err != nil {
		t.Fatalf("clean migration 11 down: %v", err)
	}
	assertColumnNullable(t, database, "appeals", "case_id", "NO")
	assertColumnMissing(t, database, "appeals", "deleted_at")
	assertColumnMissing(t, database, "submissions", "deleted_at")
	if err := m.Steps(1); err != nil {
		t.Fatalf("migration 11 re-up: %v", err)
	}
	assertColumnNullable(t, database, "appeals", "case_id", "YES")
	assertColumnExists(t, database, "appeals", "deleted_at")
	assertConstraintExists(t, database, "appeals_target_check")
}

func migrationInstance(t *testing.T, database *gorm.DB, migrationsPath string) *migrate.Migrate {
	t.Helper()
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("underlying database: %v", err)
	}
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		t.Fatalf("migration driver: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "postgres", driver)
	if err != nil {
		t.Fatalf("migration instance: %v", err)
	}
	return m
}

func insertReturningID(t *testing.T, database *gorm.DB, statement string, args ...interface{}) string {
	t.Helper()
	var id string
	if err := database.Raw(statement, args...).Scan(&id).Error; err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
	return id
}

func assertColumnNullable(t *testing.T, database *gorm.DB, table, column, want string) {
	t.Helper()
	var got string
	if err := database.Raw("SELECT is_nullable FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?", table, column).Scan(&got).Error; err != nil {
		t.Fatalf("query column %s.%s: %v", table, column, err)
	}
	if got != want {
		t.Fatalf("%s.%s nullable = %q, want %q", table, column, got, want)
	}
}

func assertColumnMissing(t *testing.T, database *gorm.DB, table, column string) {
	t.Helper()
	var count int64
	if err := database.Raw("SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?", table, column).Scan(&count).Error; err != nil {
		t.Fatalf("query column %s.%s: %v", table, column, err)
	}
	if count != 0 {
		t.Fatalf("column %s.%s still exists", table, column)
	}
}

func assertColumnExists(t *testing.T, database *gorm.DB, table, column string) {
	t.Helper()
	var count int64
	if err := database.Raw("SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?", table, column).Scan(&count).Error; err != nil {
		t.Fatalf("query column %s.%s: %v", table, column, err)
	}
	if count == 0 {
		t.Fatalf("column %s.%s missing", table, column)
	}
}

func assertConstraintExists(t *testing.T, database *gorm.DB, name string) {
	t.Helper()
	var count int64
	if err := database.Raw("SELECT count(*) FROM information_schema.table_constraints WHERE table_schema = current_schema() AND constraint_name = ?", name).Scan(&count).Error; err != nil {
		t.Fatalf("query constraint %s: %v", name, err)
	}
	if count == 0 {
		t.Fatalf("constraint %s missing", name)
	}
}
