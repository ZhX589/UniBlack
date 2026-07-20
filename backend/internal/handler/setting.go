package handler

import (
	"net/http"
	"strconv"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

// SystemSettingHandler handles system setting requests
type SystemSettingHandler struct {
	settingService *service.SystemSettingService
}

// NewSystemSettingHandler creates a new system setting handler
func NewSystemSettingHandler(settingService *service.SystemSettingService) *SystemSettingHandler {
	return &SystemSettingHandler{settingService: settingService}
}

// GetPublicSettings returns public settings
func (h *SystemSettingHandler) GetPublicSettings(c echo.Context) error {
	settings, err := h.settingService.GetPublicSettings(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, settings)
}

// GetAllSettings returns all settings as a flat array (admin only)
func (h *SystemSettingHandler) GetAllSettings(c echo.Context) error {
	settings, err := h.settingService.GetAllSettings(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"settings": settings,
	})
}

// UpdateSettings updates multiple settings (admin only)
func (h *SystemSettingHandler) UpdateSettings(c echo.Context) error {
	var req []service.UpdateSettingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	if err := h.settingService.UpdateSettings(c.Request().Context(), req, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "settings updated"})
}

// IsInitialized checks if system is initialized
func (h *SystemSettingHandler) IsInitialized(c echo.Context) error {
	initialized := h.settingService.IsInitialized(c.Request().Context())
	return c.JSON(http.StatusOK, map[string]bool{"initialized": initialized})
}

// AccessList handlers

// ListAccessListEntries lists access list entries
func (h *SystemSettingHandler) ListAccessListEntries(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	listType := c.QueryParam("type")
	target := c.QueryParam("target")

	entries, total, err := h.settingService.ListAccessListEntries(c.Request().Context(), page, pageSize, listType, target)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries":   entries,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateAccessListEntry creates a new access list entry
func (h *SystemSettingHandler) CreateAccessListEntry(c echo.Context) error {
	var req models.AccessList
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	if err := h.settingService.CreateAccessListEntry(c.Request().Context(), &req, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, req)
}

// DeleteAccessListEntry deletes an access list entry
func (h *SystemSettingHandler) DeleteAccessListEntry(c echo.Context) error {
	id := c.Param("id")

	if err := h.settingService.DeleteAccessListEntry(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "entry deleted"})
}
