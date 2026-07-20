package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

var (
	ErrEvidenceNotFound = errors.New("evidence not found")
)

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
		CaseID:      req.CaseID,
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
		CaseID:      req.CaseID,
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
