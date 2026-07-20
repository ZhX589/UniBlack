package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

var (
	ErrEvidenceNotFound = errors.New("evidence not found")
)

const MaxTextEvidenceBytes = 200 * 1024

func validateTextEvidence(text string) error {
	if len(text) == 0 || len(text) > MaxTextEvidenceBytes {
		return fmt.Errorf("text evidence must be between 1 and %d bytes", MaxTextEvidenceBytes)
	}
	if !utf8.ValidString(text) {
		return errors.New("text evidence must be valid UTF-8")
	}
	return nil
}

// EvidenceService handles evidence business logic
type EvidenceService struct {
	evidenceRepo *repository.EvidenceRepository
	caseRepo     *repository.CaseRepository
	storage      storage.Storage
}

// NewEvidenceService creates a new evidence service
func NewEvidenceService(
	evidenceRepo *repository.EvidenceRepository,
	caseRepo *repository.CaseRepository,
	storage storage.Storage,
) *EvidenceService {
	return &EvidenceService{
		evidenceRepo: evidenceRepo,
		caseRepo:     caseRepo,
		storage:      storage,
	}
}

// CreateEvidenceRequest represents an evidence creation request
type CreateEvidenceRequest struct {
	CaseID      string `json:"case_id" validate:"required"`
	Type        string `json:"type" validate:"required,oneof=image file link text"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// UploadEvidenceRequest represents a file upload request
type UploadEvidenceRequest struct {
	CaseID      string `json:"case_id" validate:"required"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// CreateEvidence creates a new evidence entry (for links and text)
func (s *EvidenceService) CreateEvidence(ctx context.Context, req CreateEvidenceRequest, uploadedBy string) (*models.Evidence, error) {
	// Verify case exists
	_, err := s.caseRepo.GetCaseByID(ctx, req.CaseID)
	if err != nil {
		return nil, fmt.Errorf("case not found: %w", err)
	}

	evidence := &models.Evidence{
		CaseID:      &req.CaseID,
		Type:        req.Type,
		Title:       &req.Title,
		Description: &req.Description,
		URL:         &req.URL,
		UploadedBy:  &uploadedBy,
	}

	if err := s.evidenceRepo.CreateEvidence(ctx, evidence); err != nil {
		return nil, err
	}

	return evidence, nil
}

// CreateEventTextEvidence stores bounded UTF-8 text as a real archive file.
// It intentionally does not replace the legacy case-based CreateEvidence API.
func (s *EvidenceService) CreateEventTextEvidence(ctx context.Context, eventID, subjectPublicID string, eventNumber, evidenceNumber int, text, title, uploadedBy string) (*models.Evidence, error) {
	if err := validateTextEvidence(text); err != nil {
		return nil, err
	}
	key := storage.BuildEvidenceKey(subjectPublicID, eventNumber, evidenceNumber, ".txt")
	content := []byte(text)
	sum := sha256.Sum256(content)
	if _, err := s.storage.Upload(ctx, key, bytes.NewReader(content), "text/plain; charset=utf-8"); err != nil {
		return nil, fmt.Errorf("store text evidence: %w", err)
	}
	size := int64(len(content))
	hash := hex.EncodeToString(sum[:])
	mime := "text/plain"
	original := filepath.Base(key)
	evidence := &models.Evidence{
		EventID:          &eventID,
		Type:             "text",
		Title:            &title,
		StorageKey:       &key,
		OriginalFilename: &original,
		FileSize:         &size,
		SHA256:           &hash,
		MimeType:         &mime,
		UploadedBy:       &uploadedBy,
	}
	if err := s.evidenceRepo.CreateEvidence(ctx, evidence); err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}
	return evidence, nil
}

// UploadEvidence uploads a file and creates evidence entry
func (s *EvidenceService) UploadEvidence(ctx context.Context, req UploadEvidenceRequest, file io.Reader, fileName string, fileSize int64, uploadedBy string) (*models.Evidence, error) {
	// Verify case exists
	_, err := s.caseRepo.GetCaseByID(ctx, req.CaseID)
	if err != nil {
		return nil, fmt.Errorf("case not found: %w", err)
	}

	// Calculate SHA256
	hasher := sha256.New()
	teeReader := io.TeeReader(file, hasher)

	// Generate unique key
	ext := filepath.Ext(fileName)
	key := fmt.Sprintf("evidence/%s/%d%s", req.CaseID, time.Now().UnixNano(), ext)

	// Upload file
	url, err := s.storage.Upload(ctx, key, teeReader, getContentType(ext))
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	sha256Hash := hex.EncodeToString(hasher.Sum(nil))

	// Check for duplicate
	existing, _ := s.evidenceRepo.GetEvidenceBySHA256(ctx, sha256Hash)
	if existing != nil {
		return existing, nil
	}

	// Create evidence entry
	evidence := &models.Evidence{
		CaseID:      &req.CaseID,
		Type:        "file",
		Title:       &req.Title,
		Description: &req.Description,
		URL:         &url,
		FileSize:    &fileSize,
		SHA256:      &sha256Hash,
		MimeType:    &ext,
		UploadedBy:  &uploadedBy,
	}

	if err := s.evidenceRepo.CreateEvidence(ctx, evidence); err != nil {
		return nil, err
	}

	return evidence, nil
}

// GetEvidence retrieves evidence by ID
func (s *EvidenceService) GetEvidence(ctx context.Context, id string) (*models.Evidence, error) {
	return s.evidenceRepo.GetEvidenceByID(ctx, id)
}

// GetEvidenceByCaseID retrieves all evidence for a case
func (s *EvidenceService) GetEvidenceByCaseID(ctx context.Context, caseID string) ([]models.Evidence, error) {
	return s.evidenceRepo.GetEvidenceByCaseID(ctx, caseID)
}

// DeleteEvidence deletes evidence
func (s *EvidenceService) DeleteEvidence(ctx context.Context, id string) error {
	evidence, err := s.evidenceRepo.GetEvidenceByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from storage if it's a file
	if evidence.URL != nil && evidence.Type == "file" {
		s.storage.Delete(ctx, *evidence.URL)
	}

	return s.evidenceRepo.DeleteEvidence(ctx, id)
}

func getContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
