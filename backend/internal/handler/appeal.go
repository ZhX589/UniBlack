package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/domain"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/service"
)

type appealRetirementService interface {
	DeleteAppeal(context.Context, string, string, ...string) error
}

type appealReviewService interface {
	ReviewAppeal(context.Context, string, service.ReviewAppealRequest, string) (*models.Appeal, error)
}

// AppealHandler handles appeal requests
type AppealHandler struct {
	appealService     *service.AppealService
	retirementService appealRetirementService
	reviewService     appealReviewService
}

// NewAppealHandler creates a new appeal handler
func NewAppealHandler(appealService *service.AppealService) *AppealHandler {
	return &AppealHandler{appealService: appealService, retirementService: appealService, reviewService: appealService}
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
	page, pageSize = domain.NormalizePagination(page, pageSize)
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

	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if req.Status != "approved" && req.Status != "rejected" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "review status must be approved or rejected"})
	}
	appeal, err := h.reviewService.ReviewAppeal(c.Request().Context(), id, req, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAppealNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "appeal not found"})
		case errors.Is(err, service.ErrAppealAlreadyResolved):
			return c.JSON(http.StatusConflict, map[string]string{"error": "appeal already resolved"})
		case errors.Is(err, service.ErrAppealResolveRequired):
			return c.JSON(http.StatusConflict, map[string]string{"error": "event appeals require the formal resolve endpoint"})
		case errors.Is(err, service.ErrAppealInvalidReview):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "review status must be approved or rejected"})
		case errors.Is(err, service.ErrAppealInvalidTarget):
			return c.JSON(http.StatusConflict, map[string]string{"error": "legacy review requires an existing case target"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to review appeal"})
		}
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

// DeleteAppeal retires an appeal without erasing its history.
func (h *AppealHandler) DeleteAppeal(c echo.Context) error {
	id := c.Param("id")

	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	roles, _ := c.Get("roles").([]string)

	if err := h.retirementService.DeleteAppeal(c.Request().Context(), id, userID, roles...); err != nil {
		switch {
		case errors.Is(err, service.ErrAppealNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "appeal not found"})
		case errors.Is(err, service.ErrAppealForbidden):
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not permitted to retire this appeal"})
		case errors.Is(err, service.ErrAppealAlreadyResolved):
			return c.JSON(http.StatusConflict, map[string]string{"error": "appeal already resolved"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retire appeal"})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "appeal deleted"})
}
