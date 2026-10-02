package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

type appealReviewRepositoryStub struct {
	appealRepository
	writes, reads        int
	status, actor, notes string
	err                  error
}

func (s *appealReviewRepositoryStub) ReviewAppeal(_ context.Context, _, actor, status, notes string) error {
	s.writes++
	s.status, s.actor, s.notes = status, actor, notes
	return s.err
}
func (s *appealReviewRepositoryStub) GetAppealByID(context.Context, string) (*models.Appeal, error) {
	s.reads++
	return &models.Appeal{Status: s.status}, nil
}
func TestLegacyAppealReviewValidationAndAtomicBoundary(t *testing.T) {
	for _, status := range []string{"", "pending", "withdrawn", "corrected", "unknown"} {
		stub := &appealReviewRepositoryStub{}
		svc := &AppealService{appealRepo: stub}
		if _, err := svc.ReviewAppeal(context.Background(), "id", ReviewAppealRequest{Status: status}, "staff"); !errors.Is(err, ErrAppealInvalidReview) || stub.writes != 0 || stub.reads != 0 {
			t.Fatalf("invalid status %q: %v, stub=%#v", status, err, stub)
		}
	}
	for _, status := range []string{"approved", "rejected"} {
		stub := &appealReviewRepositoryStub{}
		// No Case/audit dependency: the repository command owns the transaction.
		svc := &AppealService{appealRepo: stub}
		got, err := svc.ReviewAppeal(context.Background(), "id", ReviewAppealRequest{Status: status, ReviewNotes: "notes"}, "staff")
		if err != nil || got.Status != status || got.Outcome != nil || stub.writes != 1 || stub.reads != 1 || stub.actor != "staff" || stub.notes != "notes" {
			t.Fatalf("review: %#v, %v, stub=%#v", got, err, stub)
		}
	}
	for _, want := range []error{ErrAppealNotFound, ErrAppealAlreadyResolved, ErrAppealResolveRequired, ErrAppealInvalidTarget, errors.New("audit failed")} {
		stub := &appealReviewRepositoryStub{err: want}
		svc := &AppealService{appealRepo: stub}
		if _, err := svc.ReviewAppeal(context.Background(), "id", ReviewAppealRequest{Status: "approved"}, "staff"); !errors.Is(err, want) || stub.reads != 0 {
			t.Fatalf("persistence failure: %v, stub=%#v", err, stub)
		}
	}
}
