package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var (
	ErrCaseNotFound = errors.New("case not found")
)

// CaseRepository handles case database operations
type CaseRepository struct {
	db *gorm.DB
}

// NewCaseRepository creates a new case repository
func NewCaseRepository(db *gorm.DB) *CaseRepository {
	return &CaseRepository{db: db}
}

// CreateCase creates a new case
func (r *CaseRepository) CreateCase(ctx context.Context, c *models.Case) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		// Increment case count on subject
		return tx.Model(&models.Subject{}).
			Where("id = ?", c.SubjectID).
			Update("case_count", gorm.Expr("case_count + 1")).Error
	})
}

// GetCaseByID retrieves a case by ID
func (r *CaseRepository) GetCaseByID(ctx context.Context, id string) (*models.Case, error) {
	var c models.Case
	err := r.db.WithContext(ctx).
		Preload("Evidences").
		Where("id = ?", id).
		First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCaseNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ListCases lists cases with pagination and filters
func (r *CaseRepository) ListCases(ctx context.Context, offset, limit int, status string, subjectID string) ([]models.Case, int64, error) {
	var cases []models.Case
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Case{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Evidences").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&cases).Error

	return cases, total, err
}

// UpdateCase updates a case
func (r *CaseRepository) UpdateCase(ctx context.Context, c *models.Case) error {
	return r.db.WithContext(ctx).Save(c).Error
}

// DeleteCase deletes a case
func (r *CaseRepository) DeleteCase(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete evidences first
		if err := tx.Where("case_id = ?", id).Delete(&models.Evidence{}).Error; err != nil {
			return err
		}
		// Delete case
		result := tx.Where("id = ?", id).Delete(&models.Case{})
		if result.RowsAffected == 0 {
			return ErrCaseNotFound
		}
		return result.Error
	})
}

// ReviewCase updates case status after review
func (r *CaseRepository) ReviewCase(ctx context.Context, id, reviewedBy, status, verdict string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":      status,
		"reviewed_by": reviewedBy,
		"reviewed_at": now,
	}
	if verdict != "" {
		updates["verdict"] = verdict
	}
	if status == "closed" {
		updates["closed_at"] = now
	}

	return r.db.WithContext(ctx).
		Model(&models.Case{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetCasesBySubjectID retrieves all cases for a subject
func (r *CaseRepository) GetCasesBySubjectID(ctx context.Context, subjectID string) ([]models.Case, error) {
	var cases []models.Case
	err := r.db.WithContext(ctx).
		Preload("Evidences").
		Where("subject_id = ?", subjectID).
		Order("created_at DESC").
		Find(&cases).Error
	return cases, err
}
