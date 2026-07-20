package repository

import (
	"context"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"gorm.io/gorm"
)

// AuditLogRepository handles audit log database operations
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// CreateAuditLog creates a new audit log entry
func (r *AuditLogRepository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetAuditLogs retrieves audit logs with pagination
func (r *AuditLogRepository) GetAuditLogs(ctx context.Context, offset, limit int, userID, resourceType, resourceID string) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

// GetAuditLogsByResource retrieves audit logs for a specific resource
func (r *AuditLogRepository) GetAuditLogsByResource(ctx context.Context, resourceType, resourceID string) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}
