package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestAppealRetirementEnforcesOwnerStateAndAtomicAudit(t *testing.T) {
	database := setupIntegrationDatabase(t)
	ctx := context.Background()
	owner := integrationAdmin()
	if err := database.Create(owner).Error; err != nil {
		t.Fatal(err)
	}
	other := &models.User{Username: "other", Email: "other@example.test", PasswordHash: "test", AuthProvider: "local", IsActive: true}
	if err := database.Create(other).Error; err != nil {
		t.Fatal(err)
	}
	subject := &models.Subject{DisplayName: "retirement", Status: "active"}
	if err := database.Create(subject).Error; err != nil {
		t.Fatal(err)
	}
	caseObj := &models.Case{SubjectID: subject.ID, Title: "legacy fixture", Status: "approved", Severity: 1}
	if err := database.Create(caseObj).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewAppealRepository(database)
	create := func(status string) *models.Appeal {
		t.Helper()
		appeal := &models.Appeal{CaseID: &caseObj.ID, Reason: "retirement", Status: status, SubmittedBy: &owner.ID}
		if err := repo.CreateAppeal(ctx, appeal); err != nil {
			t.Fatal(err)
		}
		return appeal
	}
	assertState := func(appeal *models.Appeal, retired bool, audits int64) {
		t.Helper()
		got, err := repo.GetAppealByID(ctx, appeal.ID)
		if err != nil || (got.DeletedAt != nil) != retired {
			t.Fatalf("retirement = %#v, %v", got, err)
		}
		var count int64
		if err := database.Model(&models.AuditLog{}).Where("resource_type = 'appeal' AND resource_id = ? AND action = 'delete'", appeal.ID).Count(&count).Error; err != nil || count != audits {
			t.Fatalf("audits = %d, want %d, error %v", count, audits, err)
		}
	}
	appeal := create("pending")
	if err := repo.DeleteAppeal(ctx, appeal.ID, other.ID, false); !errors.Is(err, ErrAppealForbidden) {
		t.Fatalf("other retirement = %v", err)
	}
	assertState(appeal, false, 0)
	if err := repo.DeleteAppeal(ctx, appeal.ID, owner.ID, false); err != nil {
		t.Fatal(err)
	}
	assertState(appeal, true, 1)
	if err := repo.DeleteAppeal(ctx, appeal.ID, owner.ID, false); !errors.Is(err, ErrAppealNotFound) {
		t.Fatalf("repeat retirement = %v", err)
	}
	assertState(appeal, true, 1)

	// Simulate resolution between the service authorization read and repository write.
	final := create("pending")
	if err := repo.ResolveWithConsequences(ctx, final.ID, other.ID, "rejected", "upheld", "decision"); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteAppeal(ctx, final.ID, owner.ID, false); !errors.Is(err, ErrAppealAlreadyResolved) {
		t.Fatalf("final owner retirement = %v", err)
	}
	assertState(final, false, 0)
	if err := repo.DeleteAppeal(ctx, final.ID, other.ID, true); err != nil {
		t.Fatal(err)
	}
	assertState(final, true, 1)

	// The missing governance actor violates the audit FK after the update: rollback
	// must leave the appeal active, rather than silently hiding a row without audit.
	rollback := create("pending")
	if err := repo.DeleteAppeal(ctx, rollback.ID, "00000000-0000-0000-0000-000000000001", true); err == nil {
		t.Fatal("expected audit foreign key failure")
	}
	assertState(rollback, false, 0)
}
