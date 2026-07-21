package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

var (
	ErrCaseNotFound = errors.New("case not found")
)

// CaseService handles case business logic
type CaseService struct {
	caseRepo    caseServiceRepository
	subjectRepo caseSubjectRepository
	auditRepo   caseAuditRepository
	storage     storage.Storage
}

type caseServiceRepository interface {
	CreateCase(context.Context, *models.Case) error
	GetCaseByID(context.Context, string) (*models.Case, error)
	ListCases(context.Context, int, int, string, string) ([]models.Case, int64, error)
	UpdateCase(context.Context, *models.Case) error
	DeleteCase(context.Context, string) error
	ReviewCase(context.Context, string, string, string, string) error
	GetCasesBySubjectID(context.Context, string) ([]models.Case, error)
}

type caseSubjectRepository interface {
	GetSubjectByID(context.Context, string) (*models.Subject, error)
}

type caseAuditRepository interface {
	CreateAuditLog(context.Context, *models.AuditLog) error
	GetAuditLogsByResource(context.Context, string, string) ([]models.AuditLog, error)
}

// NewCaseService creates a new case service
func NewCaseService(
	caseRepo caseServiceRepository,
	subjectRepo caseSubjectRepository,
	auditRepo caseAuditRepository,
	storage storage.Storage,
) *CaseService {
	return &CaseService{
		caseRepo:    caseRepo,
		subjectRepo: subjectRepo,
		auditRepo:   auditRepo,
		storage:     storage,
	}
}

// CreateCaseRequest represents a case creation request
type CreateCaseRequest struct {
	SubjectID   string `json:"subject_id" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Severity    int    `json:"severity"`
}

// UpdateCaseRequest represents a case update request
type UpdateCaseRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    *int   `json:"severity"`
}

// ReviewCaseRequest represents a case review request
type ReviewCaseRequest struct {
	Status  string `json:"status" validate:"required,oneof=approved rejected"`
	Verdict string `json:"verdict"`
}

// CreateCase creates a new case
func (s *CaseService) CreateCase(ctx context.Context, req CreateCaseRequest, submittedBy string) (*models.Case, error) {
	// Verify subject exists
	_, err := s.subjectRepo.GetSubjectByID(ctx, req.SubjectID)
	if err != nil {
		return nil, fmt.Errorf("subject not found: %w", err)
	}

	c := &models.Case{
		SubjectID:   req.SubjectID,
		Title:       req.Title,
		Description: &req.Description,
		Status:      "pending",
		Severity:    req.Severity,
		SubmittedBy: &submittedBy,
	}

	if err := s.caseRepo.CreateCase(ctx, c); err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, submittedBy, "create", c.ID, nil)

	return c, nil
}

// GetCase retrieves a case by ID
func (s *CaseService) GetCase(ctx context.Context, id string) (*models.Case, error) {
	return s.caseRepo.GetCaseByID(ctx, id)
}

// ListCases lists cases with pagination
func (s *CaseService) ListCases(ctx context.Context, page, pageSize int, status, subjectID string) ([]models.Case, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.caseRepo.ListCases(ctx, offset, pageSize, status, subjectID)
}

// UpdateCase updates a case
func (s *CaseService) UpdateCase(ctx context.Context, id string, req UpdateCaseRequest, updatedBy string) (*models.Case, error) {
	c, err := s.caseRepo.GetCaseByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store old values for audit
	oldValues := map[string]interface{}{
		"title":       c.Title,
		"description": c.Description,
		"severity":    c.Severity,
	}

	if req.Title != "" {
		c.Title = req.Title
	}
	if req.Description != "" {
		c.Description = &req.Description
	}
	if req.Severity != nil {
		c.Severity = *req.Severity
	}

	if err := s.caseRepo.UpdateCase(ctx, c); err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, updatedBy, "update", c.ID, oldValues)

	return c, nil
}

// DeleteCase deletes a case
func (s *CaseService) DeleteCase(ctx context.Context, id, deletedBy string) error {
	c, err := s.caseRepo.GetCaseByID(ctx, id)
	if err != nil {
		return err
	}

	for _, evidence := range c.Evidences {
		if evidence.StorageKey == nil {
			continue
		}
		if err := s.storage.Delete(ctx, *evidence.StorageKey); err != nil {
			return fmt.Errorf("delete stored evidence: %w", err)
		}
	}

	if err := s.caseRepo.DeleteCase(ctx, id); err != nil {
		return err
	}

	// Create audit log
	s.createAuditLog(ctx, deletedBy, "delete", c.ID, nil)

	return nil
}

// ReviewCase reviews a case (approve or reject)
func (s *CaseService) ReviewCase(ctx context.Context, id string, req ReviewCaseRequest, reviewedBy string) (*models.Case, error) {
	c, err := s.caseRepo.GetCaseByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store old values for audit
	oldValues := map[string]interface{}{
		"status":      c.Status,
		"reviewed_by": c.ReviewedBy,
		"verdict":     c.Verdict,
	}

	if err := s.caseRepo.ReviewCase(ctx, id, reviewedBy, req.Status, req.Verdict); err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, reviewedBy, "review", c.ID, oldValues)

	// Reload case
	return s.caseRepo.GetCaseByID(ctx, id)
}

// GetCaseHistory returns audit logs for a case
func (s *CaseService) GetCaseHistory(ctx context.Context, caseID string) ([]models.AuditLog, error) {
	return s.auditRepo.GetAuditLogsByResource(ctx, "case", caseID)
}

// GetCasesBySubjectID returns all cases for a subject
func (s *CaseService) GetCasesBySubjectID(ctx context.Context, subjectID string) ([]models.Case, error) {
	return s.caseRepo.GetCasesBySubjectID(ctx, subjectID)
}

// createAuditLog creates an audit log entry for case resources.
func (s *CaseService) createAuditLog(ctx context.Context, userID, action, resourceID string, oldValues map[string]interface{}) {
	_ = s.auditRepo.CreateAuditLog(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: "case",
		ResourceID:   &resourceID,
		Changes:      oldValues,
	})
}
