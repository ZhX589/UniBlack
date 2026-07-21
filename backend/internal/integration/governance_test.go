package integration_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/db"
	exporter "github.com/ZhX589/UniBlack/backend/internal/export"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
	"gorm.io/gorm"
)

func TestEventGovernanceIntegration(t *testing.T) {
	database := integrationDatabase(t)

	t.Run("archive round trips link evidence without a body or hash", func(t *testing.T) {
		resetDatabase(t, database)
		ctx := context.Background()
		publisher := createUser(t, database, "archive-publisher")
		store := storage.NewLocalStorage(t.TempDir(), "/uploads")
		subjectRepo := repository.NewSubjectRepository(database)
		eventRepo := repository.NewEventRepository(database, store)
		evidenceRepo := repository.NewEvidenceRepository(database)
		eventService := service.NewEventService(eventRepo, subjectRepo, nil, nil, nil)
		want := service.PublishLinkEvidenceRequest{EventIndex: 0, Title: "Original report", Description: "Publisher supplied context", URL: "https://example.test/reports/42"}
		subject, err := eventService.Publish(ctx, service.PublishSubjectRequest{
			DisplayName:  "Archive Subject",
			Accounts:     []service.PublishAccountRequest{{Platform: "discord", Username: "archive-user", IsPrimary: true}},
			Events:       []service.PublishEventRequest{{Title: "Published event", Details: "Event details", Severity: 2}},
			LinkEvidence: []service.PublishLinkEvidenceRequest{want},
		}, publisher.ID)
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		archiveService := exporter.NewArchiveService(subjectRepo, eventRepo, evidenceRepo, store)
		archive, err := archiveService.Build(ctx, subject.PublicID)
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		assertArchiveHasNoEvidenceBody(t, archive)
		assertExportedLinkHasNoHash(t, archive, want)

		target := integrationDatabase(t)
		targetStore := storage.NewLocalStorage(t.TempDir(), "/target-uploads")
		targetSubjects := repository.NewSubjectRepository(target)
		targetEvents := repository.NewEventRepository(target, targetStore)
		targetEvidence := repository.NewEvidenceRepository(target)
		targetArchiveService := exporter.NewArchiveService(targetSubjects, targetEvents, targetEvidence, targetStore)
		targetPublisher := createUser(t, target, "archive-importer")
		preview, err := targetArchiveService.PreviewImport(bytes.NewReader(archive))
		if err != nil || !preview.Valid || len(preview.Conflicts) != 0 {
			t.Fatalf("PreviewImport() = %#v, %v", preview, err)
		}
		imported, err := targetArchiveService.Import(ctx, bytes.NewReader(archive), targetPublisher.ID)
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}
		importedEvents, err := targetEvents.ListBySubject(ctx, imported.ID)
		if err != nil || len(importedEvents) != 1 {
			t.Fatalf("imported events = %#v, %v", importedEvents, err)
		}
		links, err := targetEvidence.GetEvidenceByEventID(ctx, importedEvents[0].ID)
		if err != nil || len(links) != 1 {
			t.Fatalf("imported evidence = %#v, %v", links, err)
		}
		link := links[0]
		if link.Type != "link" || value(link.Title) != want.Title || value(link.Description) != want.Description || value(link.URL) != want.URL {
			t.Fatalf("link metadata changed: %#v", link)
		}
		if link.StorageKey != nil || link.SHA256 != nil || link.FileSize != nil || link.MimeType != nil {
			t.Fatalf("link unexpectedly has stored body metadata: %#v", link)
		}
	})

	t.Run("malicious adjudication commits together and warning permits publishing", func(t *testing.T) {
		resetDatabase(t, database)
		ctx := context.Background()
		submitter := createUser(t, database, "malicious-submitter")
		reviewer := createUser(t, database, "malicious-reviewer")
		subject, event := createSubjectEvent(t, database, submitter.ID, "malicious")
		appeal := &models.Appeal{EventID: &event.ID, Reason: "appeal", Status: "pending", SubmittedBy: &submitter.ID}
		if err := database.Create(appeal).Error; err != nil {
			t.Fatalf("create appeal: %v", err)
		}
		repo := repository.NewAppealRepository(database)
		if err := repo.ResolveWithConsequences(ctx, appeal.ID, reviewer.ID, "rejected", "malicious_submission", "abusive appeal"); err != nil {
			t.Fatalf("ResolveWithConsequences() error = %v", err)
		}
		var got models.Appeal
		if err := database.First(&got, "id = ?", appeal.ID).Error; err != nil || value(got.Outcome) != "malicious_submission" {
			t.Fatalf("resolved appeal = %#v, %v", got, err)
		}
		assertCount(t, database, &models.Sanction{}, "related_appeal_id = ? AND type = 'warning'", appeal.ID, 1)
		assertCount(t, database, &models.AuditLog{}, "resource_type = 'appeal' AND resource_id = ? AND action = 'resolve'", appeal.ID, 1)
		assertCount(t, database, &models.AuditLog{}, "resource_type = 'sanction' AND action = 'create'", nil, 1)

		eventService := service.NewEventService(repository.NewEventRepository(database), repository.NewSubjectRepository(database), repository.NewSanctionRepository(database), nil, nil)
		published, err := eventService.Publish(ctx, service.PublishSubjectRequest{
			DisplayName: "Warning allowed",
			Accounts:    []service.PublishAccountRequest{{Platform: "telegram", Username: "warning-allowed"}},
			Events:      []service.PublishEventRequest{{Title: "Allowed event", Details: "Warnings are non-blocking"}},
		}, submitter.ID)
		if err != nil || published.ID == subject.ID {
			t.Fatalf("Publish() after warning = %#v, %v", published, err)
		}
	})

	t.Run("malicious adjudication rolls back every consequence on late failure", func(t *testing.T) {
		resetDatabase(t, database)
		ctx := context.Background()
		submitter := createUser(t, database, "rollback-submitter")
		reviewer := createUser(t, database, "rollback-reviewer")
		_, event := createSubjectEvent(t, database, submitter.ID, "rollback")
		appeal := &models.Appeal{EventID: &event.ID, Reason: "appeal", Status: "pending", SubmittedBy: &submitter.ID}
		if err := database.Create(appeal).Error; err != nil {
			t.Fatalf("create appeal: %v", err)
		}
		trigger := `CREATE FUNCTION fail_sanction_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.resource_type = 'sanction' THEN RAISE EXCEPTION 'induced sanction audit failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_sanction_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION fail_sanction_audit()`
		if err := database.Exec(trigger).Error; err != nil {
			t.Fatalf("create failure trigger: %v", err)
		}
		err := repository.NewAppealRepository(database).ResolveWithConsequences(ctx, appeal.ID, reviewer.ID, "rejected", "malicious_submission", "must roll back")
		if err == nil || !strings.Contains(err.Error(), "induced sanction audit failure") {
			t.Fatalf("ResolveWithConsequences() error = %v", err)
		}
		var got models.Appeal
		if err := database.First(&got, "id = ?", appeal.ID).Error; err != nil || got.Status != "pending" || got.Outcome != nil || got.ReviewedAt != nil {
			t.Fatalf("appeal was not rolled back: %#v, %v", got, err)
		}
		assertCount(t, database, &models.Sanction{}, "related_appeal_id = ?", appeal.ID, 0)
		assertCount(t, database, &models.AuditLog{}, "resource_type = 'appeal' AND resource_id = ?", appeal.ID, 0)
		assertCount(t, database, &models.AuditLog{}, "resource_type = 'sanction'", nil, 0)
	})

	t.Run("submission and appeal retirement hides active reads but preserves audit history", func(t *testing.T) {
		resetDatabase(t, database)
		ctx := context.Background()
		actor := createUser(t, database, "retirement-actor")
		_, event := createSubjectEvent(t, database, actor.ID, "retirement")
		auditRepo := repository.NewAuditLogRepository(database)
		submissionRepo := repository.NewSubmissionRepository(database)
		submission := &models.Submission{SubjectIdentifiers: `[]`, Reason: "retire me", Status: "pending", SubmittedBy: &actor.ID}
		if err := submissionRepo.CreateSubmission(ctx, submission); err != nil {
			t.Fatalf("create submission: %v", err)
		}
		submissionService := service.NewSubmissionService(submissionRepo, repository.NewSubjectRepository(database), repository.NewCaseRepository(database), auditRepo)
		if err := submissionService.DeleteSubmission(ctx, submission.ID, actor.ID); err != nil {
			t.Fatalf("DeleteSubmission() error = %v", err)
		}
		retiredSubmission, err := submissionService.GetSubmission(ctx, submission.ID)
		if err != nil || retiredSubmission.DeletedAt == nil {
			t.Fatalf("GetSubmission() = %#v, %v", retiredSubmission, err)
		}
		rows, total, err := submissionService.ListSubmissions(ctx, 1, 20, "", "")
		if err != nil || total != 0 || len(rows) != 0 {
			t.Fatalf("ListSubmissions() = %#v, %d, %v", rows, total, err)
		}
		assertRetiredAndAudited(t, database, auditRepo, "submissions", "submission", submission.ID)

		appealRepo := repository.NewAppealRepository(database)
		appeal := &models.Appeal{EventID: &event.ID, Reason: "retire appeal", Status: "pending", SubmittedBy: &actor.ID}
		if err := appealRepo.CreateAppeal(ctx, appeal); err != nil {
			t.Fatalf("create appeal: %v", err)
		}
		appealService := service.NewAppealService(appealRepo, repository.NewCaseRepository(database), repository.NewEventRepository(database), auditRepo)
		if err := appealService.DeleteAppeal(ctx, appeal.ID, actor.ID); err != nil {
			t.Fatalf("DeleteAppeal() error = %v", err)
		}
		retiredAppeal, err := appealService.GetAppeal(ctx, appeal.ID)
		if err != nil || retiredAppeal.DeletedAt == nil {
			t.Fatalf("GetAppeal() = %#v, %v", retiredAppeal, err)
		}
		appeals, total, err := appealService.ListAppeals(ctx, 1, 20, "", "")
		if err != nil || total != 0 || len(appeals) != 0 {
			t.Fatalf("ListAppeals() = %#v, %d, %v", appeals, total, err)
		}
		history, err := appealService.GetAppealsByEventID(ctx, event.ID)
		if err != nil || len(history) != 1 || history[0].ID != appeal.ID || history[0].DeletedAt == nil {
			t.Fatalf("GetAppealsByEventID() = %#v, %v", history, err)
		}
		assertRetiredAndAudited(t, database, auditRepo, "appeals", "appeal", appeal.ID)
	})

	t.Run("account lookup and search precede identifier fallback and statistics use events", func(t *testing.T) {
		resetDatabase(t, database)
		ctx := context.Background()
		actor := createUser(t, database, "lookup-actor")
		accountSubject, accountEvent := createSubjectEvent(t, database, actor.ID, "shared-token")
		accountID := "Account-4242"
		if err := database.Model(&models.Account{}).Where("subject_id = ?", accountSubject.ID).Updates(map[string]interface{}{"platform": "Discord", "username": "Shared-Token", "account_id": accountID}).Error; err != nil {
			t.Fatalf("update account: %v", err)
		}
		fallback := &models.Subject{PublicID: "UBS_IDENTIFIER_FALLBACK", DisplayName: "Identifier fallback", Status: "active", CreatedBy: &actor.ID}
		if err := database.Create(fallback).Error; err != nil {
			t.Fatalf("create fallback subject: %v", err)
		}
		identifier := &models.Identifier{SubjectID: fallback.ID, Platform: "discord", AccountType: "username", Value: "shared-token", IsPrimary: true}
		if err := database.Create(identifier).Error; err != nil {
			t.Fatalf("create fallback identifier: %v", err)
		}
		inactive := &models.Subject{PublicID: "UBS_INACTIVE_STATS", DisplayName: "Inactive", Status: "archived", CreatedBy: &actor.ID}
		if err := database.Create(inactive).Error; err != nil {
			t.Fatalf("create inactive subject: %v", err)
		}
		if err := database.Create(&models.Event{SubjectID: fallback.ID, Title: "Corrected", Details: "not published", Status: "corrected", Severity: 1, SubmittedBy: &actor.ID}).Error; err != nil {
			t.Fatalf("create corrected event: %v", err)
		}

		subjectService := service.NewSubjectService(repository.NewSubjectRepository(database))
		got, err := subjectService.GetSubjectByIdentifier(ctx, " DISCORD ", " account-4242 ")
		if err != nil || got.ID != accountSubject.ID {
			t.Fatalf("account ID lookup = %#v, %v", got, err)
		}
		got, err = subjectService.GetSubjectByIdentifier(ctx, "discord", "shared-token")
		if err != nil || got.ID != accountSubject.ID {
			t.Fatalf("account-first collision lookup = %#v, %v", got, err)
		}
		results, err := subjectService.SearchSubjects(ctx, "shared-token")
		if err != nil || len(results) < 2 || results[0].ID != accountSubject.ID {
			t.Fatalf("account-first search = %#v, %v", results, err)
		}
		identifier.Value = "identifier-only"
		if err := database.Save(identifier).Error; err != nil {
			t.Fatalf("prepare identifier fallback: %v", err)
		}
		got, err = subjectService.GetSubjectByIdentifier(ctx, "discord", "identifier-only")
		if err != nil || got.ID != fallback.ID {
			t.Fatalf("identifier fallback lookup = %#v, %v", got, err)
		}
		subjects, events, err := subjectService.PublicStatistics(ctx)
		if err != nil || subjects != 2 || events != 2 {
			t.Fatalf("PublicStatistics() = (%d, %d, %v), want (2, 2, nil); published event %s", subjects, events, err, accountEvent.ID)
		}
	})
}

func integrationDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for governance integration tests")
	}
	base, err := db.Connect(databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	schema := fmt.Sprintf("governance_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() { _ = base.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE").Error })
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	database, err := db.Connect(u.String())
	if err != nil {
		t.Fatalf("connect isolated schema: %v", err)
	}
	_, file, _, _ := runtime.Caller(0)
	t.Setenv("MIGRATIONS_PATH", filepath.Join(filepath.Dir(file), "..", "migrations"))
	if err := db.RunMigrations(database); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}
	return database
}

func resetDatabase(t *testing.T, database *gorm.DB) {
	t.Helper()
	if err := database.Exec("DROP TRIGGER IF EXISTS fail_sanction_audit ON audit_logs; DROP FUNCTION IF EXISTS fail_sanction_audit(); TRUNCATE audit_logs, sanction_appeals, sanctions, appeals, submissions, evidence, events, accounts, identifiers, cases, subjects, user_roles, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("reset database: %v", err)
	}
}

func createUser(t *testing.T, database *gorm.DB, name string) models.User {
	t.Helper()
	user := models.User{Username: name, Email: name + "@example.test", PasswordHash: "test", AuthProvider: "local", IsActive: true}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func createSubjectEvent(t *testing.T, database *gorm.DB, actorID, token string) (models.Subject, models.Event) {
	t.Helper()
	subject := models.Subject{PublicID: "UBS_" + strings.ToUpper(strings.ReplaceAll(token, "-", "_")), DisplayName: token, Status: "active", CreatedBy: &actorID}
	if err := database.Create(&subject).Error; err != nil {
		t.Fatalf("create subject: %v", err)
	}
	username := token
	if err := database.Create(&models.Account{SubjectID: subject.ID, Platform: "discord", AccountType: "username", Username: &username, CustomAttributes: map[string]interface{}{}}).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	event := models.Event{SubjectID: subject.ID, Title: token, Details: "details", Status: "published", Severity: 1, SubmittedBy: &actorID}
	if err := database.Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}
	return subject, event
}

func assertArchiveHasNoEvidenceBody(t *testing.T, archive []byte) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	for _, file := range zr.File {
		if file.Name != "manifest.json" && file.Name != "README.txt" {
			t.Fatalf("link archive contains stored body %q", file.Name)
		}
	}
}

func assertExportedLinkHasNoHash(t *testing.T, archive []byte, want service.PublishLinkEvidenceRequest) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	var manifest exporter.Manifest
	for _, file := range zr.File {
		if file.Name != "manifest.json" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open manifest: %v", err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("decode manifest: %v", err)
		}
	}
	if len(manifest.Events) != 1 || len(manifest.Events[0].Evidence) != 1 {
		t.Fatalf("unexpected exported evidence: %#v", manifest.Events)
	}
	link := manifest.Events[0].Evidence[0]
	if link.Type != "link" || link.Title != want.Title || link.Description != want.Description || link.URL != want.URL {
		t.Fatalf("exported link metadata = %#v", link)
	}
	if link.SHA256 != "" || link.FileName != "" || link.OriginalFilename != "" {
		t.Fatalf("exported link retained body metadata: %#v", link)
	}
}

func assertCount(t *testing.T, database *gorm.DB, model interface{}, where string, arg interface{}, want int64) {
	t.Helper()
	var got int64
	query := database.Model(model)
	if arg == nil {
		query = query.Where(where)
	} else {
		query = query.Where(where, arg)
	}
	if err := query.Count(&got).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if got != want {
		t.Fatalf("count %T = %d, want %d", model, got, want)
	}
}

func assertRetiredAndAudited(t *testing.T, database *gorm.DB, auditRepo *repository.AuditLogRepository, table, resourceType, id string) {
	t.Helper()
	var retired bool
	if err := database.Raw("SELECT deleted_at IS NOT NULL FROM "+table+" WHERE id = ?", id).Scan(&retired).Error; err != nil || !retired {
		t.Fatalf("%s retirement = %t, %v", table, retired, err)
	}
	logs, err := auditRepo.GetAuditLogsByResource(context.Background(), resourceType, id)
	if err != nil {
		t.Fatalf("%s audit history error = %v", resourceType, err)
	}
	var sawDelete bool
	for _, entry := range logs {
		if entry.Action == "delete" {
			sawDelete = true
			break
		}
	}
	if !sawDelete {
		t.Fatalf("%s audit history missing delete entry: %#v", resourceType, logs)
	}
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
