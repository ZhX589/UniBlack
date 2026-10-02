package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func sanctionFixture(t *testing.T, database *gorm.DB, actor string) (*models.Sanction, *models.SanctionAppeal) {
	t.Helper()
	sanction := &models.Sanction{UserID: actor, Type: "submission_ban", Reason: "fixture", ImposedBy: actor, StartsAt: time.Now()}
	if err := database.Create(sanction).Error; err != nil {
		t.Fatal(err)
	}
	appeal := &models.SanctionAppeal{SanctionID: sanction.ID, SubmittedBy: actor, Reason: "appeal", Status: "pending"}
	if err := database.Create(appeal).Error; err != nil {
		t.Fatal(err)
	}
	return sanction, appeal
}

func sanctionAuditCount(t *testing.T, database *gorm.DB, id, action string) int64 {
	t.Helper()
	var count int64
	if err := database.Model(&models.AuditLog{}).Where("resource_id = ? AND action = ?", id, action).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestSanctionResolutionAtomicConsequencesAndTerminalConflict(t *testing.T) {
	database := setupIntegrationDatabase(t) // DATABASE_URL absent: explicit skip; otherwise isolated migrated schema.
	actor := integrationAdmin()
	if err := database.Create(actor).Error; err != nil {
		t.Fatal(err)
	}
	repo, ctx := NewSanctionRepository(database), context.Background()
	created := &models.Sanction{UserID: actor.ID, Type: "warning", Reason: "created", ImposedBy: actor.ID, StartsAt: time.Now()}
	if err := repo.Create(ctx, created); err != nil {
		t.Fatal(err)
	}
	if sanctionAuditCount(t, database, created.ID, "create") != 1 {
		t.Fatal("create missing atomic audit")
	}
	for _, status := range []string{"approved", "rejected"} {
		t.Run(status, func(t *testing.T) {
			sanction, appeal := sanctionFixture(t, database, actor.ID)
			got, err := repo.ResolveAppealWithConsequences(ctx, appeal.ID, actor.ID, status, "decision")
			if err != nil || got.Status != status || got.ReviewedAt == nil || got.ReviewedBy == nil || *got.ReviewedBy != actor.ID {
				t.Fatalf("resolution = %#v, %v", got, err)
			}
			stored, err := repo.GetByID(ctx, sanction.ID)
			if err != nil || (stored.RevokedAt != nil) != (status == "approved") {
				t.Fatalf("sanction = %#v, %v", stored, err)
			}
			wantRevoke := int64(0)
			if status == "approved" {
				wantRevoke = 1
			}
			if sanctionAuditCount(t, database, sanction.ID, "revoke") != wantRevoke || sanctionAuditCount(t, database, appeal.ID, "resolve_appeal") != 1 {
				t.Fatal("incorrect consequence audits")
			}
			if _, err := repo.ResolveAppealWithConsequences(ctx, appeal.ID, actor.ID, "approved", "repeat"); !errors.Is(err, ErrSanctionAppealAlreadyResolved) {
				t.Fatalf("repeat resolution = %v", err)
			}
			if sanctionAuditCount(t, database, appeal.ID, "resolve_appeal") != 1 {
				t.Fatal("repeat emitted audit")
			}
		})
	}
	sanction, appeal := sanctionFixture(t, database, actor.ID)
	if err := repo.Revoke(ctx, sanction.ID, actor.ID, "manual"); err != nil {
		t.Fatal(err)
	}
	before, err := repo.GetByID(ctx, sanction.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ResolveAppealWithConsequences(ctx, appeal.ID, actor.ID, "approved", "later"); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetByID(ctx, sanction.ID)
	if err != nil || !after.RevokedAt.Equal(*before.RevokedAt) || *after.RevokeReason != "manual" || sanctionAuditCount(t, database, sanction.ID, "revoke") != 1 {
		t.Fatalf("existing revocation overwritten: %#v, %v", after, err)
	}
	if err := repo.Revoke(ctx, sanction.ID, actor.ID, "repeat"); !errors.Is(err, ErrSanctionNotFound) {
		t.Fatalf("repeat revoke = %v", err)
	}
	if _, err := repo.ResolveAppealWithConsequences(ctx, "00000000-0000-0000-0000-000000000001", actor.ID, "approved", "missing"); !errors.Is(err, ErrSanctionAppealNotFound) {
		t.Fatalf("missing appeal = %v", err)
	}
}

func TestSanctionAuditFailureRollsBackAllCommands(t *testing.T) {
	database := setupIntegrationDatabase(t)
	actor := integrationAdmin()
	if err := database.Create(actor).Error; err != nil {
		t.Fatal(err)
	}
	repo, ctx := NewSanctionRepository(database), context.Background()
	// Fail at audit insertion, not at an earlier sanction/reviewer foreign key.
	if err := database.Exec(`CREATE FUNCTION reject_sanction_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced audit failure'; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TRIGGER reject_sanction_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION reject_sanction_audit()`).Error; err != nil {
		t.Fatal(err)
	}
	created := &models.Sanction{UserID: actor.ID, Type: "warning", Reason: "test", ImposedBy: actor.ID, StartsAt: time.Now()}
	if err := repo.Create(ctx, created); err == nil {
		t.Fatal("expected create audit failure")
	}
	var count int64
	if err := database.Model(&models.Sanction{}).Where("id = ?", created.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("created rows = %d, %v", count, err)
	}
	sanction, appeal := sanctionFixture(t, database, actor.ID)
	if err := repo.Revoke(ctx, sanction.ID, actor.ID, "manual"); err == nil {
		t.Fatal("expected revoke audit failure")
	}
	for _, status := range []string{"approved", "rejected"} {
		if _, err := repo.ResolveAppealWithConsequences(ctx, appeal.ID, actor.ID, status, "decision"); err == nil {
			t.Fatal("expected resolve audit failure")
		}
		stored, err := repo.GetAppealByID(ctx, appeal.ID)
		if err != nil || stored.Status != "pending" || stored.ReviewedAt != nil {
			t.Fatalf("appeal not rolled back: %#v, %v", stored, err)
		}
	}
	stored, err := repo.GetByID(ctx, sanction.ID)
	if err != nil || stored.RevokedAt != nil {
		t.Fatalf("revoke not rolled back: %#v, %v", stored, err)
	}
	if sanctionAuditCount(t, database, sanction.ID, "revoke") != 0 || sanctionAuditCount(t, database, appeal.ID, "resolve_appeal") != 0 {
		t.Fatal("failed commands retained audits")
	}
	// Allow revoke audit but fail the later resolution audit, proving both roll back.
	if err := database.Exec(`DROP TRIGGER reject_sanction_audit ON audit_logs`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TRIGGER reject_sanction_audit BEFORE INSERT ON audit_logs FOR EACH ROW WHEN (NEW.action = 'resolve_appeal') EXECUTE FUNCTION reject_sanction_audit()`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ResolveAppealWithConsequences(ctx, appeal.ID, actor.ID, "approved", "decision"); err == nil {
		t.Fatal("expected final audit failure")
	}
	stored, err = repo.GetByID(ctx, sanction.ID)
	if err != nil || stored.RevokedAt != nil || sanctionAuditCount(t, database, sanction.ID, "revoke") != 0 {
		t.Fatalf("earlier revoke/audit not rolled back: %#v, %v", stored, err)
	}
}

func TestSanctionConcurrentResolutionHasOneTerminalDecision(t *testing.T) {
	database := setupIntegrationDatabase(t)
	actor := integrationAdmin()
	if err := database.Create(actor).Error; err != nil {
		t.Fatal(err)
	}
	repo, ctx := NewSanctionRepository(database), context.Background()
	sanction, appeal := sanctionFixture(t, database, actor.ID)
	start, results := make(chan struct{}), make(chan error, 2)
	var wg sync.WaitGroup
	for _, status := range []string{"approved", "rejected"} {
		wg.Add(1)
		go func(status string) {
			defer wg.Done()
			<-start
			_, err := repo.ResolveAppealWithConsequences(ctx, appeal.ID, actor.ID, status, "concurrent")
			results <- err
		}(status)
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrSanctionAppealAlreadyResolved) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 || sanctionAuditCount(t, database, appeal.ID, "resolve_appeal") != 1 {
		t.Fatalf("success/conflict = %d/%d", success, conflict)
	}
	final, err := repo.GetAppealByID(ctx, appeal.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetByID(ctx, sanction.ID)
	if err != nil || (stored.RevokedAt != nil) != (final.Status == "approved") {
		t.Fatalf("inconsistent terminal consequence: %#v, %#v, %v", final, stored, err)
	}
}
