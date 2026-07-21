package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

var (
	ErrAppealNotFound = errors.New("appeal not found")
)

// AppealService handles appeal business logic
type AppealService struct {
	appealRepo *repository.AppealRepository
	caseRepo   *repository.CaseRepository
	eventRepo  *repository.EventRepository
	auditRepo  *repository.AuditLogRepository
}

// NewAppealService creates a new appeal service
func NewAppealService(
	appealRepo *repository.AppealRepository,
	caseRepo *repository.CaseRepository,
	eventRepo *repository.EventRepository,
	auditRepo *repository.AuditLogRepository,
) *AppealService {
	return &AppealService{
		appealRepo: appealRepo,
		caseRepo:   caseRepo,
		eventRepo:  eventRepo,
		auditRepo:  auditRepo,
	}
}

// CreateAppealRequest represents an appeal creation request
type CreateAppealRequest struct {
	CaseID string `json:"case_id" validate:"required"`
	Reason string `json:"reason" validate:"required"`
}

// ReviewAppealRequest represents an appeal review request
type ReviewAppealRequest struct {
	Status      string `json:"status" validate:"required,oneof=approved rejected"`
	ReviewNotes string `json:"review_notes"`
}

type ResolveAppealRequest struct {
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}

// CreateAppeal creates a new appeal
func (s *AppealService) CreateAppeal(ctx context.Context, req CreateAppealRequest, submittedBy string) (*models.Appeal, error) {
	// Verify case exists
	caseObj, err := s.caseRepo.GetCaseByID(ctx, req.CaseID)
	if err != nil {
		return nil, err
	}

	// Only allow appeals on approved cases
	if caseObj.Status != "approved" && caseObj.Status != "closed" {
		return nil, errors.New("can only appeal approved or closed cases")
	}

	appeal := &models.Appeal{
		CaseID:      req.CaseID,
		Reason:      req.Reason,
		Status:      "pending",
		SubmittedBy: &submittedBy,
	}

	if err := s.appealRepo.CreateAppeal(ctx, appeal); err != nil {
		return nil, err
	}

	// Create audit log
	s.createAuditLog(ctx, submittedBy, "create", "appeal", appeal.ID, nil)

	return appeal, nil
}

// GetAppeal retrieves an appeal by ID
func (s *AppealService) GetAppeal(ctx context.Context, id string) (*models.Appeal, error) {
	return s.appealRepo.GetAppealByID(ctx, id)
}

// GetAppealsByCaseID retrieves all appeals for a case
func (s *AppealService) GetAppealsByCaseID(ctx context.Context, caseID string) ([]models.Appeal, error) {
	return s.appealRepo.GetAppealsByCaseID(ctx, caseID)
}

// ListAppeals lists appeals with pagination
func (s *AppealService) ListAppeals(ctx context.Context, page, pageSize int, status, submittedBy string) ([]models.Appeal, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.appealRepo.ListAppeals(ctx, offset, pageSize, status, submittedBy)
}

// ReviewAppeal reviews an appeal
func (s *AppealService) ReviewAppeal(ctx context.Context, id string, req ReviewAppealRequest, reviewedBy string) (*models.Appeal, error) {
	appeal, err := s.appealRepo.GetAppealByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update appeal status
	if err := s.appealRepo.ReviewAppeal(ctx, id, reviewedBy, req.Status, req.ReviewNotes); err != nil {
		return nil, err
	}

	// If appeal approved, update case status
	if req.Status == "approved" {
		s.caseRepo.UpdateCase(ctx, &models.Case{
			ID: appeal.CaseID,
		})
	}

	// Create audit log
	s.createAuditLog(ctx, reviewedBy, "review", "appeal", appeal.ID, nil)

	return s.appealRepo.GetAppealByID(ctx, id)
}

func (s *AppealService) ResolveAppeal(ctx context.Context, id string, req ResolveAppealRequest, reviewedBy string) (*models.Appeal, error) {
	if req.Outcome != "upheld" && req.Outcome != "corrected" && req.Outcome != "withdrawn" && req.Outcome != "malicious_submission" {
		return nil, errors.New("invalid appeal outcome")
	}
	appeal, err := s.appealRepo.GetAppealByID(ctx, id)
	if err != nil {
		return nil, err
	}
	status := "rejected"
	if req.Outcome == "corrected" || req.Outcome == "withdrawn" {
		status = "approved"
	}
	if err := s.appealRepo.ResolveAppeal(ctx, id, reviewedBy, status, req.Outcome, req.Reason); err != nil {
		return nil, err
	}
	if appeal.EventID != nil && s.eventRepo != nil {
		switch req.Outcome {
		case "corrected":
			if err := s.eventRepo.UpdateStatus(ctx, *appeal.EventID, "corrected", req.Reason); err != nil {
				return nil, err
			}
		case "withdrawn":
			if err := s.eventRepo.UpdateStatus(ctx, *appeal.EventID, "withdrawn", req.Reason); err != nil {
				return nil, err
			}
		}
	} else if req.Outcome == "withdrawn" {
		if err := s.caseRepo.ReviewCase(ctx, appeal.CaseID, reviewedBy, "closed", req.Reason); err != nil {
			return nil, err
		}
	}
	s.createAuditLog(ctx, reviewedBy, "resolve", "appeal", id, map[string]interface{}{"outcome": req.Outcome, "reason": req.Reason})
	return s.appealRepo.GetAppealByID(ctx, id)
}

// DeleteAppeal deletes an appeal
func (s *AppealService) DeleteAppeal(ctx context.Context, id, deletedBy string) error {
	if err := s.appealRepo.DeleteAppeal(ctx, id); err != nil {
		return err
	}

	s.createAuditLog(ctx, deletedBy, "delete", "appeal", id, nil)

	return nil
}

// createAuditLog creates an audit log entry
func (s *AppealService) createAuditLog(ctx context.Context, userID, action, resourceType, resourceID string, oldValues map[string]interface{}) {
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
