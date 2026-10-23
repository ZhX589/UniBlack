package service

import (
	"context"
	"errors"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"testing"
)

type sanctionSubmitStub struct {
	sanctionRepository
	calls int
	row   *models.SanctionAppeal
	err   error
}

func (s *sanctionSubmitStub) CreateAppeal(_ context.Context, row *models.SanctionAppeal) error {
	s.calls++
	s.row = row
	return s.err
}
func TestSanctionSubmissionDelegatesAtomicCommandAndErrors(t *testing.T) {
	for _, failure := range []error{nil, repository.ErrSanctionForbidden, repository.ErrSanctionRevoked, repository.ErrSanctionAlreadyAppealed, errors.New("audit unavailable")} {
		stub := &sanctionSubmitStub{err: failure}
		svc := &SanctionService{repo: stub}
		got, err := svc.Appeal(context.Background(), "sanction", "owner", "reason")
		if !errors.Is(err, failure) || stub.calls != 1 {
			t.Fatalf("error=%v calls=%d", err, stub.calls)
		}
		if stub.row.Status != "pending" || stub.row.SubmittedBy != "owner" || stub.row.SanctionID != "sanction" {
			t.Fatal("submission identity or status changed")
		}
		if failure == nil && got != stub.row {
			t.Fatal("returned unrelated submission")
		}
	}
}
func TestSanctionSubmissionEmptyReasonRejectedBeforeCommand(t *testing.T) {
	stub := &sanctionSubmitStub{}
	svc := &SanctionService{repo: stub}
	if _, err := svc.Appeal(context.Background(), "sanction", "owner", " \n "); !errors.Is(err, repository.ErrSanctionInvalidRequest) {
		t.Fatalf("error=%v", err)
	}
	if stub.calls != 0 {
		t.Fatal("invalid submission reached repository")
	}
}
