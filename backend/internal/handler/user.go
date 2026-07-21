package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

// UserManagementHandler handles user management requests
type UserManagementHandler struct {
	db *gorm.DB
}

// NewUserManagementHandler creates a new user management handler
func NewUserManagementHandler(db *gorm.DB) *UserManagementHandler {
	return &UserManagementHandler{db: db}
}

// ListUsers lists users with pagination
func (h *UserManagementHandler) ListUsers(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	search := c.QueryParam("search")
	role := c.QueryParam("role")
	status := c.QueryParam("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	userRepo := repository.NewUserRepository(h.db)
	users, total, err := userRepo.ListUsers(c.Request().Context(), offset, pageSize, search, role, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Remove sensitive fields
	type UserResponse struct {
		ID          string   `json:"id"`
		Username    string   `json:"username"`
		Email       string   `json:"email"`
		IsActive    bool     `json:"is_active"`
		Roles       []string `json:"roles"`
		CreatedAt   string   `json:"created_at"`
		LastLoginAt *string  `json:"last_login_at,omitempty"`
	}

	var userResponses []UserResponse
	for _, user := range users {
		roles := make([]string, len(user.Roles))
		for i, role := range user.Roles {
			roles[i] = role.Name
		}
		var lastLogin *string
		if user.LastLoginAt != nil {
			s := user.LastLoginAt.Format("2006-01-02T15:04:05Z")
			lastLogin = &s
		}
		userResponses = append(userResponses, UserResponse{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			IsActive:    user.IsActive,
			Roles:       roles,
			CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastLoginAt: lastLogin,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":     userResponses,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetUser retrieves a user by ID
func (h *UserManagementHandler) GetUser(c echo.Context) error {
	id := c.Param("id")

	userRepo := repository.NewUserRepository(h.db)
	user, err := userRepo.GetUserByID(c.Request().Context(), id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"is_active":      user.IsActive,
		"email_verified": user.EmailVerified,
		"roles":          roles,
		"created_at":     user.CreatedAt,
		"last_login_at":  user.LastLoginAt,
	})
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// UpdateUser updates a user
func (h *UserManagementHandler) UpdateUser(c echo.Context) error {
	id := c.Param("id")

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userRepo := repository.NewUserRepository(h.db)
	user, err := userRepo.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	if err := userRepo.UpdateUser(c.Request().Context(), user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "user updated"})
}

// ToggleUserActive toggles a user's active status
func (h *UserManagementHandler) ToggleUserActive(c echo.Context) error {
	id := c.Param("id")

	var req struct {
		Active bool `json:"active"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userRepo := repository.NewUserRepository(h.db)
	if err := userRepo.ToggleUserActive(c.Request().Context(), id, req.Active); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "user status updated"})
}

// UpdateUserRoleRequest represents a role update request
type UpdateUserRoleRequest struct {
	RoleName string `json:"role_name"`
}

// AssignRole assigns a role to a user
func (h *UserManagementHandler) AssignRole(c echo.Context) error {
	id := c.Param("id")

	var req UpdateUserRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userRepo := repository.NewUserRepository(h.db)

	// Get role by name
	role, err := userRepo.GetRoleByName(c.Request().Context(), req.RoleName)
	if err != nil || role == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role not found"})
	}

	if err := userRepo.AssignRole(c.Request().Context(), id, role.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "role assigned"})
}

// RemoveRole removes a role from a user
func (h *UserManagementHandler) RemoveRole(c echo.Context) error {
	userID := c.Param("id")
	roleName := c.Param("role")

	userRepo := repository.NewUserRepository(h.db)

	// Get role by name
	role, err := userRepo.GetRoleByName(c.Request().Context(), roleName)
	if err != nil || role == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role not found"})
	}

	if err := userRepo.RemoveRole(c.Request().Context(), userID, role.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "role removed"})
}
