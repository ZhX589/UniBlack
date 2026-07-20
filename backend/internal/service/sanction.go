package service

import (
	"context"
	"errors"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"time"
)

type SanctionService struct {
	repo *repository.SanctionRepository
}

func NewSanctionService(repo *repository.SanctionRepository) *SanctionService {
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
		return nil, errors.New("user_id and reason required")
	}
	if req.Type != "warning" && req.Type != "submission_suspension" && req.Type != "submission_ban" {
		return nil, errors.New("invalid sanction type")
	}
	if req.Type == "submission_suspension" && req.EndsAt == nil {
		return nil, errors.New("suspension end required")
	}
	v := &models.Sanction{UserID: req.UserID, Type: req.Type, Reason: req.Reason, EndsAt: req.EndsAt, ImposedBy: actor, RelatedEventID: req.RelatedEventID, RelatedAppealID: req.RelatedAppealID, StartsAt: time.Now()}
	return v, s.repo.Create(ctx, v)
}
func (s *SanctionService) Revoke(ctx context.Context, id, actor, reason string) error {
	return s.repo.Revoke(ctx, id, actor, reason)
}
