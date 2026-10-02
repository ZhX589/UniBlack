package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/domain"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

type sanctionRepository interface {
	Create(context.Context, *models.Sanction) error
	Revoke(context.Context, string, string, string) error
	List(context.Context, int, int, string, bool) ([]models.Sanction, int64, error)
	CreateAppeal(context.Context, *models.SanctionAppeal) error
	ResolveAppealWithConsequences(context.Context, string, string, string, string) (*models.SanctionAppeal, error)
}

type SanctionService struct {
	repo sanctionRepository
}

func NewSanctionService(repo *repository.SanctionRepository, audit *repository.AuditLogRepository) *SanctionService {
	// Audit writes belong to repository transactions; retain the constructor
	// argument for compatibility with existing dependency wiring.
	return &SanctionService{repo: repo}
}

type CreateSanctionRequest struct {
	UserID          string     `json:"user_id"`
	Type            string     `json:"type"`
	Reason          string     `json:"reason"`
	EndsAt          *time.Time `json:"ends_at"`
	RelatedEventID  *string    `json:"related_event_id"`
	RelatedAppealID *string    `json:"related_appeal_id"`
}

func (s *SanctionService) Create(ctx context.Context, req CreateSanctionRequest, actor string) (*models.Sanction, error) {
	if req.UserID == "" || req.Reason == "" {
		return nil, fmt.Errorf("%w: %s", repository.ErrSanctionInvalidRequest, "user_id and reason required")
	}
	if req.Type != "warning" && req.Type != "submission_suspension" && req.Type != "submission_ban" {
		return nil, fmt.Errorf("%w: %s", repository.ErrSanctionInvalidRequest, "invalid sanction type")
	}
	if req.Type == "submission_suspension" && req.EndsAt == nil {
		return nil, fmt.Errorf("%w: %s", repository.ErrSanctionInvalidRequest, "suspension end required")
	}
	v := &models.Sanction{UserID: req.UserID, Type: req.Type, Reason: req.Reason, EndsAt: req.EndsAt, ImposedBy: actor, RelatedEventID: req.RelatedEventID, RelatedAppealID: req.RelatedAppealID, StartsAt: time.Now()}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *SanctionService) Revoke(ctx context.Context, id, actor, reason string) error {
	if reason == "" {
		return fmt.Errorf("%w: %s", repository.ErrSanctionInvalidRequest, "revoke reason required")
	}
	return s.repo.Revoke(ctx, id, actor, reason)
}

func (s *SanctionService) List(ctx context.Context, page, pageSize int, userID string, activeOnly bool) ([]models.Sanction, int64, error) {
	page, pageSize = domain.NormalizePagination(page, pageSize)
	return s.repo.List(ctx, (page-1)*pageSize, pageSize, userID, activeOnly)
}

func (s *SanctionService) Appeal(ctx context.Context, sanctionID, userID, reason string) (*models.SanctionAppeal, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("%w: %s", repository.ErrSanctionInvalidRequest, "appeal reason required")
	}
	appeal := &models.SanctionAppeal{
		SanctionID:  sanctionID,
		Reason:      reason,
		Status:      "pending",
		SubmittedBy: userID,
	}
	if err := s.repo.CreateAppeal(ctx, appeal); err != nil {
		return nil, err
	}
	return appeal, nil
}

func (s *SanctionService) ResolveAppeal(ctx context.Context, appealID, actor, status, notes string) (*models.SanctionAppeal, error) {
	if status != "approved" && status != "rejected" {
		return nil, fmt.Errorf("%w: %s", repository.ErrSanctionInvalidRequest, "status must be approved or rejected")
	}
	return s.repo.ResolveAppealWithConsequences(ctx, appealID, actor, status, notes)
}
