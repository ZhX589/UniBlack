package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/service"
)

// SubmissionHandler handles submission requests
type SubmissionHandler struct {
	submissionService *service.SubmissionService
}

// NewSubmissionHandler creates a new submission handler
func NewSubmissionHandler(submissionService *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{submissionService: submissionService}
}

// CreateSubmission creates a new submission
func (h *SubmissionHandler) CreateSubmission(c echo.Context) error {
	var req service.CreateSubmissionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	submission, err := h.submissionService.CreateSubmission(c.Request().Context(), req, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, submission)
}

// GetSubmission retrieves a submission by ID
func (h *SubmissionHandler) GetSubmission(c echo.Context) error {
	id := c.Param("id")

	submission, err := h.submissionService.GetSubmission(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrSubmissionNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "submission not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, submission)
}

// ListSubmissions lists submissions with pagination
func (h *SubmissionHandler) ListSubmissions(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	status := c.QueryParam("status")
	submittedBy := c.QueryParam("submitted_by")

	submissions, total, err := h.submissionService.ListSubmissions(c.Request().Context(), page, pageSize, status, submittedBy)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"submissions": submissions,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// ReviewSubmission reviews a submission
func (h *SubmissionHandler) ReviewSubmission(c echo.Context) error {
	id := c.Param("id")

	var req service.ReviewSubmissionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	submission, err := h.submissionService.ReviewSubmission(c.Request().Context(), id, req, userID)
	if err != nil {
		if err == service.ErrSubmissionNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "submission not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, submission)
}

// DeleteSubmission deletes a submission
func (h *SubmissionHandler) DeleteSubmission(c echo.Context) error {
	id := c.Param("id")

	userID, _ := c.Get("user_id").(string)

	if err := h.submissionService.DeleteSubmission(c.Request().Context(), id, userID); err != nil {
		if err == service.ErrSubmissionNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "submission not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "submission deleted"})
}
