package repository

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
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

// LoadRawMap returns key → raw JSON value for all rows (for OptionMap merge).
func (r *SystemSettingRepository) LoadRawMap(ctx context.Context) (map[string]string, error) {
	settings, err := r.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(settings))
	for _, s := range settings {
		out[s.Key] = s.Value
	}
	return out, nil
}

// SetRawJSON upserts a pre-encoded JSON value string.
func (r *SystemSettingRepository) SetRawJSON(ctx context.Context, key, valueJSON string, description *string, updatedBy *string) error {
	existing, err := r.GetSetting(ctx, key)
	if err != nil && err != ErrSettingNotFound {
		return err
	}
	if existing != nil {
		existing.Value = valueJSON
		if description != nil {
			existing.Description = description
		}
		existing.UpdatedBy = updatedBy
		return r.db.WithContext(ctx).Save(existing).Error
	}
	setting := &models.SystemSetting{
		Key:         key,
		Value:       valueJSON,
		Description: description,
		UpdatedBy:   updatedBy,
	}
	return r.db.WithContext(ctx).Create(setting).Error
}
