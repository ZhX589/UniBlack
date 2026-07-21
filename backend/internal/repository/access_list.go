package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var (
	ErrAccessListNotFound = errors.New("access list entry not found")
)

// AccessListRepository handles access list database operations
type AccessListRepository struct {
	db *gorm.DB
}

// NewAccessListRepository creates a new access list repository
func NewAccessListRepository(db *gorm.DB) *AccessListRepository {
	return &AccessListRepository{db: db}
}

// CreateEntry creates a new access list entry
func (r *AccessListRepository) CreateEntry(ctx context.Context, entry *models.AccessList) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

// GetEntryByID retrieves an entry by ID
func (r *AccessListRepository) GetEntryByID(ctx context.Context, id string) (*models.AccessList, error) {
	var entry models.AccessList
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&entry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccessListNotFound
		}
		return nil, err
	}
	return &entry, nil
}

// ListEntries lists entries with pagination and filters
func (r *AccessListRepository) ListEntries(ctx context.Context, offset, limit int, listType, target string) ([]models.AccessList, int64, error) {
	var entries []models.AccessList
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AccessList{})
	if listType != "" {
		query = query.Where("type = ?", listType)
	}
	if target != "" {
		query = query.Where("target = ?", target)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&entries).Error

	return entries, total, err
}

// DeleteEntry deletes an entry
func (r *AccessListRepository) DeleteEntry(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.AccessList{})
	if result.RowsAffected == 0 {
		return ErrAccessListNotFound
	}
	return result.Error
}

// IsListed checks if a value is in a specific list
func (r *AccessListRepository) IsListed(ctx context.Context, listType, target, value string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AccessList{}).
		Where("type = ? AND target = ? AND value = ?", listType, target, value).
		Count(&count).Error
	return count > 0, err
}

// GetEntriesByType gets all entries of a specific type
func (r *AccessListRepository) GetEntriesByType(ctx context.Context, listType string) ([]models.AccessList, error) {
	var entries []models.AccessList
	err := r.db.WithContext(ctx).
		Where("type = ?", listType).
		Find(&entries).Error
	return entries, err
}
