package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidPassword   = errors.New("invalid password")
)

// UserRepository handles user database operations
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user
func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateLastLogin updates the last login time
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", now).Error
}

// AssignRole assigns a role to a user
func (r *UserRepository) AssignRole(ctx context.Context, userID, roleID string) error {
	return r.db.WithContext(ctx).
		Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING", userID, roleID).Error
}

// GetUserPermissions retrieves all permissions for a user
func (r *UserRepository) GetUserPermissions(ctx context.Context, userID string) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.WithContext(ctx).
		Distinct().
		Select("permissions.*").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Joins("JOIN user_roles ON role_permissions.role_id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&permissions).Error
	return permissions, err
}

// ListUsers lists users with pagination and filters
func (r *UserRepository) ListUsers(ctx context.Context, offset, limit int, search, role, status string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.WithContext(ctx).Model(&models.User{})
	if search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		query = query.Joins("JOIN user_roles ON users.id = user_roles.user_id").
			Joins("JOIN roles ON user_roles.role_id = roles.id").
			Where("roles.name = ?", role)
	}
	if status == "active" {
		query = query.Where("is_active = ?", true)
	} else if status == "inactive" {
		query = query.Where("is_active = ?", false)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Roles").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&users).Error

	return users, total, err
}

// ToggleUserActive toggles a user's active status
func (r *UserRepository) ToggleUserActive(ctx context.Context, userID string, active bool) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("is_active", active).Error
}

// RemoveRole removes a role from a user
func (r *UserRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	return r.db.WithContext(ctx).
		Exec("DELETE FROM user_roles WHERE user_id = ? AND role_id = ?", userID, roleID).Error
}

// GetUserRoles retrieves all roles for a user
func (r *UserRepository) GetUserRoles(ctx context.Context, userID string) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

// GetRoleByName retrieves a role by name
func (r *UserRepository) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}
