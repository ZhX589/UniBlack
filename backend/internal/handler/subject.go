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

type subjectService interface {
	CreateSubject(context.Context, service.CreateSubjectRequest, string) (*models.Subject, error)
	GetSubject(context.Context, string) (*models.Subject, error)
	GetSubjectByIdentifier(context.Context, string, string) (*models.Subject, error)
	ListSubjects(context.Context, int, int, string) ([]models.Subject, int64, error)
	UpdateSubject(context.Context, string, service.UpdateSubjectRequest, string, []string) (*models.Subject, error)
	DeleteSubject(context.Context, string, string, []string) error
	SearchSubjects(context.Context, string) ([]models.Subject, error)
	AddIdentifier(context.Context, string, service.IdentifierRequest, string, []string) (*models.Identifier, error)
	RemoveIdentifier(context.Context, string, string, []string) error
	GetIdentifiersBySubjectID(context.Context, string) ([]models.Identifier, error)
}

// SubjectHandler handles subject requests
type SubjectHandler struct {
	subjectService subjectService
}

// NewSubjectHandler creates a new subject handler
func NewSubjectHandler(subjectService *service.SubjectService) *SubjectHandler {
	return &SubjectHandler{subjectService: subjectService}
}

// CreateSubject creates a new subject
func (h *SubjectHandler) CreateSubject(c echo.Context) error {
	var req service.CreateSubjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	subject, err := h.subjectService.CreateSubject(c.Request().Context(), req, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, subject)
}

// GetSubject retrieves a subject by ID
func (h *SubjectHandler) GetSubject(c echo.Context) error {
	id := c.Param("id")

	subject, err := h.subjectService.GetSubject(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrSubjectNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subject not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, subjectWithVisibleEvents(c, subject))
}

// GetSubjectByIdentifier retrieves a subject by identifier
func (h *SubjectHandler) GetSubjectByIdentifier(c echo.Context) error {
	platform := c.QueryParam("platform")
	value := c.QueryParam("value")

	if platform == "" || value == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "platform and value are required"})
	}

	subject, err := h.subjectService.GetSubjectByIdentifier(c.Request().Context(), platform, value)
	if err != nil {
		if err == service.ErrSubjectNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subject not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, subjectWithVisibleEvents(c, subject))
}

// ListSubjects lists subjects with pagination
func (h *SubjectHandler) ListSubjects(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = domain.NormalizePagination(page, pageSize)
	status := c.QueryParam("status")

	subjects, total, err := h.subjectService.ListSubjects(c.Request().Context(), page, pageSize, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subjects":  subjectsWithVisibleEvents(c, subjects),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateSubject updates a subject
func (h *SubjectHandler) UpdateSubject(c echo.Context) error {
	id := c.Param("id")

	var req service.UpdateSubjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	actorID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	subject, err := h.subjectService.UpdateSubject(c.Request().Context(), id, req, actorID, roles)
	if err != nil {
		if errors.Is(err, service.ErrSubjectAccessDenied) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "subject access denied"})
		}
		if err == service.ErrSubjectNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subject not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, subjectWithVisibleEvents(c, subject))
}

// DeleteSubject deletes a subject
func (h *SubjectHandler) DeleteSubject(c echo.Context) error {
	id := c.Param("id")

	actorID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	if err := h.subjectService.DeleteSubject(c.Request().Context(), id, actorID, roles); err != nil {
		if errors.Is(err, service.ErrSubjectAccessDenied) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "subject access denied"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "subject deleted"})
}

// SearchSubjects searches subjects
func (h *SubjectHandler) SearchSubjects(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query is required"})
	}

	subjects, err := h.subjectService.SearchSubjects(c.Request().Context(), query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subjects": subjectsWithVisibleEvents(c, subjects),
		"total":    len(subjects),
	})
}

// AddIdentifier adds an identifier to a subject
func (h *SubjectHandler) AddIdentifier(c echo.Context) error {
	subjectID := c.Param("id")

	var req service.IdentifierRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	actorID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	identifier, err := h.subjectService.AddIdentifier(c.Request().Context(), subjectID, req, actorID, roles)
	if err != nil {
		if errors.Is(err, service.ErrSubjectAccessDenied) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "subject access denied"})
		}
		if err == service.ErrSubjectNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subject not found"})
		}
		if err == service.ErrIdentifierExists {
			return c.JSON(http.StatusConflict, map[string]string{"error": "identifier already exists"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, identifier)
}

// RemoveIdentifier removes an identifier
func (h *SubjectHandler) RemoveIdentifier(c echo.Context) error {
	id := c.Param("id")

	actorID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	if err := h.subjectService.RemoveIdentifier(c.Request().Context(), id, actorID, roles); err != nil {
		if errors.Is(err, service.ErrSubjectAccessDenied) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "subject access denied"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "identifier removed"})
}

// GetIdentifiersBySubjectID retrieves all identifiers for a subject
func (h *SubjectHandler) GetIdentifiersBySubjectID(c echo.Context) error {
	subjectID := c.Param("id")

	identifiers, err := h.subjectService.GetIdentifiersBySubjectID(c.Request().Context(), subjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, identifiers)
}
