package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/service"
)

// CaseHandler handles case requests
type CaseHandler struct {
	caseService *service.CaseService
}

// NewCaseHandler creates a new case handler
func NewCaseHandler(caseService *service.CaseService) *CaseHandler {
	return &CaseHandler{caseService: caseService}
}

// CreateCase creates a new case
func (h *CaseHandler) CreateCase(c echo.Context) error {
	var req service.CreateCaseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	caseObj, err := h.caseService.CreateCase(c.Request().Context(), req, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, caseObj)
}

// GetCase retrieves a case by ID
func (h *CaseHandler) GetCase(c echo.Context) error {
	id := c.Param("id")

	caseObj, err := h.caseService.GetCase(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrCaseNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "case not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, caseObj)
}

// ListCases lists cases with pagination
func (h *CaseHandler) ListCases(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	status := c.QueryParam("status")
	subjectID := c.QueryParam("subject_id")

	cases, total, err := h.caseService.ListCases(c.Request().Context(), page, pageSize, status, subjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"cases":     cases,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateCase updates a case
func (h *CaseHandler) UpdateCase(c echo.Context) error {
	id := c.Param("id")

	var req service.UpdateCaseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	caseObj, err := h.caseService.UpdateCase(c.Request().Context(), id, req, userID)
	if err != nil {
		if err == service.ErrCaseNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "case not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, caseObj)
}

// DeleteCase deletes a case
func (h *CaseHandler) DeleteCase(c echo.Context) error {
	id := c.Param("id")

	userID, _ := c.Get("user_id").(string)

	if err := h.caseService.DeleteCase(c.Request().Context(), id, userID); err != nil {
		if err == service.ErrCaseNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "case not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "case deleted"})
}

// ReviewCase reviews a case
func (h *CaseHandler) ReviewCase(c echo.Context) error {
	id := c.Param("id")

	var req service.ReviewCaseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	caseObj, err := h.caseService.ReviewCase(c.Request().Context(), id, req, userID)
	if err != nil {
		if err == service.ErrCaseNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "case not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, caseObj)
}

// GetCaseHistory returns audit logs for a case
func (h *CaseHandler) GetCaseHistory(c echo.Context) error {
	id := c.Param("id")

	logs, err := h.caseService.GetCaseHistory(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, logs)
}

// GetCasesBySubjectID returns all cases for a subject
func (h *CaseHandler) GetCasesBySubjectID(c echo.Context) error {
	subjectID := c.Param("id")

	cases, err := h.caseService.GetCasesBySubjectID(c.Request().Context(), subjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, cases)
}
