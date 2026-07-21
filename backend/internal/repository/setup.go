package repository

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/setting"
)

// SetupRepository performs first-run setup as one transaction.
type SetupRepository struct{ db *gorm.DB }

func NewSetupRepository(db *gorm.DB) *SetupRepository { return &SetupRepository{db: db} }

// Initialize claims setup under a row lock before creating the administrator.
// It returns false when another request has already completed setup.
func (r *SetupRepository) Initialize(ctx context.Context, admin *models.User, siteName string) (bool, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var initialized models.SystemSetting
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key = ?", setting.KeySystemInitialized).First(&initialized).Error; err != nil {
			return err
		}
		if initialized.Value == "true" {
			return errSetupAlreadyInitialized
		}
		if err := tx.Create(admin).Error; err != nil {
			return err
		}
		var role models.Role
		if err := tx.Where("name = ?", "admin").First(&role).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.UserRole{UserID: admin.ID, RoleID: role.ID}).Error; err != nil {
			return err
		}
		if siteName != "" {
			if err := tx.Model(&models.SystemSetting{}).Where("key = ?", setting.KeySiteName).Update("value", setupSiteNameJSON(siteName)).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.SystemSetting{}).Where("key = ?", setting.KeySystemInitialized).Update("value", "true").Error
	})
	if errors.Is(err, errSetupAlreadyInitialized) {
		return false, nil
	}
	return err == nil, err
}

var errSetupAlreadyInitialized = errors.New("system already initialized")

func setupSiteNameJSON(siteName string) string {
	value, _ := json.Marshal(siteName)
	return string(value)
}
