package service

import (
	"context"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

var (
	ErrSubjectNotFound  = errors.New("subject not found")
	ErrIdentifierExists = errors.New("identifier already exists")
)

// SubjectService handles subject business logic
type SubjectService struct {
	subjectRepo *repository.SubjectRepository
}

// NewSubjectService creates a new subject service
func NewSubjectService(subjectRepo *repository.SubjectRepository) *SubjectService {
	return &SubjectService{subjectRepo: subjectRepo}
}

// CreateSubjectRequest represents a subject creation request
type CreateSubjectRequest struct {
	DisplayName string              `json:"display_name" validate:"required"`
	Notes       string              `json:"notes"`
	RiskLevel   int                 `json:"risk_level"`
	Identifiers []IdentifierRequest `json:"identifiers"`
}

// IdentifierRequest represents an identifier request
type IdentifierRequest struct {
	Type      string `json:"type" validate:"required"`
	Value     string `json:"value" validate:"required"`
	IsPrimary bool   `json:"is_primary"`
}

// UpdateSubjectRequest represents a subject update request
type UpdateSubjectRequest struct {
	DisplayName string `json:"display_name"`
	Notes       string `json:"notes"`
	RiskLevel   *int   `json:"risk_level"`
	Status      string `json:"status"`
}

// CreateSubject creates a new subject with identifiers
func (s *SubjectService) CreateSubject(ctx context.Context, req CreateSubjectRequest, createdBy string) (*models.Subject, error) {
	subject := &models.Subject{
		DisplayName: req.DisplayName,
		Notes:       &req.Notes,
		RiskLevel:   req.RiskLevel,
		Status:      "active",
		CreatedBy:   &createdBy,
	}

	if err := s.subjectRepo.CreateSubject(ctx, subject); err != nil {
		return nil, err
	}

	// Add identifiers
	for _, idReq := range req.Identifiers {
		identifier := &models.Identifier{
			SubjectID: subject.ID,
			Type:      idReq.Type,
			Value:     idReq.Value,
			IsPrimary: idReq.IsPrimary,
		}
		if err := s.subjectRepo.AddIdentifier(ctx, identifier); err != nil {
			// Log error but continue
			continue
		}
	}

	// Reload with identifiers
	return s.subjectRepo.GetSubjectByID(ctx, subject.ID)
}

// GetSubject retrieves a subject by ID
func (s *SubjectService) GetSubject(ctx context.Context, id string) (*models.Subject, error) {
	return s.subjectRepo.GetSubjectByID(ctx, id)
}

// GetSubjectByIdentifier retrieves a subject by identifier
func (s *SubjectService) GetSubjectByIdentifier(ctx context.Context, idType, value string) (*models.Subject, error) {
	return s.subjectRepo.GetSubjectByIdentifier(ctx, idType, value)
}

// ListSubjects lists subjects with pagination
func (s *SubjectService) ListSubjects(ctx context.Context, page, pageSize int, status string) ([]models.Subject, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.subjectRepo.ListSubjects(ctx, offset, pageSize, status)
}

// UpdateSubject updates a subject
func (s *SubjectService) UpdateSubject(ctx context.Context, id string, req UpdateSubjectRequest) (*models.Subject, error) {
	subject, err := s.subjectRepo.GetSubjectByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.DisplayName != "" {
		subject.DisplayName = req.DisplayName
	}
	if req.Notes != "" {
		subject.Notes = &req.Notes
	}
	if req.RiskLevel != nil {
		subject.RiskLevel = *req.RiskLevel
	}
	if req.Status != "" {
		subject.Status = req.Status
	}

	if err := s.subjectRepo.UpdateSubject(ctx, subject); err != nil {
		return nil, err
	}

	return subject, nil
}

// DeleteSubject deletes a subject
func (s *SubjectService) DeleteSubject(ctx context.Context, id string) error {
	return s.subjectRepo.DeleteSubject(ctx, id)
}

// SearchSubjects searches subjects by query
func (s *SubjectService) SearchSubjects(ctx context.Context, query string) ([]models.Subject, error) {
	return s.subjectRepo.SearchSubjects(ctx, query)
}

// AddIdentifier adds an identifier to a subject
func (s *SubjectService) AddIdentifier(ctx context.Context, subjectID string, req IdentifierRequest) (*models.Identifier, error) {
	// Verify subject exists
	_, err := s.subjectRepo.GetSubjectByID(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	identifier := &models.Identifier{
		SubjectID: subjectID,
		Type:      req.Type,
		Value:     req.Value,
		IsPrimary: req.IsPrimary,
	}

	if err := s.subjectRepo.AddIdentifier(ctx, identifier); err != nil {
		return nil, err
	}

	return identifier, nil
}

// RemoveIdentifier removes an identifier
func (s *SubjectService) RemoveIdentifier(ctx context.Context, id string) error {
	return s.subjectRepo.RemoveIdentifier(ctx, id)
}

// GetIdentifiersBySubjectID retrieves all identifiers for a subject
func (s *SubjectService) GetIdentifiersBySubjectID(ctx context.Context, subjectID string) ([]models.Identifier, error) {
	return s.subjectRepo.GetIdentifiersBySubjectID(ctx, subjectID)
}
