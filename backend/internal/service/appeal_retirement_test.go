package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

type appealRetirementStub struct {
	appeal            *models.Appeal
	readErr, writeErr error
	calls             int
	actor             string
	governance        bool
}

func (s *appealRetirementStub) GetAppealByID(context.Context, string) (*models.Appeal, error) {
	return s.appeal, s.readErr
}
func (s *appealRetirementStub) DeleteAppeal(_ context.Context, _, actor string, governance bool) error {
	s.calls++
	s.actor, s.governance = actor, governance
	return s.writeErr
}

func TestAppealRetirementAuthorization(t *testing.T) {
	owner := "owner"
	now := time.Now()
	outcome := "upheld"
	for _, tc := range []struct {
		name, actor, status string
		roles               []string
		submitter           *string
		outcome             *string
		reviewed, retired   *time.Time
		want                error
		governance          bool
	}{
		{name: "other pending", actor: "other", status: "pending", submitter: &owner, want: ErrAppealForbidden},
		{name: "owner pending", actor: owner, status: "pending", submitter: &owner},
		{name: "owner approved", actor: owner, status: "approved", submitter: &owner, want: ErrAppealAlreadyResolved},
		{name: "owner rejected", actor: owner, status: "rejected", submitter: &owner, want: ErrAppealAlreadyResolved},
		{name: "owner inconsistent outcome", actor: owner, status: "pending", submitter: &owner, outcome: &outcome, want: ErrAppealAlreadyResolved},
		{name: "owner reviewed", actor: owner, status: "pending", submitter: &owner, reviewed: &now, want: ErrAppealAlreadyResolved},
		{name: "unknown state", actor: owner, status: "unknown", submitter: &owner, want: ErrAppealAlreadyResolved},
		{name: "no owner", actor: "other", status: "pending", want: ErrAppealForbidden},
		{name: "unknown role", actor: "other", roles: []string{"reviewer"}, status: "pending", submitter: &owner, want: ErrAppealForbidden},
		{name: "admin final", actor: "staff", roles: []string{"admin"}, status: "approved", submitter: &owner, governance: true},
		{name: "moderator pending", actor: "staff", roles: []string{"moderator"}, status: "pending", submitter: &owner, governance: true},
		{name: "moderator final", actor: "staff", roles: []string{"moderator"}, status: "rejected", submitter: &owner, governance: true},
		{name: "anonymous admin", roles: []string{"admin"}, status: "pending", submitter: &owner, want: ErrAppealForbidden},
		{name: "retired owner", actor: owner, status: "pending", submitter: &owner, retired: &now, want: ErrAppealNotFound},
		{name: "retired admin", actor: "staff", roles: []string{"admin"}, status: "pending", submitter: &owner, retired: &now, want: ErrAppealNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &appealRetirementStub{appeal: &models.Appeal{Status: tc.status, SubmittedBy: tc.submitter, Outcome: tc.outcome, ReviewedAt: tc.reviewed, DeletedAt: tc.retired}}
			err := retireAppeal(context.Background(), stub, "appeal", tc.actor, tc.roles)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if tc.want != nil {
				if stub.calls != 0 {
					t.Fatal("unauthorized retirement reached persistence")
				}
			} else if stub.calls != 1 || stub.actor != tc.actor || stub.governance != tc.governance {
				t.Fatalf("persistence actor/policy = %#v", stub)
			}
		})
	}
}

func TestAppealRetirementPropagatesPersistenceErrors(t *testing.T) {
	owner := "owner"
	for _, want := range []error{ErrAppealNotFound, ErrAppealAlreadyResolved, ErrAppealForbidden, errors.New("audit unavailable")} {
		stub := &appealRetirementStub{appeal: &models.Appeal{Status: "pending", SubmittedBy: &owner}, writeErr: want}
		if err := retireAppeal(context.Background(), stub, "appeal", owner, nil); !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
	}
	stub := &appealRetirementStub{readErr: ErrAppealNotFound}
	if err := retireAppeal(context.Background(), stub, "missing", owner, nil); !errors.Is(err, ErrAppealNotFound) || stub.calls != 0 {
		t.Fatalf("missing retirement = %v, writes = %d", err, stub.calls)
	}
}
