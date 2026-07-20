package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
)

// SubmissionService handles submission business logic
type SubmissionService struct {
	submissionRepo *repository.SubmissionRepository
	subjectRepo    *repository.SubjectRepository
	caseRepo       *repository.CaseRepository
	auditRepo      *repository.AuditLogRepository
}

// NewSubmissionService creates a new submission service
func NewSubmissionService(
	submissionRepo *repository.SubmissionRepository,
	subjectRepo *repository.SubjectRepository,
	caseRepo *repository.CaseRepository,
	auditRepo *repository.AuditLogRepository,
) *SubmissionService {
	return &SubmissionService{
		submissionRepo: submissionRepo,
		subjectRepo:    subjectRepo,
		caseRepo:       caseRepo,
		auditRepo:      auditRepo,
	}
}

// CreateSubmissionRequest represents a submission creation request
type CreateSubmissionRequest struct {
	SubjectIdentifiers []SubjectIdentifier `json:"subject_identifiers" validate:"required"`
	Reason             string              `json:"reason" validate:"required"`
	SaveAsDraft        bool                `json:"save_as_draft"`
}

// SubjectIdentifier represents an identifier for a subject
type SubjectIdentifier struct {
	Platform    string `json:"platform"`
	AccountType string `json:"account_type"`
	Value       string `json:"value"`
}

// ReviewSubmissionRequest represents a submission review request
type ReviewSubmissionRequest struct {
	Status      string `json:"status" validate:"required,oneof=approved rejected"`
	ReviewNotes string `json:"review_notes"`
}

// CreateSubmission creates a new submission
func (s *SubmissionService) CreateSubmission(ctx context.Context, req CreateSubmissionRequest, submittedBy string) (*models.Submission, error) {
	// Convert identifiers to JSON
	identifiersJSON, _ := json.Marshal(req.SubjectIdentifiers)

	status := "pending"
	if req.SaveAsDraft {
		status = "draft"
	}

	submission := &models.Submission{
		SubjectIdentifiers: string(identifiersJSON),
		Reason:             req.Reason,
		Status:             status,
		SubmittedBy:        &submittedBy,
	}

	if err := s.submissionRepo.CreateSubmission(ctx, submission); err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, submittedBy, "create", "submission", submission.ID, nil)

	return submission, nil
}

// GetSubmission retrieves a submission by ID
func (s *SubmissionService) GetSubmission(ctx context.Context, id string) (*models.Submission, error) {
	return s.submissionRepo.GetSubmissionByID(ctx, id)
}

// ListSubmissions lists submissions with pagination
func (s *SubmissionService) ListSubmissions(ctx context.Context, page, pageSize int, status, submittedBy string) ([]models.Submission, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.submissionRepo.ListSubmissions(ctx, offset, pageSize, status, submittedBy)
}

// ReviewSubmission reviews a submission (approve or reject)
func (s *SubmissionService) ReviewSubmission(ctx context.Context, id string, req ReviewSubmissionRequest, reviewedBy string) (*models.Submission, error) {
	submission, err := s.submissionRepo.GetSubmissionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// If approved, create a case
	var caseID *string
	if req.Status == "approved" {
		// Parse identifiers to find or create subject
		// For simplicity, we'll create a new subject
		subject := &models.Subject{
			DisplayName: "待补充",
			Status:      "active",
			CreatedBy:   &reviewedBy,
		}
		if err := s.subjectRepo.CreateSubject(ctx, subject); err != nil {
			return nil, fmt.Errorf("failed to create subject: %w", err)
		}

		// Create case
		caseObj := &models.Case{
			SubjectID:   subject.ID,
			Title:       "来自举报",
			Description: &submission.Reason,
			Status:      "pending",
			SubmittedBy: submission.SubmittedBy,
		}
		if err := s.caseRepo.CreateCase(ctx, caseObj); err != nil {
			return nil, fmt.Errorf("failed to create case: %w", err)
		}
		caseID = &caseObj.ID
	}

	// Update submission
	if err := s.submissionRepo.ReviewSubmission(ctx, id, reviewedBy, req.Status, req.ReviewNotes, caseID); err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, reviewedBy, "review", "submission", submission.ID, nil)

	return s.submissionRepo.GetSubmissionByID(ctx, id)
}

// DeleteSubmission deletes a submission
func (s *SubmissionService) DeleteSubmission(ctx context.Context, id, deletedBy string) error {
	if err := s.submissionRepo.DeleteSubmission(ctx, id); err != nil {
		return err
	}

	s.createAuditLog(ctx, deletedBy, "delete", "submission", id, nil)

	return nil
}

// createAuditLog creates an audit log entry
func (s *SubmissionService) createAuditLog(ctx context.Context, userID, action, resourceType, resourceID string, oldValues map[string]interface{}) {
	changesJSON, _ := json.Marshal(oldValues)
	changes := make(map[string]interface{})
	json.Unmarshal(changesJSON, &changes)

	log := &models.AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Changes:      changes,
	}
	s.auditRepo.CreateAuditLog(ctx, log)
}
