package handler

import (
	"errors"
	"net/http"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

// SetupHandler handles system setup requests
type SetupHandler struct {
	setupService   *service.SetupService
	settingService *service.SystemSettingService
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(setupService *service.SetupService, settingService *service.SystemSettingService) *SetupHandler {
	return &SetupHandler{
		setupService:   setupService,
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
	var req InitializeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.setupService.Initialize(c.Request().Context(), req.AdminPassword, req.SiteName); err != nil {
		if errors.Is(err, service.ErrAlreadyInitialized) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "system already initialized"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to initialize"})
	}
	h.settingService.ApplySetupCache(req.SiteName)

	return c.JSON(http.StatusOK, map[string]string{"message": "system initialized successfully"})
}
