package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

type subjectManagementStore struct {
	SubjectStore
	updates, deletes, adds, removes int
}

func (s *subjectManagementStore) GetSubjectByID(context.Context, string) (*models.Subject, error) {
	return &models.Subject{ID: "target"}, nil
}
func (s *subjectManagementStore) UpdateSubject(context.Context, *models.Subject) error {
	s.updates++
	return nil
}
func (s *subjectManagementStore) DeleteSubject(context.Context, string) error {
	s.deletes++
	return nil
}
func (s *subjectManagementStore) AddIdentifier(context.Context, *models.Identifier) error {
	s.adds++
	return nil
}
func (s *subjectManagementStore) RemoveIdentifier(context.Context, string) error {
	s.removes++
	return nil
}

func TestSubjectManagementAllowsGovernanceOperations(t *testing.T) {
	for _, role := range []string{"admin", "moderator"} {
		t.Run(role, func(t *testing.T) {
			store := &subjectManagementStore{}
			s := NewSubjectService(store)
			ctx := context.Background()
			roles := []string{role}
			if _, err := s.UpdateSubject(ctx, "target", UpdateSubjectRequest{Status: "active"}, "staff", roles); err != nil {
				t.Fatal(err)
			}
			if err := s.DeleteSubject(ctx, "target", "staff", roles); err != nil {
				t.Fatal(err)
			}
			if _, err := s.AddIdentifier(ctx, "target", IdentifierRequest{Platform: "test", Value: "account"}, "staff", roles); err != nil {
				t.Fatal(err)
			}
			if err := s.RemoveIdentifier(ctx, "identifier", "staff", roles); err != nil {
				t.Fatal(err)
			}
			if store.updates != 1 || store.deletes != 1 || store.adds != 1 || store.removes != 1 {
				t.Fatalf("operations: %+v", store)
			}
		})
	}
}

func TestSubjectManagementRequiresAuthenticatedGovernanceRole(t *testing.T) {
	for _, tc := range []struct {
		name, actor string
		roles       []string
		want        bool
	}{
		{"anonymous", "", nil, false}, {"empty admin identity", "", []string{"admin"}, false},
		{"ordinary contributor", "owner", []string{"user"}, false},
		{"unrelated user", "other", []string{"user"}, false},
		{"moderator", "staff", []string{"moderator"}, true},
		{"admin", "staff", []string{"admin"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanManageSubject(tc.actor, tc.roles); got != tc.want {
				t.Fatalf("allowed = %v", got)
			}
			if tc.want {
				return
			}
			// Nil repository proves authorization executes before any database access.
			s := NewSubjectService(nil)
			ctx := context.Background()
			if _, err := s.UpdateSubject(ctx, "target", UpdateSubjectRequest{}, tc.actor, tc.roles); !errors.Is(err, ErrSubjectAccessDenied) {
				t.Fatalf("update = %v", err)
			}
			if err := s.DeleteSubject(ctx, "target", tc.actor, tc.roles); !errors.Is(err, ErrSubjectAccessDenied) {
				t.Fatalf("delete = %v", err)
			}
			if _, err := s.AddIdentifier(ctx, "target", IdentifierRequest{}, tc.actor, tc.roles); !errors.Is(err, ErrSubjectAccessDenied) {
				t.Fatalf("add = %v", err)
			}
			if err := s.RemoveIdentifier(ctx, "target", tc.actor, tc.roles); !errors.Is(err, ErrSubjectAccessDenied) {
				t.Fatalf("remove = %v", err)
			}
		})
	}
}
