package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

// EventVersionRepository manages append-only event version history.
type EventVersionRepository struct {
	db *gorm.DB
}

func NewEventVersionRepository(db *gorm.DB) *EventVersionRepository {
	return &EventVersionRepository{db: db}
}

// RecordVersion creates a new version snapshot for an event.
func (r *EventVersionRepository) RecordVersion(ctx context.Context, event *models.Event, changeType string) error {
	var maxVersion int
	if err := r.db.WithContext(ctx).
		Model(&models.EventVersion{}).
		Where("event_id = ?", event.ID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion).Error; err != nil {
		return err
	}

	v := &models.EventVersion{
		EventID:       event.ID,
		VersionNumber: maxVersion + 1,
		Title:         event.Title,
		Details:       event.Details,
		Severity:      event.Severity,
		Status:        event.Status,
		ChangeType:    changeType,
		ChangedBy:     event.SubmittedBy,
	}
	return r.db.WithContext(ctx).Create(v).Error
}

// ListByEvent returns all versions for an event, newest first.
func (r *EventVersionRepository) ListByEvent(ctx context.Context, eventID string) ([]models.EventVersion, error) {
	var versions []models.EventVersion
	err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Order("created_at DESC").
		Find(&versions).Error
	return versions, err
}
