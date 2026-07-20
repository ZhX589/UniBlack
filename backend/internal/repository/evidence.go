package repository

import (
	"context"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrEvidenceNotFound = errors.New("evidence not found")
)

// EvidenceRepository handles evidence database operations
type EvidenceRepository struct {
	db *gorm.DB
}

// NewEvidenceRepository creates a new evidence repository
func NewEvidenceRepository(db *gorm.DB) *EvidenceRepository {
	return &EvidenceRepository{db: db}
}

// CreateEvidence creates a new evidence entry
func (r *EvidenceRepository) CreateEvidence(ctx context.Context, evidence *models.Evidence) error {
	return r.db.WithContext(ctx).Create(evidence).Error
}

// GetEvidenceByID retrieves evidence by ID
func (r *EvidenceRepository) GetEvidenceByID(ctx context.Context, id string) (*models.Evidence, error) {
	var evidence models.Evidence
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&evidence).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEvidenceNotFound
		}
		return nil, err
	}
	return &evidence, nil
}

// GetEvidenceByCaseID retrieves all evidence for a case
func (r *EvidenceRepository) GetEvidenceByCaseID(ctx context.Context, caseID string) ([]models.Evidence, error) {
	var evidences []models.Evidence
	err := r.db.WithContext(ctx).
		Where("case_id = ?", caseID).
		Order("created_at DESC").
		Find(&evidences).Error
	return evidences, err
}

// DeleteEvidence deletes evidence
func (r *EvidenceRepository) DeleteEvidence(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Evidence{})
	if result.RowsAffected == 0 {
		return ErrEvidenceNotFound
	}
	return result.Error
}

// GetEvidenceBySHA256 checks if evidence with same SHA256 exists
func (r *EvidenceRepository) GetEvidenceBySHA256(ctx context.Context, sha256 string) (*models.Evidence, error) {
	var evidence models.Evidence
	err := r.db.WithContext(ctx).
		Where("sha256 = ?", sha256).
		First(&evidence).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &evidence, nil
}
