package handler

import (
	"net/http"
	"strconv"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

// EvidenceHandler handles evidence requests
type EvidenceHandler struct {
	evidenceService *service.EvidenceService
	eventService    *service.EventService
}

func (h *EvidenceHandler) SetEventService(eventService *service.EventService) {
	h.eventService = eventService
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
	if evidence.EventID != nil && h.eventService != nil {
		userID, _ := c.Get("user_id").(string)
		roles, _ := c.Get("roles").([]string)
		if _, err := h.eventService.CanReadEvent(c.Request().Context(), *evidence.EventID, userID, roles); err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "evidence not found"})
		}
	}

	return c.JSON(http.StatusOK, evidence)
}

// CreateEventTextEvidence attaches bounded text evidence to an Event archive.
func (h *EvidenceHandler) CreateEventTextEvidence(c echo.Context) error {
	if h.eventService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "event evidence unavailable"})
	}
	var req struct {
		Text           string `json:"text"`
		Title          string `json:"title"`
		EventNumber    int    `json:"event_number"`
		EvidenceNumber int    `json:"evidence_number"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	eventID := c.Param("id")
	userID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	if _, err := h.eventService.CanManageEvent(c.Request().Context(), eventID, userID, roles); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
	}
	publicID, err := h.eventService.SubjectPublicID(c.Request().Context(), eventID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
	}
	if req.EventNumber < 1 {
		req.EventNumber = 1
	}
	if req.EvidenceNumber < 1 {
		req.EvidenceNumber = 1
	}
	evidence, err := h.evidenceService.CreateEventTextEvidence(c.Request().Context(), eventID, publicID, req.EventNumber, req.EvidenceNumber, req.Text, req.Title, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, evidence)
}

// CreateEventLinkEvidence records link metadata without fetching the remote URL.
func (h *EvidenceHandler) CreateEventLinkEvidence(c echo.Context) error {
	if h.eventService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "event evidence unavailable"})
	}
	var req service.CreateEventLinkEvidenceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	eventID := c.Param("id")
	userID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	if _, err := h.eventService.CanManageEvent(c.Request().Context(), eventID, userID, roles); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
	}
	evidence, err := h.evidenceService.CreateEventLinkEvidence(c.Request().Context(), eventID, req, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, evidence)
}

// CreateEventFileEvidence attaches a binary file under the subject archive namespace.
func (h *EvidenceHandler) CreateEventFileEvidence(c echo.Context) error {
	if h.eventService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "event evidence unavailable"})
	}
	eventID := c.Param("id")
	userID, _ := c.Get("user_id").(string)
	roles, _ := c.Get("roles").([]string)
	if _, err := h.eventService.CanManageEvent(c.Request().Context(), eventID, userID, roles); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
	}
	publicID, err := h.eventService.SubjectPublicID(c.Request().Context(), eventID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid file"})
	}
	defer src.Close()
	eventNumber, _ := strconv.Atoi(c.FormValue("event_number"))
	evidenceNumber, _ := strconv.Atoi(c.FormValue("evidence_number"))
	if eventNumber < 1 {
		eventNumber = 1
	}
	if evidenceNumber < 1 {
		evidenceNumber = 1
	}
	evidence, err := h.evidenceService.CreateEventFileEvidence(c.Request().Context(), eventID, publicID, eventNumber, evidenceNumber, src, file.Filename, file.Size, c.FormValue("title"), userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, evidence)
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
