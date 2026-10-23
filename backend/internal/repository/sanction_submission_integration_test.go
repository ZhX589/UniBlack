package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestSanctionAppealSubmissionAuditRollbackAndRetry(t *testing.T) {
	db := setupIntegrationDatabase(t)
	ctx := context.Background()
	repo := NewSanctionRepository(db)
	actor := integrationAdmin()
	if err := db.Create(actor).Error; err != nil {
		t.Fatal(err)
	}
	sanction := &models.Sanction{UserID: actor.ID, ImposedBy: actor.ID, Type: "warning", Reason: "fixture", StartsAt: time.Now()}
	if err := db.Create(sanction).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE FUNCTION reject_submission_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action = 'appeal' THEN RAISE EXCEPTION 'forced submission audit failure'; END IF; RETURN NEW; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_submission_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION reject_submission_audit()`).Error; err != nil {
		t.Fatal(err)
	}
	submit := func() *models.SanctionAppeal {
		return &models.SanctionAppeal{SanctionID: sanction.ID, SubmittedBy: actor.ID, Reason: "appeal", Status: "approved"}
	}
	if err := repo.CreateAppeal(ctx, submit()); err == nil {
		t.Fatal("audit failure did not propagate")
	}
	var count int64
	if err := db.Model(&models.SanctionAppeal{}).Where("sanction_id = ?", sanction.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("appeals=%d error=%v", count, err)
	}
	if sanctionAuditCount(t, db, sanction.ID, "appeal") != 0 {
		t.Fatal("failed transaction emitted audit")
	}
	if err := db.Exec(`DROP TRIGGER reject_submission_audit ON audit_logs`).Error; err != nil {
		t.Fatal(err)
	}
	appeal := submit()
	if err := repo.CreateAppeal(ctx, appeal); err != nil {
		t.Fatal(err)
	}
	if appeal.Status != "pending" || appeal.ReviewedAt != nil {
		t.Fatal("submission trusted pre-resolved input")
	}
	if sanctionAuditCount(t, db, sanction.ID, "appeal") != 1 {
		t.Fatal("successful submission missing audit")
	}
	if err := repo.CreateAppeal(ctx, submit()); !errors.Is(err, ErrSanctionAlreadyAppealed) {
		t.Fatalf("duplicate error=%v", err)
	}
}

func TestSanctionAppealConcurrentSubmissionHasOneWinner(t *testing.T) {
	db := setupIntegrationDatabase(t)
	repo := NewSanctionRepository(db)
	ctx := context.Background()
	actor := integrationAdmin()
	if err := db.Create(actor).Error; err != nil {
		t.Fatal(err)
	}
	sanction := &models.Sanction{UserID: actor.ID, ImposedBy: actor.ID, Type: "warning", Reason: "fixture", StartsAt: time.Now()}
	if err := db.Create(sanction).Error; err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- repo.CreateAppeal(ctx, &models.SanctionAppeal{SanctionID: sanction.ID, SubmittedBy: actor.ID, Reason: "appeal"})
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	winners, conflicts := 0, 0
	for err := range results {
		if err == nil {
			winners++
		} else if errors.Is(err, ErrSanctionAlreadyAppealed) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d", winners, conflicts)
	}
	if sanctionAuditCount(t, db, sanction.ID, "appeal") != 1 {
		t.Fatal("concurrent submissions emitted duplicate audits")
	}
}
