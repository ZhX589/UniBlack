package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/service"
)

// AppealHandler handles appeal requests
type AppealHandler struct {
	appealService *service.AppealService
}

// NewAppealHandler creates a new appeal handler
func NewAppealHandler(appealService *service.AppealService) *AppealHandler {
	return &AppealHandler{appealService: appealService}
}

// CreateAppeal creates a new appeal
func (h *AppealHandler) CreateAppeal(c echo.Context) error {
	var req service.CreateAppealRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if eventID := c.Param("id"); eventID != "" {
		req.EventID = eventID
	}

	userID, _ := c.Get("user_id").(string)

	appeal, err := h.appealService.CreateAppeal(c.Request().Context(), req, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, appeal)
}

// GetAppeal retrieves an appeal by ID
func (h *AppealHandler) GetAppeal(c echo.Context) error {
	id := c.Param("id")

	appeal, err := h.appealService.GetAppeal(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrAppealNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "appeal not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, appeal)
}

// GetAppealsByCaseID retrieves all appeals for a case
func (h *AppealHandler) GetAppealsByCaseID(c echo.Context) error {
	caseID := c.Param("id")

	appeals, err := h.appealService.GetAppealsByCaseID(c.Request().Context(), caseID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, appeals)
}

// GetAppealsByEventID returns the canonical Event-first appeal history.
func (h *AppealHandler) GetAppealsByEventID(c echo.Context) error {
	appeals, err := h.appealService.GetAppealsByEventID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, appeals)
}

// ListAppeals lists appeals with pagination
func (h *AppealHandler) ListAppeals(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	status := c.QueryParam("status")
	submittedBy := c.QueryParam("submitted_by")

	appeals, total, err := h.appealService.ListAppeals(c.Request().Context(), page, pageSize, status, submittedBy)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"appeals":   appeals,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ReviewAppeal reviews an appeal
func (h *AppealHandler) ReviewAppeal(c echo.Context) error {
	id := c.Param("id")

	var req service.ReviewAppealRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	appeal, err := h.appealService.ReviewAppeal(c.Request().Context(), id, req, userID)
	if err != nil {
		if err == service.ErrAppealNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "appeal not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, appeal)
}

func (h *AppealHandler) ResolveAppeal(c echo.Context) error {
	var req service.ResolveAppealRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	userID, _ := c.Get("user_id").(string)
	appeal, err := h.appealService.ResolveAppeal(c.Request().Context(), c.Param("id"), req, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, appeal)
}

// DeleteAppeal deletes an appeal
func (h *AppealHandler) DeleteAppeal(c echo.Context) error {
	id := c.Param("id")

	userID, _ := c.Get("user_id").(string)

	if err := h.appealService.DeleteAppeal(c.Request().Context(), id, userID); err != nil {
		if err == service.ErrAppealNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "appeal not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "appeal deleted"})
}
