package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

type duplicateEvidenceRepository struct {
	existing     *models.Evidence
	createErr    error
	createCalls  int
	deleteCalls  int
	requestedSum string
}

func (r *duplicateEvidenceRepository) CreateEvidence(context.Context, *models.Evidence) error {
	r.createCalls++
	return r.createErr
}

func (r *duplicateEvidenceRepository) GetEvidenceByID(context.Context, string) (*models.Evidence, error) {
	return r.existing, nil
}

func (r *duplicateEvidenceRepository) GetEvidenceByCaseID(context.Context, string) ([]models.Evidence, error) {
	return nil, nil
}

func (r *duplicateEvidenceRepository) GetEvidenceBySHA256(_ context.Context, sum string) (*models.Evidence, error) {
	r.requestedSum = sum
	return r.existing, nil
}

func (r *duplicateEvidenceRepository) DeleteEvidence(context.Context, string) error {
	r.deleteCalls++
	return nil
}

type existingCaseRepository struct{}

func (existingCaseRepository) GetCaseByID(context.Context, string) (*models.Case, error) {
	return &models.Case{}, nil
}

type recordingStorage struct {
	uploadedKey string
	deletedKey  string
	deleteErr   error
}

func (s *recordingStorage) Upload(_ context.Context, key string, reader io.Reader, _ string) (string, error) {
	if _, err := io.ReadAll(reader); err != nil {
		return "", err
	}
	s.uploadedKey = key
	return "https://storage.test/" + key, nil
}

func (s *recordingStorage) Delete(_ context.Context, key string) error {
	s.deletedKey = key
	return s.deleteErr
}

func TestUploadEvidencePreservesMetadataErrorWhenCleanupFails(t *testing.T) {
	metadataErr := errors.New("metadata creation failed")
	repo := &duplicateEvidenceRepository{createErr: metadataErr}
	store := &recordingStorage{deleteErr: errors.New("cleanup failed")}
	service := NewEvidenceService(repo, existingCaseRepository{}, store)

	_, err := service.UploadEvidence(context.Background(), UploadEvidenceRequest{CaseID: "case-id"}, bytes.NewBufferString("evidence"), "proof.pdf", 8, "user-id")
	if !errors.Is(err, metadataErr) {
		t.Fatalf("upload error = %v, want metadata error %v", err, metadataErr)
	}
}

func (s *recordingStorage) GetURL(key string) string { return "https://storage.test/" + key }

func (s *recordingStorage) Open(context.Context, string) (io.ReadCloser, error) { return nil, nil }

func (s *recordingStorage) Path(string) (string, error) { return "", nil }

func TestUploadEvidenceDuplicateDeletesNewObjectWithoutCreatingMetadata(t *testing.T) {
	existing := &models.Evidence{ID: "existing-evidence"}
	repo := &duplicateEvidenceRepository{existing: existing}
	store := &recordingStorage{}
	service := NewEvidenceService(repo, existingCaseRepository{}, store)

	got, err := service.UploadEvidence(context.Background(), UploadEvidenceRequest{CaseID: "case-id"}, bytes.NewBufferString("duplicate evidence"), "proof.pdf", 18, "user-id")
	if err != nil {
		t.Fatal(err)
	}
	if got != existing {
		t.Fatalf("evidence = %p, want existing %p", got, existing)
	}
	if store.uploadedKey == "" {
		t.Fatal("duplicate upload did not create an object")
	}
	if store.deletedKey != store.uploadedKey {
		t.Fatalf("deleted key = %q, want newly uploaded key %q", store.deletedKey, store.uploadedKey)
	}
	if repo.createCalls != 0 {
		t.Fatalf("metadata creates = %d, want 0", repo.createCalls)
	}
}

func TestDeleteEvidenceDeletesStoredTextEvidence(t *testing.T) {
	key := "subjects/subject-1/events/1/evidence/1.txt"
	repo := &duplicateEvidenceRepository{
		existing: &models.Evidence{
			ID:         "evidence-id",
			Type:       "text",
			StorageKey: &key,
		},
	}
	store := &recordingStorage{}
	service := NewEvidenceService(repo, existingCaseRepository{}, store)

	if err := service.DeleteEvidence(context.Background(), "evidence-id"); err != nil {
		t.Fatal(err)
	}
	if store.deletedKey != key {
		t.Fatalf("deleted key = %q, want %q", store.deletedKey, key)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("metadata deletes = %d, want 1", repo.deleteCalls)
	}
}
