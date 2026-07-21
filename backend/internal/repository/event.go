package repository

import (
	"context"
	"errors"
	"io"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

var ErrEventNotFound = errors.New("event not found")

type EventRepository struct {
	db      *gorm.DB
	storage storage.Storage
}

func NewEventRepository(db *gorm.DB, stores ...storage.Storage) *EventRepository {
	repo := &EventRepository{db: db}
	if len(stores) > 0 {
		repo.storage = stores[0]
	}
	return repo
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	var event models.Event
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrEventNotFound
	}
	return &event, err
}

func (r *EventRepository) ListBySubject(ctx context.Context, subjectID string) ([]models.Event, error) {
	var events []models.Event
	err := r.db.WithContext(ctx).Where("subject_id = ?", subjectID).Order("created_at DESC").Find(&events).Error
	return events, err
}

func (r *EventRepository) Publish(ctx context.Context, subject *models.Subject, accounts []models.Account, events []models.Event, audit *models.AuditLog) error {
	return r.PublishWithEvidence(ctx, subject, accounts, events, nil, audit)
}

type EventEvidence struct {
	EventIndex int
	Evidence   models.Evidence
}

func (r *EventRepository) PublishWithEvidence(ctx context.Context, subject *models.Subject, accounts []models.Account, events []models.Event, evidence []EventEvidence, audit *models.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(subject).Error; err != nil {
			return err
		}
		for i := range accounts {
			accounts[i].SubjectID = subject.ID
			if err := tx.Create(&accounts[i]).Error; err != nil {
				return err
			}
		}
		for i := range events {
			events[i].SubjectID = subject.ID
			if err := tx.Create(&events[i]).Error; err != nil {
				return err
			}
		}
		for i := range evidence {
			if evidence[i].EventIndex < 0 || evidence[i].EventIndex >= len(events) {
				return errors.New("evidence event index missing")
			}
			eventID := events[evidence[i].EventIndex].ID
			evidence[i].Evidence.EventID = &eventID
			if err := tx.Create(&evidence[i].Evidence).Error; err != nil {
				return err
			}
		}
		audit.ResourceID = &subject.ID
		return tx.Create(audit).Error
	})
}

func (r *EventRepository) StoreText(ctx context.Context, key string, body io.Reader) (string, error) {
	return r.StoreBlob(ctx, key, body, "text/plain; charset=utf-8")
}

func (r *EventRepository) StoreBlob(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	if r.storage == nil {
		return "", errors.New("event storage unavailable")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return r.storage.Upload(ctx, key, body, contentType)
}

func (r *EventRepository) DeleteStored(ctx context.Context, key string) error {
	if r.storage == nil {
		return nil
	}
	return r.storage.Delete(ctx, key)
}

func (r *EventRepository) UpdateStatus(ctx context.Context, id, status, note string) error {
	updates := map[string]interface{}{"status": status}
	if note != "" {
		updates["correction_note"] = note
	}
	result := r.db.WithContext(ctx).Model(&models.Event{}).Where("id = ?", id).Updates(updates)
	if result.RowsAffected == 0 {
		return ErrEventNotFound
	}
	return result.Error
}
