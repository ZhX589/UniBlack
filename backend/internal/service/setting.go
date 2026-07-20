package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/setting"
)

var (
	ErrSettingNotFound = errors.New("setting not found")
)

// SystemSettingService handles system settings (NewAPI OptionMap style).
// Layer: env defaults → memory cache → DB override → console/API.
type SystemSettingService struct {
	settingRepo    *repository.SystemSettingRepository
	accessListRepo *repository.AccessListRepository
	auditRepo      *repository.AuditLogRepository
	cache          *setting.Cache
}

// NewSystemSettingService creates a service and loads options from DB onto env defaults.
func NewSystemSettingService(
	settingRepo *repository.SystemSettingRepository,
	accessListRepo *repository.AccessListRepository,
	auditRepo *repository.AuditLogRepository,
) *SystemSettingService {
	s := &SystemSettingService{
		settingRepo:    settingRepo,
		accessListRepo: accessListRepo,
		auditRepo:      auditRepo,
		cache:          setting.NewCache(),
	}
	return s
}

// Bootstrap merges DB into cache and ensures default keys exist in DB (idempotent).
func (s *SystemSettingService) Bootstrap(ctx context.Context) error {
	raw, err := s.settingRepo.LoadRawMap(ctx)
	if err != nil {
		log.Printf("Warning: load settings from DB: %v", err)
		raw = map[string]string{}
	}
	// Ensure every catalog/default key has a DB row (migration may be older)
	for k, v := range setting.DefaultMap() {
		if _, ok := raw[k]; !ok {
			if err := s.settingRepo.SetRawJSON(ctx, k, v, descFor(k), nil); err != nil {
				log.Printf("Warning: seed setting %s: %v", k, err)
				continue
			}
			raw[k] = v
		}
	}
	s.cache.Merge(raw)
	return nil
}

func descFor(key string) *string {
	if m := setting.Meta(key); m != nil && m.Description != "" {
		d := m.Description
		return &d
	}
	if m := setting.Meta(key); m != nil {
		d := m.Label
		return &d
	}
	return nil
}

// Cache exposes the in-memory option map for other services.
func (s *SystemSettingService) Cache() *setting.Cache {
	return s.cache
}

// GetSetting retrieves a setting by key (DB).
func (s *SystemSettingService) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	return s.settingRepo.GetSetting(ctx, key)
}

// GetSettingValue decodes option: prefers cache, falls back to DB then env default.
func (s *SystemSettingService) GetSettingValue(ctx context.Context, key string, dest interface{}) error {
	if s.cache != nil {
		if err := s.cache.Decode(key, dest); err == nil {
			return nil
		}
	}
	if err := s.settingRepo.GetSettingValue(ctx, key, dest); err == nil {
		return nil
	}
	// env/default map
	if raw, ok := setting.DefaultMap()[key]; ok {
		return json.Unmarshal([]byte(raw), dest)
	}
	return ErrSettingNotFound
}

// Catalog returns option schema for admin console (NewAPI-like rich options).
func (s *SystemSettingService) Catalog() []setting.OptionMeta {
	return setting.Catalog
}

// AdminSettingsResponse is console payload: schema + values.
type AdminSettingsResponse struct {
	Schema   []setting.OptionMeta   `json:"schema"`
	Settings []AdminSettingRow      `json:"settings"`
	ByKey    map[string]interface{} `json:"values"`
}

// AdminSettingRow is one row for the console table.
type AdminSettingRow struct {
	Key         string `json:"key"`
	Value       string `json:"value"` // JSON text; secrets redacted
	Category    string `json:"category,omitempty"`
	Type        string `json:"type,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret"`
	Configured  bool   `json:"configured,omitempty"` // for secrets: whether a real value is set
}

// GetAdminSettings builds schema + values for control panel.
func (s *SystemSettingService) GetAdminSettings(ctx context.Context) (*AdminSettingsResponse, error) {
	snap := s.cache.Snapshot()
	// refresh secrets knowledge from DB for "configured" flag
	dbMap, _ := s.settingRepo.LoadRawMap(ctx)

	rows := make([]AdminSettingRow, 0, len(setting.Catalog))
	values := make(map[string]interface{})
	for _, meta := range setting.Catalog {
		raw, ok := snap[meta.Key]
		if !ok {
			raw = setting.DefaultMap()[meta.Key]
		}
		row := AdminSettingRow{
			Key:         meta.Key,
			Value:       raw,
			Category:    meta.Category,
			Type:        meta.Type,
			Label:       meta.Label,
			Description: meta.Description,
			Secret:      meta.Secret,
		}
		if meta.Secret {
			// configured if non-empty string in DB/cache
			var str string
			_ = json.Unmarshal([]byte(raw), &str)
			if str == "" && dbMap != nil {
				if dbr, ok := dbMap[meta.Key]; ok {
					_ = json.Unmarshal([]byte(dbr), &str)
				}
			}
			row.Configured = str != ""
			row.Value = `"••••••••"`
			values[meta.Key] = map[string]interface{}{
				"configured": row.Configured,
				"redacted":   true,
			}
		} else {
			var decoded interface{}
			if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
				values[meta.Key] = decoded
			} else {
				values[meta.Key] = raw
			}
		}
		rows = append(rows, row)
	}
	return &AdminSettingsResponse{
		Schema:   setting.Catalog,
		Settings: rows,
		ByKey:    values,
	}, nil
}

// GetAllSettings flat list for backward-compatible admin UI.
func (s *SystemSettingService) GetAllSettings(ctx context.Context) ([]models.SystemSetting, error) {
	admin, err := s.GetAdminSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.SystemSetting, 0, len(admin.Settings))
	for _, row := range admin.Settings {
		desc := row.Description
		out = append(out, models.SystemSetting{
			Key:         row.Key,
			Value:       row.Value,
			Description: &desc,
		})
	}
	return out, nil
}

// UpdateSettingRequest represents a setting update request
type UpdateSettingRequest struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// UpdateSettings updates multiple settings (DB + memory cache).
func (s *SystemSettingService) UpdateSettings(ctx context.Context, settings []UpdateSettingRequest, updatedBy string) error {
	for _, item := range settings {
		if setting.IsSecret(item.Key) {
			if str, ok := item.Value.(string); ok && (str == "" || str == "••••••••") {
				continue
			}
		}
		valueJSON, err := json.Marshal(item.Value)
		if err != nil {
			return err
		}
		if err := s.settingRepo.SetRawJSON(ctx, item.Key, string(valueJSON), descFor(item.Key), &updatedBy); err != nil {
			return err
		}
		s.cache.Set(item.Key, string(valueJSON))
	}
	s.createAuditLog(ctx, updatedBy, "update", "system_settings", nil, nil)
	return nil
}

// GetPublicSettings returns settings safe for unauthenticated clients.
func (s *SystemSettingService) GetPublicSettings(ctx context.Context) (map[string]interface{}, error) {
	_ = ctx
	config := make(map[string]interface{})
	for _, meta := range setting.Catalog {
		if !meta.Public {
			continue
		}
		var value interface{}
		if err := s.GetSettingValue(ctx, meta.Key, &value); err == nil {
			config[meta.Key] = value
		}
	}
	return config, nil
}

// GetSiteConfig returns site branding keys.
func (s *SystemSettingService) GetSiteConfig(ctx context.Context) (map[string]interface{}, error) {
	return s.GetPublicSettings(ctx)
}

// IsInitialized checks if the system has been initialized
func (s *SystemSettingService) IsInitialized(ctx context.Context) bool {
	var value bool
	if err := s.GetSettingValue(ctx, setting.KeySystemInitialized, &value); err != nil {
		return false
	}
	return value
}

// InitializeSystem marks system as initialized.
func (s *SystemSettingService) InitializeSystem(ctx context.Context, _ string) error {
	return s.UpdateSettings(ctx, []UpdateSettingRequest{
		{Key: setting.KeySystemInitialized, Value: true},
	}, "system")
}

// AccessList methods

func (s *SystemSettingService) CreateAccessListEntry(ctx context.Context, entry *models.AccessList, createdBy string) error {
	entry.CreatedBy = &createdBy
	return s.accessListRepo.CreateEntry(ctx, entry)
}

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

func (s *SystemSettingService) DeleteAccessListEntry(ctx context.Context, id string) error {
	return s.accessListRepo.DeleteEntry(ctx, id)
}

func (s *SystemSettingService) IsListed(ctx context.Context, listType, target, value string) (bool, error) {
	return s.accessListRepo.IsListed(ctx, listType, target, value)
}

func (s *SystemSettingService) createAuditLog(ctx context.Context, userID, action, resourceType string, resourceID *string, oldValues map[string]interface{}) {
	changesJSON, _ := json.Marshal(oldValues)
	changes := make(map[string]interface{})
	_ = json.Unmarshal(changesJSON, &changes)
	logEntry := &models.AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      changes,
	}
	_ = s.auditRepo.CreateAuditLog(ctx, logEntry)
}
