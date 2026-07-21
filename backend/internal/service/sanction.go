package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

type SanctionService struct {
	repo  *repository.SanctionRepository
	audit *repository.AuditLogRepository
}

func NewSanctionService(repo *repository.SanctionRepository, audit *repository.AuditLogRepository) *SanctionService {
	return &SanctionService{repo: repo, audit: audit}
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
		return nil, errors.New("user_id and reason required")
	}
	if req.Type != "warning" && req.Type != "submission_suspension" && req.Type != "submission_ban" {
		return nil, errors.New("invalid sanction type")
	}
	if req.Type == "submission_suspension" && req.EndsAt == nil {
		return nil, errors.New("suspension end required")
	}
	v := &models.Sanction{UserID: req.UserID, Type: req.Type, Reason: req.Reason, EndsAt: req.EndsAt, ImposedBy: actor, RelatedEventID: req.RelatedEventID, RelatedAppealID: req.RelatedAppealID, StartsAt: time.Now()}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.CreateAuditLog(ctx, &models.AuditLog{UserID: &actor, Action: "create", ResourceType: "sanction", ResourceID: &v.ID, Changes: map[string]interface{}{"type": v.Type, "user_id": v.UserID, "reason": v.Reason}})
	}
	return v, nil
}
func (s *SanctionService) Revoke(ctx context.Context, id, actor, reason string) error {
	if reason == "" {
		return errors.New("revoke reason required")
	}
	if err := s.repo.Revoke(ctx, id, actor, reason); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.CreateAuditLog(ctx, &models.AuditLog{UserID: &actor, Action: "revoke", ResourceType: "sanction", ResourceID: &id, Changes: map[string]interface{}{"reason": reason}})
	}
	return nil
}

func (s *SanctionService) List(ctx context.Context, page, pageSize int, userID string, activeOnly bool) ([]models.Sanction, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, (page-1)*pageSize, pageSize, userID, activeOnly)
}

func (s *SanctionService) Appeal(ctx context.Context, sanctionID, userID, reason string) (*models.SanctionAppeal, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("appeal reason required")
	}
	sanction, err := s.repo.GetByID(ctx, sanctionID)
	if err != nil {
		return nil, err
	}
	if sanction.UserID != userID {
		return nil, errors.New("can only appeal own sanctions")
	}
	if sanction.RevokedAt != nil {
		return nil, errors.New("sanction already revoked")
	}
	existing, err := s.repo.GetAppealBySanctionID(ctx, sanctionID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("sanction already appealed")
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
	if s.audit != nil {
		_ = s.audit.CreateAuditLog(ctx, &models.AuditLog{
			UserID: &userID, Action: "appeal", ResourceType: "sanction", ResourceID: &sanctionID,
			Changes: map[string]interface{}{"appeal_id": appeal.ID, "reason": reason},
		})
	}
	return appeal, nil
}

func (s *SanctionService) ResolveAppeal(ctx context.Context, appealID, actor, status, notes string) (*models.SanctionAppeal, error) {
	if status != "approved" && status != "rejected" {
		return nil, errors.New("status must be approved or rejected")
	}
	appeal, err := s.repo.GetAppealByID(ctx, appealID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ResolveAppeal(ctx, appealID, actor, status, notes); err != nil {
		return nil, err
	}
	// Approving a sanction appeal revokes the underlying sanction.
	if status == "approved" {
		if err := s.repo.Revoke(ctx, appeal.SanctionID, actor, "sanction appeal approved: "+notes); err != nil && !errors.Is(err, repository.ErrSanctionNotFound) {
			return nil, err
		}
	}
	if s.audit != nil {
		_ = s.audit.CreateAuditLog(ctx, &models.AuditLog{
			UserID: &actor, Action: "resolve_appeal", ResourceType: "sanction_appeal", ResourceID: &appealID,
			Changes: map[string]interface{}{"status": status, "notes": notes, "sanction_id": appeal.SanctionID},
		})
	}
	return s.repo.GetAppealByID(ctx, appealID)
}
