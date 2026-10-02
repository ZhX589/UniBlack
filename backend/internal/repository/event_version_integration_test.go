package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestEventPublicationVersionAndAuditCommitTogether(t *testing.T) {
	db := setupIntegrationDatabase(t)
	ctx := context.Background()
	user := &models.User{Username: "version-user", Email: "version@example.test", PasswordHash: "test", IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewEventRepository(db)
	subject := &models.Subject{PublicID: "UBS_VERSION_TEST", DisplayName: "version", Status: "active", CreatedBy: &user.ID}
	events := []models.Event{{Title: "event", Details: "details", Status: "published", Severity: 3, SubmittedBy: &user.ID}}
	audit := &models.AuditLog{UserID: &user.ID, Action: "publish", ResourceType: "subject"}
	if err := repo.Publish(ctx, subject, nil, events, audit); err != nil {
		t.Fatal(err)
	}
	var versions []models.EventVersion
	if err := db.Where("event_id = ?", events[0].ID).Find(&versions).Error; err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].VersionNumber != 1 || versions[0].ChangedBy == nil || *versions[0].ChangedBy != user.ID {
		t.Fatalf("versions=%+v", versions)
	}
	// Reject snapshots: none of the publish metadata may survive the failure.
	if err := db.Exec(`CREATE FUNCTION reject_version() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'snapshot failure'; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_version BEFORE INSERT ON event_versions FOR EACH ROW EXECUTE FUNCTION reject_version()`).Error; err != nil {
		t.Fatal(err)
	}
	failed := &models.Subject{PublicID: "UBS_FAILED_VERSION", DisplayName: "fail", Status: "active", CreatedBy: &user.ID}
	if err := repo.Publish(ctx, failed, nil, []models.Event{{Title: "event", Details: "details", Status: "published", SubmittedBy: &user.ID}}, &models.AuditLog{UserID: &user.ID, Action: "publish", ResourceType: "subject"}); err == nil {
		t.Fatal("snapshot failure swallowed")
	}
	var count int64
	if err := db.Model(&models.Subject{}).Where("public_id = ?", "UBS_FAILED_VERSION").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("partial publish count=%d, err=%v", count, err)
	}
}

func TestEventConcurrentVersionsAndAdjudicationActor(t *testing.T) {
	db := setupIntegrationDatabase(t)
	ctx := context.Background()
	owner := &models.User{Username: "version-owner", Email: "version-owner@example.test", PasswordHash: "test", IsActive: true}
	reviewer := &models.User{Username: "version-reviewer", Email: "version-reviewer@example.test", PasswordHash: "test", IsActive: true}
	if err := db.Create(owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(reviewer).Error; err != nil {
		t.Fatal(err)
	}
	subject := &models.Subject{PublicID: "UBS_CONCURRENT_VERSION", DisplayName: "version", Status: "active", CreatedBy: &owner.ID}
	events := []models.Event{{Title: "event", Details: "details", Status: "published", SubmittedBy: &owner.ID}}
	if err := NewEventRepository(db).Publish(ctx, subject, nil, events, &models.AuditLog{UserID: &owner.ID, Action: "publish", ResourceType: "subject"}); err != nil {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, n)
	repo := NewEventVersionRepository(db)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; errs <- repo.RecordVersion(ctx, &events[0], "snapshot") }()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	versions, err := repo.ListByEvent(ctx, events[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for _, v := range versions {
		if seen[v.VersionNumber] {
			t.Fatal("duplicate version")
		}
		seen[v.VersionNumber] = true
	}
	if len(seen) != n+1 {
		t.Fatalf("versions=%d", len(seen))
	}
	for i := 1; i <= n+1; i++ {
		if !seen[i] {
			t.Fatalf("missing version %d", i)
		}
	}
	appeal := &models.Appeal{EventID: &events[0].ID, Reason: "correction", Status: "pending", SubmittedBy: &owner.ID}
	if err := db.Create(appeal).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewAppealRepository(db).ResolveWithConsequences(ctx, appeal.ID, reviewer.ID, "approved", "corrected", "note"); err != nil {
		t.Fatal(err)
	}
	var latest models.EventVersion
	if err := db.Where("event_id = ?", events[0].ID).Order("version_number DESC").First(&latest).Error; err != nil {
		t.Fatal(err)
	}
	if latest.VersionNumber != n+2 || latest.Status != "corrected" || latest.ChangedBy == nil || *latest.ChangedBy != reviewer.ID {
		t.Fatalf("adjudication snapshot=%+v", latest)
	}
}
