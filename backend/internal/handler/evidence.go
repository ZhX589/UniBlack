package handler

import (
	"net/http"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

// EvidenceHandler handles evidence requests
type EvidenceHandler struct {
	evidenceService *service.EvidenceService
}

// NewEvidenceHandler creates a new evidence handler
func NewEvidenceHandler(evidenceService *service.EvidenceService) *EvidenceHandler {
	return &EvidenceHandler{evidenceService: evidenceService}
}

// CreateEvidence creates a new evidence entry (for links/text)
func (h *EvidenceHandler) CreateEvidence(c echo.Context) error {
	var req service.CreateEvidenceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	userID, _ := c.Get("user_id").(string)

	evidence, err := h.evidenceService.CreateEvidence(c.Request().Context(), req, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, evidence)
}

// UploadEvidence uploads a file as evidence
func (h *EvidenceHandler) UploadEvidence(c echo.Context) error {
	caseID := c.FormValue("case_id")
	title := c.FormValue("title")
	description := c.FormValue("description")

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open file"})
	}
	defer src.Close()

	userID, _ := c.Get("user_id").(string)

	req := service.UploadEvidenceRequest{
		CaseID:      caseID,
		Title:       title,
		Description: description,
	}

	evidence, err := h.evidenceService.UploadEvidence(c.Request().Context(), req, src, file.Filename, file.Size, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, evidence)
}

// GetEvidence retrieves evidence by ID
func (h *EvidenceHandler) GetEvidence(c echo.Context) error {
	id := c.Param("id")

	evidence, err := h.evidenceService.GetEvidence(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrEvidenceNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "evidence not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, evidence)
}

// GetEvidenceByCaseID retrieves all evidence for a case
func (h *EvidenceHandler) GetEvidenceByCaseID(c echo.Context) error {
	caseID := c.Param("id")

	evidences, err := h.evidenceService.GetEvidenceByCaseID(c.Request().Context(), caseID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, evidences)
}

// DeleteEvidence deletes evidence
func (h *EvidenceHandler) DeleteEvidence(c echo.Context) error {
	id := c.Param("id")

	if err := h.evidenceService.DeleteEvidence(c.Request().Context(), id); err != nil {
		if err == service.ErrEvidenceNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "evidence not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "evidence deleted"})
}
