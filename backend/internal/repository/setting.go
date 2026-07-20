package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrSettingNotFound = errors.New("setting not found")
)

// SystemSettingRepository handles system setting database operations
type SystemSettingRepository struct {
	db *gorm.DB
}

// NewSystemSettingRepository creates a new system setting repository
func NewSystemSettingRepository(db *gorm.DB) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

// GetSetting retrieves a setting by key
func (r *SystemSettingRepository) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	var setting models.SystemSetting
	err := r.db.WithContext(ctx).
		Where("key = ?", key).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSettingNotFound
		}
		return nil, err
	}
	return &setting, nil
}

// GetSettingValue retrieves a setting value by key and unmarshals it
func (r *SystemSettingRepository) GetSettingValue(ctx context.Context, key string, dest interface{}) error {
	setting, err := r.GetSetting(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(setting.Value), dest)
}

// GetAllSettings retrieves all settings
func (r *SystemSettingRepository) GetAllSettings(ctx context.Context) ([]models.SystemSetting, error) {
	var settings []models.SystemSetting
	err := r.db.WithContext(ctx).
		Order("key ASC").
		Find(&settings).Error
	return settings, err
}

// SetSetting creates or updates a setting
func (r *SystemSettingRepository) SetSetting(ctx context.Context, key string, value interface{}, description *string, updatedBy *string) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Try to find existing setting
	existing, err := r.GetSetting(ctx, key)
	if err != nil && err != ErrSettingNotFound {
		return err
	}

	if existing != nil {
		// Update existing
		existing.Value = string(valueJSON)
		if description != nil {
			existing.Description = description
		}
		existing.UpdatedBy = updatedBy
		return r.db.WithContext(ctx).Save(existing).Error
	}

	// Create new
	setting := &models.SystemSetting{
		Key:         key,
		Value:       string(valueJSON),
		Description: description,
		UpdatedBy:   updatedBy,
	}
	return r.db.WithContext(ctx).Create(setting).Error
}

// DeleteSetting deletes a setting
func (r *SystemSettingRepository) DeleteSetting(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).
		Where("key = ?", key).
		Delete(&models.SystemSetting{}).Error
}

// IsInitialized checks if the system has been initialized
func (r *SystemSettingRepository) IsInitialized(ctx context.Context) bool {
	var value bool
	err := r.GetSettingValue(ctx, "system.initialized", &value)
	return err == nil && value
}
