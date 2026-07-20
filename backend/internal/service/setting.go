package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

var (
	ErrSettingNotFound = errors.New("setting not found")
)

// SystemSettingService handles system setting business logic
type SystemSettingService struct {
	settingRepo    *repository.SystemSettingRepository
	accessListRepo *repository.AccessListRepository
	auditRepo      *repository.AuditLogRepository
}

// NewSystemSettingService creates a new system setting service
func NewSystemSettingService(
	settingRepo *repository.SystemSettingRepository,
	accessListRepo *repository.AccessListRepository,
	auditRepo *repository.AuditLogRepository,
) *SystemSettingService {
	return &SystemSettingService{
		settingRepo:    settingRepo,
		accessListRepo: accessListRepo,
		auditRepo:      auditRepo,
	}
}

// GetSetting retrieves a setting by key
func (s *SystemSettingService) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	return s.settingRepo.GetSetting(ctx, key)
}

// GetSettingValue retrieves a setting value by key
func (s *SystemSettingService) GetSettingValue(ctx context.Context, key string, dest interface{}) error {
	return s.settingRepo.GetSettingValue(ctx, key, dest)
}

// GetAllSettings retrieves all settings grouped by category
func (s *SystemSettingService) GetAllSettings(ctx context.Context) (map[string][]models.SystemSetting, error) {
	settings, err := s.settingRepo.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]models.SystemSetting)
	for _, setting := range settings {
		// Extract category from key (e.g., "site.name" -> "site")
		category := "other"
		for i, c := range setting.Key {
			if c == '.' {
				category = setting.Key[:i]
				break
			}
		}
		grouped[category] = append(grouped[category], setting)
	}

	return grouped, nil
}

// UpdateSettingRequest represents a setting update request
type UpdateSettingRequest struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// UpdateSettings updates multiple settings
func (s *SystemSettingService) UpdateSettings(ctx context.Context, settings []UpdateSettingRequest, updatedBy string) error {
	for _, setting := range settings {
		if err := s.settingRepo.SetSetting(ctx, setting.Key, setting.Value, nil, &updatedBy); err != nil {
			return err
		}
	}

	// Create audit log
	s.createAuditLog(ctx, updatedBy, "update", "system_settings", nil, nil)

	return nil
}

// GetSiteConfig returns site configuration
func (s *SystemSettingService) GetSiteConfig(ctx context.Context) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// Get site settings
	siteKeys := []string{"site.name", "site.description", "site.theme_color", "site.logo_url", "site.contact_email"}
	for _, key := range siteKeys {
		var value interface{}
		if err := s.settingRepo.GetSettingValue(ctx, key, &value); err == nil {
			config[key] = value
		}
	}

	return config, nil
}

// GetPublicSettings returns settings that are safe to expose publicly
func (s *SystemSettingService) GetPublicSettings(ctx context.Context) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// Site settings
	siteKeys := []string{"site.name", "site.description", "site.theme_color", "site.logo_url"}
	for _, key := range siteKeys {
		var value interface{}
		if err := s.settingRepo.GetSettingValue(ctx, key, &value); err == nil {
			config[key] = value
		}
	}

	// Security settings (without secrets)
	var captchaEnabled bool
	s.settingRepo.GetSettingValue(ctx, "security.captcha_enabled", &captchaEnabled)
	config["security.captcha_enabled"] = captchaEnabled

	var captchaProvider string
	s.settingRepo.GetSettingValue(ctx, "security.captcha_provider", &captchaProvider)
	config["security.captcha_provider"] = captchaProvider

	var captchaSiteKey string
	s.settingRepo.GetSettingValue(ctx, "security.captcha_site_key", &captchaSiteKey)
	config["security.captcha_site_key"] = captchaSiteKey

	var registrationEnabled bool
	s.settingRepo.GetSettingValue(ctx, "auth.registration_enabled", &registrationEnabled)
	config["auth.registration_enabled"] = registrationEnabled

	// Email verification setting
	var emailVerification bool
	s.settingRepo.GetSettingValue(ctx, "security.email_verification", &emailVerification)
	config["security.email_verification"] = emailVerification

	return config, nil
}

// IsInitialized checks if the system has been initialized
func (s *SystemSettingService) IsInitialized(ctx context.Context) bool {
	return s.settingRepo.IsInitialized(ctx)
}

// InitializeSystem initializes the system with admin password
func (s *SystemSettingService) InitializeSystem(ctx context.Context, adminPassword string) error {
	// This will be called by auth service to set admin password
	// Mark system as initialized
	return s.settingRepo.SetSetting(ctx, "system.initialized", true, nil, nil)
}

// AccessList methods

// CreateAccessListEntry creates a new access list entry
func (s *SystemSettingService) CreateAccessListEntry(ctx context.Context, entry *models.AccessList, createdBy string) error {
	entry.CreatedBy = &createdBy
	return s.accessListRepo.CreateEntry(ctx, entry)
}

// ListAccessListEntries lists access list entries
func (s *SystemSettingService) ListAccessListEntries(ctx context.Context, page, pageSize int, listType, target string) ([]models.AccessList, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.accessListRepo.ListEntries(ctx, offset, pageSize, listType, target)
}

// DeleteAccessListEntry deletes an access list entry
func (s *SystemSettingService) DeleteAccessListEntry(ctx context.Context, id string) error {
	return s.accessListRepo.DeleteEntry(ctx, id)
}

// IsListed checks if a value is in a specific list
func (s *SystemSettingService) IsListed(ctx context.Context, listType, target, value string) (bool, error) {
	return s.accessListRepo.IsListed(ctx, listType, target, value)
}

// createAuditLog creates an audit log entry
func (s *SystemSettingService) createAuditLog(ctx context.Context, userID, action, resourceType string, resourceID *string, oldValues map[string]interface{}) {
	changesJSON, _ := json.Marshal(oldValues)
	changes := make(map[string]interface{})
	json.Unmarshal(changesJSON, &changes)

	log := &models.AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      changes,
	}
	s.auditRepo.CreateAuditLog(ctx, log)
}
