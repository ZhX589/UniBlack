package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

// stubEvidenceRepo satisfies evidenceRepository for authorization tests.
type stubEvidenceRepo struct {
	existing *models.Evidence
	err      error
	deleted  int
}

func (r *stubEvidenceRepo) CreateEvidence(context.Context, *models.Evidence) error { return nil }
func (r *stubEvidenceRepo) GetEvidenceByID(context.Context, string) (*models.Evidence, error) {
	return r.existing, r.err
}
func (r *stubEvidenceRepo) GetEvidenceByCaseID(context.Context, string) ([]models.Evidence, error) {
	return nil, nil
}
func (r *stubEvidenceRepo) GetEvidenceBySHA256(context.Context, string) (*models.Evidence, error) {
	return nil, nil
}
func (r *stubEvidenceRepo) DeleteEvidence(context.Context, string) error {
	r.deleted++
	return nil
}

func TestDeleteEvidenceAuthorizesBeforeMutating(t *testing.T) {
	uploader := "uploader-1"
	key := "subjects/s/events/1/evidence/1.bin"
	existing := &models.Evidence{ID: "ev-1", UploadedBy: &uploader, StorageKey: &key}

	cases := []struct {
		name    string
		actor   string
		roles   []string
		wantErr error
	}{
		{"owner", uploader, nil, nil},
		{"governance admin", "other-admin", []string{"admin"}, nil},
		{"governance moderator", "other-mod", []string{"moderator"}, nil},
		{"empty actor", "", nil, ErrEvidenceAccessDenied},
		{"empty actor with role", "", []string{"admin"}, ErrEvidenceAccessDenied},
		{"missing evidence with role", "staff", []string{"admin"}, ErrEvidenceAccessDenied},
		{"unrelated user", "stranger", []string{"user"}, ErrEvidenceAccessDenied},
		{"empty uploader cannot be matched", "stranger", nil, ErrEvidenceAccessDenied},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := *existing
			if tc.name == "empty uploader cannot be matched" {
				ev.UploadedBy = nil
			}
			repo := &stubEvidenceRepo{existing: &ev}
			if tc.name == "missing evidence with role" {
				repo.existing = nil
			}
			store := &recordingStorage{}
			svc := NewEvidenceService(repo, existingCaseRepository{}, store)
			err := svc.DeleteEvidence(context.Background(), "ev-1", tc.actor, tc.roles)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (repo.deleted != 1 || store.deletedKey != key) {
				t.Fatal("authorized deletion did not remove both metadata and object")
			}
			if tc.wantErr != nil {
				if repo.deleted != 0 {
					t.Fatal("metadata deleted despite denial")
				}
				if store.deletedKey != "" {
					t.Fatal("storage deleted despite denial")
				}
			}
		})
	}
}
