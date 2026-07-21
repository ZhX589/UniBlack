package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

type caseDeleteRepository struct {
	caseValue   *models.Case
	deleteCalls int
	operations  *[]string
}

func (r *caseDeleteRepository) GetCaseByID(context.Context, string) (*models.Case, error) {
	return r.caseValue, nil
}

func (r *caseDeleteRepository) DeleteCase(context.Context, string) error {
	r.deleteCalls++
	*r.operations = append(*r.operations, "metadata")
	return nil
}

func (*caseDeleteRepository) CreateCase(context.Context, *models.Case) error { return nil }
func (*caseDeleteRepository) ListCases(context.Context, int, int, string, string) ([]models.Case, int64, error) {
	return nil, 0, nil
}
func (*caseDeleteRepository) UpdateCase(context.Context, *models.Case) error { return nil }
func (*caseDeleteRepository) ReviewCase(context.Context, string, string, string, string) error {
	return nil
}
func (*caseDeleteRepository) GetCasesBySubjectID(context.Context, string) ([]models.Case, error) {
	return nil, nil
}

type caseDeleteStorage struct {
	deleted    []string
	err        error
	operations *[]string
}

type caseDeleteAuditRepository struct{}

func (caseDeleteAuditRepository) CreateAuditLog(context.Context, *models.AuditLog) error { return nil }
func (caseDeleteAuditRepository) GetAuditLogsByResource(context.Context, string, string) ([]models.AuditLog, error) {
	return nil, nil
}

func (s *caseDeleteStorage) Upload(context.Context, string, io.Reader, string) (string, error) {
	return "", nil
}

func (s *caseDeleteStorage) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	*s.operations = append(*s.operations, "object:"+key)
	return s.err
}

func (s *caseDeleteStorage) GetURL(string) string { return "" }
func (s *caseDeleteStorage) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, nil
}
func (s *caseDeleteStorage) Path(string) (string, error) { return "", nil }

func TestDeleteCaseDeletesAllStoredEvidenceBeforeMetadata(t *testing.T) {
	fileKey := "subjects/subject-1/evidence/file.pdf"
	textKey := "subjects/subject-1/evidence/note.txt"
	operations := []string{}
	repo := &caseDeleteRepository{caseValue: &models.Case{
		ID: "case-id",
		Evidences: []models.Evidence{
			{Type: "file", StorageKey: &fileKey},
			{Type: "text", StorageKey: &textKey},
			{Type: "link"},
		},
	}, operations: &operations}
	store := &caseDeleteStorage{operations: &operations}
	service := NewCaseService(repo, nil, caseDeleteAuditRepository{}, store)

	if err := service.DeleteCase(context.Background(), "case-id", "user-id"); err != nil {
		t.Fatal(err)
	}
	if got, want := len(store.deleted), 2; got != want {
		t.Fatalf("deleted object count = %d, want %d", got, want)
	}
	if store.deleted[0] != fileKey || store.deleted[1] != textKey {
		t.Fatalf("deleted keys = %q, want %q then %q", store.deleted, fileKey, textKey)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("metadata deletes = %d, want 1", repo.deleteCalls)
	}
	wantOperations := []string{"object:" + fileKey, "object:" + textKey, "metadata"}
	for i, want := range wantOperations {
		if operations[i] != want {
			t.Fatalf("operations = %q, want %q", operations, wantOperations)
		}
	}
}

func TestDeleteCaseRetainsMetadataWhenObjectCleanupFails(t *testing.T) {
	key := "subjects/subject-1/evidence/file.pdf"
	operations := []string{}
	repo := &caseDeleteRepository{caseValue: &models.Case{ID: "case-id", Evidences: []models.Evidence{{StorageKey: &key}}}, operations: &operations}
	cleanupErr := errors.New("object store unavailable")
	service := NewCaseService(repo, nil, caseDeleteAuditRepository{}, &caseDeleteStorage{err: cleanupErr, operations: &operations})

	err := service.DeleteCase(context.Background(), "case-id", "user-id")
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("delete error = %v, want cleanup error %v", err, cleanupErr)
	}
	if repo.deleteCalls != 0 {
		t.Fatalf("metadata deletes = %d, want 0", repo.deleteCalls)
	}
}
