package repository

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestLegacyAppealReviewIsolatedTransactions(t *testing.T) {
	db := setupIntegrationDatabase(t)
	ctx := context.Background()
	user := integrationAdmin()
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	subject := &models.Subject{DisplayName: "legacy review", Status: "active"}
	if err := db.Create(subject).Error; err != nil {
		t.Fatal(err)
	}
	description, verdict := "preserve description", "original verdict"
	c := &models.Case{SubjectID: subject.ID, Title: "preserve title", Description: &description, Verdict: &verdict, Status: "approved", Severity: 4, SubmittedBy: &user.ID}
	if err := db.Create(c).Error; err != nil {
		t.Fatal(err)
	}
	var before models.Case
	if err := db.First(&before, "id = ?", c.ID).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewAppealRepository(db)
	create := func() *models.Appeal {
		t.Helper()
		a := &models.Appeal{CaseID: &c.ID, Reason: "review", Status: "pending", SubmittedBy: &user.ID}
		if err := repo.CreateAppeal(ctx, a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	auditCount := func(id string) int64 {
		t.Helper()
		var n int64
		if err := db.Model(&models.AuditLog{}).Where("resource_type = 'appeal' AND resource_id = ? AND action = 'review'", id).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		return n
	}
	for _, status := range []string{"approved", "rejected"} {
		a := create()
		if err := repo.ReviewAppeal(ctx, a.ID, user.ID, status, "notes"); err != nil {
			t.Fatal(err)
		}
		got, err := repo.GetAppealByID(ctx, a.ID)
		if err != nil || got.Status != status || got.Outcome != nil || got.ResolutionReason != nil || got.ReviewedAt == nil || got.ReviewedBy == nil || *got.ReviewedBy != user.ID || auditCount(a.ID) != 1 {
			t.Fatalf("decision %#v, %v", got, err)
		}
		if err := repo.ReviewAppeal(ctx, a.ID, user.ID, "rejected", "overwrite"); !errors.Is(err, ErrAppealAlreadyResolved) {
			t.Fatalf("repeat %v", err)
		}
	}
	var after models.Case
	if err := db.First(&after, "id = ?", c.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Case altered: before=%#v after=%#v", before, after)
	}
	for _, status := range []string{"", "pending", "corrected", "withdrawn"} {
		a := create()
		if err := repo.ReviewAppeal(ctx, a.ID, user.ID, status, ""); !errors.Is(err, ErrAppealInvalidReview) {
			t.Fatalf("invalid %q: %v", status, err)
		}
		got, err := repo.GetAppealByID(ctx, a.ID)
		if err != nil || got.Status != "pending" || auditCount(a.ID) != 0 {
			t.Fatalf("invalid mutation %#v %v", got, err)
		}
	}
	// Audit failure occurs after the update, and must roll back the decision.
	if err := db.Exec(`CREATE FUNCTION reject_legacy_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action = 'review' AND NEW.resource_type = 'appeal' THEN RAISE EXCEPTION 'audit failure'; END IF; RETURN NEW; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_legacy_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION reject_legacy_audit()`).Error; err != nil {
		t.Fatal(err)
	}
	rollback := create()
	if err := repo.ReviewAppeal(ctx, rollback.ID, user.ID, "approved", ""); err == nil {
		t.Fatal("audit failure swallowed")
	}
	got, err := repo.GetAppealByID(ctx, rollback.ID)
	if err != nil || got.Status != "pending" || got.ReviewedAt != nil || auditCount(rollback.ID) != 0 {
		t.Fatalf("partial decision %#v %v", got, err)
	}
	if err := db.Exec(`DROP TRIGGER reject_legacy_audit ON audit_logs`).Error; err != nil {
		t.Fatal(err)
	}

	// Dirty pending rows are not reopened; retirement shares the same appeal lock.
	now := time.Now()
	outcome := "upheld"
	for _, updates := range []map[string]interface{}{
		{"deleted_at": now}, {"outcome": outcome}, {"reviewed_at": now}, {"reviewed_by": user.ID}, {"resolution_reason": "already decided"}, {"status": "unknown"},
	} {
		a := create()
		if err := db.Model(a).Updates(updates).Error; err != nil {
			t.Fatal(err)
		}
		want := ErrAppealAlreadyResolved
		if _, ok := updates["deleted_at"]; ok {
			want = ErrAppealNotFound
		}
		if err := repo.ReviewAppeal(ctx, a.ID, user.ID, "approved", ""); !errors.Is(err, want) {
			t.Fatalf("guard %v: %v", updates, err)
		}
		if auditCount(a.ID) != 0 {
			t.Fatal("guard wrote audit")
		}
	}
	// Two simultaneous terminal requests must produce exactly one decision/audit.
	concurrent := create()
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, status := range []string{"approved", "rejected"} {
		wg.Add(1)
		go func(status string) {
			defer wg.Done()
			<-start
			errs <- repo.ReviewAppeal(ctx, concurrent.ID, user.ID, status, "")
		}(status)
	}
	close(start)
	wg.Wait()
	close(errs)
	successes, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrAppealAlreadyResolved) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 || auditCount(concurrent.ID) != 1 {
		t.Fatalf("success=%d conflict=%d", successes, conflicts)
	}
	// Legacy review races with retirement on the same lock: only one commits.
	race := create()
	raceStart := make(chan struct{})
	raceErrs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-raceStart
		raceErrs <- repo.ReviewAppeal(ctx, race.ID, user.ID, "approved", "")
	}()
	go func() { defer wg.Done(); <-raceStart; raceErrs <- repo.DeleteAppeal(ctx, race.ID, user.ID, false) }()
	close(raceStart)
	wg.Wait()
	close(raceErrs)
	successes = 0
	for err := range raceErrs {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrAppealNotFound) && !errors.Is(err, ErrAppealAlreadyResolved) {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("review/retire commits=%d", successes)
	}
	var decisions int64
	if err := db.Model(&models.AuditLog{}).Where("resource_type = 'appeal' AND resource_id = ?", race.ID).Count(&decisions).Error; err != nil || decisions != 1 {
		t.Fatalf("race audits %d %v", decisions, err)
	}
	// Formal resolution and legacy review also serialize; neither overwrites a terminal.
	formalRace := create()
	formalStart := make(chan struct{})
	formalErrs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-formalStart
		formalErrs <- repo.ReviewAppeal(ctx, formalRace.ID, user.ID, "approved", "")
	}()
	go func() {
		defer wg.Done()
		<-formalStart
		formalErrs <- repo.ResolveWithConsequences(ctx, formalRace.ID, user.ID, "rejected", "upheld", "formal")
	}()
	close(formalStart)
	wg.Wait()
	close(formalErrs)
	successes = 0
	for err := range formalErrs {
		if err == nil {
			successes++
		} else if err.Error() != ErrAppealAlreadyResolved.Error() {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("review/resolve commits=%d", successes)
	}
	if err := db.Model(&models.AuditLog{}).Where("resource_type = 'appeal' AND resource_id = ?", formalRace.ID).Count(&decisions).Error; err != nil || decisions != 1 {
		t.Fatalf("formal race audits %d %v", decisions, err)
	}
	if err := repo.ReviewAppeal(ctx, "00000000-0000-0000-0000-000000000001", user.ID, "approved", ""); !errors.Is(err, ErrAppealNotFound) {
		t.Fatalf("missing %v", err)
	}
	// A retired appeal cannot be resurrected by the legacy endpoint.
	retired := create()
	if err := repo.DeleteAppeal(ctx, retired.ID, user.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReviewAppeal(ctx, retired.ID, user.ID, "approved", ""); !errors.Is(err, ErrAppealNotFound) {
		t.Fatalf("retired: %v", err)
	}
}

func TestLegacyEventReviewRejectsWithoutCanonicalConsequences(t *testing.T) {
	db := setupIntegrationDatabase(t)
	ctx := context.Background()
	user := integrationAdmin()
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	subject := &models.Subject{DisplayName: "event legacy", Status: "active"}
	if err := db.Create(subject).Error; err != nil {
		t.Fatal(err)
	}
	event := &models.Event{SubjectID: subject.ID, Title: "event", Details: "details", Status: "published", SubmittedBy: &user.ID}
	if err := db.Create(event).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewAppealRepository(db)
	for _, status := range []string{"approved", "rejected"} {
		a := &models.Appeal{EventID: &event.ID, Reason: "event appeal", Status: "pending", SubmittedBy: &user.ID}
		if err := repo.CreateAppeal(ctx, a); err != nil {
			t.Fatal(err)
		}
		if err := repo.ReviewAppeal(ctx, a.ID, user.ID, status, "legacy"); !errors.Is(err, ErrAppealResolveRequired) {
			t.Fatalf("event review %v", err)
		}
		got, err := repo.GetAppealByID(ctx, a.ID)
		if err != nil || got.Status != "pending" || got.Outcome != nil || got.ReviewedAt != nil {
			t.Fatalf("event bypass %#v %v", got, err)
		}
		var n int64
		if err := db.Model(&models.AuditLog{}).Where("resource_type = 'appeal' AND resource_id = ?", a.ID).Count(&n).Error; err != nil || n != 0 {
			t.Fatalf("audit %d %v", n, err)
		}
		// The rejected bypass must not prevent subsequent formal resolution.
		if err := repo.ResolveWithConsequences(ctx, a.ID, user.ID, "approved", "withdrawn", "formal"); err != nil {
			t.Fatal(err)
		}
		if err := db.First(event, "id = ?", event.ID).Error; err != nil || event.Status != "withdrawn" {
			t.Fatalf("formal event %#v %v", event, err)
		}
	}
}
