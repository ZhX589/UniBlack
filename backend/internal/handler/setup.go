package handler

import (
	"net/http"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

// SetupHandler handles system setup requests
type SetupHandler struct {
	authService    *service.AuthService
	settingService *service.SystemSettingService
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(authService *service.AuthService, settingService *service.SystemSettingService) *SetupHandler {
	return &SetupHandler{
		authService:    authService,
		settingService: settingService,
	}
}

// CheckSetup checks if the system has been initialized
func (h *SetupHandler) CheckSetup(c echo.Context) error {
	initialized := h.settingService.IsInitialized(c.Request().Context())
	return c.JSON(http.StatusOK, map[string]bool{"initialized": initialized})
}

// InitializeRequest represents an initialization request
type InitializeRequest struct {
	AdminPassword string `json:"admin_password" validate:"required,min=8"`
	SiteName      string `json:"site_name"`
}

// Initialize initializes the system
func (h *SetupHandler) Initialize(c echo.Context) error {
	// Check if already initialized
	if h.settingService.IsInitialized(c.Request().Context()) {
		return c.JSON(http.StatusConflict, map[string]string{"error": "system already initialized"})
	}

	var req InitializeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Create admin user
	if err := h.authService.SeedAdmin(c.Request().Context(), req.AdminPassword); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create admin"})
	}

	// Update site name if provided
	if req.SiteName != "" {
		siteName := req.SiteName
		h.settingService.UpdateSettings(c.Request().Context(), []service.UpdateSettingRequest{
			{Key: "site.name", Value: siteName},
		}, "system")
	}

	// Mark system as initialized
	if err := h.settingService.InitializeSystem(c.Request().Context(), req.AdminPassword); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to initialize"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "system initialized successfully"})
}
