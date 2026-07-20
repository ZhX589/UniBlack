package repository

import (
	"context"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
)

var ErrEventNotFound = errors.New("event not found")

type EventRepository struct{ db *gorm.DB }

func NewEventRepository(db *gorm.DB) *EventRepository { return &EventRepository{db: db} }

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
		audit.ResourceID = &subject.ID
		return tx.Create(audit).Error
	})
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
