package service

import (
	"context"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

var (
	ErrAppealNotFound = repository.ErrAppealNotFound
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
	EventID string `json:"event_id"`
	// CaseID is a legacy compatibility adapter. New clients must use EventID.
	CaseID string `json:"case_id"`
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
	if (req.EventID == "") == (req.CaseID == "") {
		return nil, errors.New("exactly one of event_id or legacy case_id is required")
	}
	appeal := &models.Appeal{
		Reason:      req.Reason,
		Status:      "pending",
		SubmittedBy: &submittedBy,
	}
	if req.EventID != "" {
		if s.eventRepo == nil {
			return nil, errors.New("event appeals unavailable")
		}
		event, err := s.eventRepo.GetByID(ctx, req.EventID)
		if err != nil {
			return nil, err
		}
		if event.Status != "published" && event.Status != "corrected" {
			return nil, errors.New("can only appeal published events")
		}
		appeal.EventID = &req.EventID
	} else {
		caseObj, err := s.caseRepo.GetCaseByID(ctx, req.CaseID)
		if err != nil {
			return nil, err
		}
		if caseObj.Status != "approved" && caseObj.Status != "closed" {
			return nil, errors.New("can only appeal approved or closed cases")
		}
		appeal.CaseID = &req.CaseID
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

// GetAppealsByEventID is the canonical Event-first history read.
func (s *AppealService) GetAppealsByEventID(ctx context.Context, eventID string) ([]models.Appeal, error) {
	return s.appealRepo.GetAppealsByEventID(ctx, eventID)
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

	// If appeal approved, update case status (legacy Case path only).
	if req.Status == "approved" {
		if appeal.CaseID != nil {
			if err := s.caseRepo.UpdateCase(ctx, &models.Case{ID: *appeal.CaseID}); err != nil {
				return nil, err
			}
		}
	}

	// Create audit log
	s.createAuditLog(ctx, reviewedBy, "review", "appeal", appeal.ID, nil)

	return s.appealRepo.GetAppealByID(ctx, id)
}

func (s *AppealService) ResolveAppeal(ctx context.Context, id string, req ResolveAppealRequest, reviewedBy string) (*models.Appeal, error) {
	if req.Outcome != "upheld" && req.Outcome != "corrected" && req.Outcome != "withdrawn" && req.Outcome != "malicious_submission" {
		return nil, errors.New("invalid appeal outcome")
	}
	status := "rejected"
	if req.Outcome == "corrected" || req.Outcome == "withdrawn" {
		status = "approved"
	}
	if err := s.appealRepo.ResolveWithConsequences(ctx, id, reviewedBy, status, req.Outcome, req.Reason); err != nil {
		return nil, err
	}
	return s.appealRepo.GetAppealByID(ctx, id)
}

// DeleteAppeal retires an appeal from active reads and writes an audit row atomically.
func (s *AppealService) DeleteAppeal(ctx context.Context, id, deletedBy string) error {
	return s.appealRepo.DeleteAppeal(ctx, id, deletedBy)
}

// createAuditLog creates an audit log entry
func (s *AppealService) createAuditLog(ctx context.Context, userID, action, resourceType, resourceID string, oldValues map[string]interface{}) {
	_ = s.auditRepo.CreateAuditLog(ctx, &models.AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Changes:      oldValues,
	})
}
