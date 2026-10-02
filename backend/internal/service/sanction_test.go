package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

// Embedding catches accidental calls to legacy reads or independent writes:
// only the command methods explicitly implemented here may be used.
type sanctionCommandStub struct {
	sanctionRepository
	calls  int
	got    [4]string
	result *models.SanctionAppeal
	err    error
}

func (r *sanctionCommandStub) ResolveAppealWithConsequences(_ context.Context, id, actor, status, notes string) (*models.SanctionAppeal, error) {
	r.calls++
	r.got = [4]string{id, actor, status, notes}
	return r.result, r.err
}
func (r *sanctionCommandStub) Create(_ context.Context, sanction *models.Sanction) error {
	r.calls++
	return r.err
}
func (r *sanctionCommandStub) Revoke(_ context.Context, id, actor, reason string) error {
	r.calls++
	return r.err
}

func TestSanctionResolveDelegatesSingleAtomicCommand(t *testing.T) {
	for _, status := range []string{"approved", "rejected"} {
		t.Run(status, func(t *testing.T) {
			want := &models.SanctionAppeal{ID: "appeal", Status: status}
			repo := &sanctionCommandStub{result: want}
			service := &SanctionService{repo: repo}
			got, err := service.ResolveAppeal(context.Background(), "appeal", "actor", status, "decision")
			if err != nil || got != want || repo.calls != 1 || repo.got != [4]string{"appeal", "actor", status, "decision"} {
				t.Fatalf("ResolveAppeal = (%#v, %v), calls %d args %v", got, err, repo.calls, repo.got)
			}
		})
	}
}

func TestSanctionResolvePropagatesAllCommandErrors(t *testing.T) {
	for _, want := range []error{repository.ErrSanctionNotFound, repository.ErrSanctionAppealNotFound, repository.ErrSanctionAppealAlreadyResolved, errors.New("audit insert failed"), context.Canceled} {
		repo := &sanctionCommandStub{err: want}
		got, err := (&SanctionService{repo: repo}).ResolveAppeal(context.Background(), "appeal", "actor", "approved", "decision")
		if got != nil || !errors.Is(err, want) || repo.calls != 1 {
			t.Fatalf("ResolveAppeal = (%#v, %v), want error %v, calls %d", got, err, want, repo.calls)
		}
	}
}

func TestSanctionResolveRejectsInvalidStatusBeforeCommand(t *testing.T) {
	repo := &sanctionCommandStub{}
	if _, err := (&SanctionService{repo: repo}).ResolveAppeal(context.Background(), "appeal", "actor", "pending", "decision"); err == nil || repo.calls != 0 {
		t.Fatalf("invalid status = %v, calls %d", err, repo.calls)
	}
}

func TestSanctionCreateAndRevokePropagateAtomicAuditFailure(t *testing.T) {
	want := errors.New("audit insert failed")
	repo := &sanctionCommandStub{err: want}
	service := &SanctionService{repo: repo}
	if got, err := service.Create(context.Background(), CreateSanctionRequest{UserID: "user", Type: "warning", Reason: "reason"}, "actor"); got != nil || !errors.Is(err, want) {
		t.Fatalf("Create = (%#v, %v)", got, err)
	}
	if err := service.Revoke(context.Background(), "sanction", "actor", "reason"); !errors.Is(err, want) {
		t.Fatalf("Revoke = %v", err)
	}
	if repo.calls != 2 {
		t.Fatalf("commands = %d, want 2", repo.calls)
	}
}
