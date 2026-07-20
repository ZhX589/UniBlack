package handler

import (
	"net/http"
	"strconv"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

// PublicAPIHandler handles public API requests
type PublicAPIHandler struct {
	subjectService  *service.SubjectService
	caseService     *service.CaseService
	evidenceService *service.EvidenceService
}

// NewPublicAPIHandler creates a new public API handler
func NewPublicAPIHandler(
	subjectService *service.SubjectService,
	caseService *service.CaseService,
	evidenceService *service.EvidenceService,
) *PublicAPIHandler {
	return &PublicAPIHandler{
		subjectService:  subjectService,
		caseService:     caseService,
		evidenceService: evidenceService,
	}
}

// SearchSubjects searches subjects (public)
func (h *PublicAPIHandler) SearchSubjects(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query is required"})
	}

	subjects, err := h.subjectService.SearchSubjects(c.Request().Context(), query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Format response for public API
	var results []map[string]interface{}
	for _, s := range subjects {
		results = append(results, map[string]interface{}{
			"id":           s.ID,
			"public_id":    s.PublicID,
			"display_name": s.DisplayName,
			"risk_level":   s.RiskLevel,
			"case_count":   s.CaseCount,
			"status":       s.Status,
			"identifiers":  s.Identifiers,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   len(results),
	})
}

// LookupSubject looks up a subject by identifier (public)
func (h *PublicAPIHandler) LookupSubject(c echo.Context) error {
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

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":           subject.ID,
		"public_id":    subject.PublicID,
		"display_name": subject.DisplayName,
		"risk_level":   subject.RiskLevel,
		"case_count":   subject.CaseCount,
		"status":       subject.Status,
		"identifiers":  subject.Identifiers,
	})
}

// GetSubject gets a subject by ID (public)
func (h *PublicAPIHandler) GetSubject(c echo.Context) error {
	id := c.Param("id")

	subject, err := h.subjectService.GetSubject(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrSubjectNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "subject not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":           subject.ID,
		"public_id":    subject.PublicID,
		"display_name": subject.DisplayName,
		"risk_level":   subject.RiskLevel,
		"case_count":   subject.CaseCount,
		"status":       subject.Status,
		"identifiers":  subject.Identifiers,
		"accounts":     subject.Accounts,
		"created_at":   subject.CreatedAt,
	})
}

// GetCase gets a case by ID (public)
func (h *PublicAPIHandler) GetCase(c echo.Context) error {
	id := c.Param("id")

	caseObj, err := h.caseService.GetCase(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrCaseNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "case not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Only show approved cases to public
	if caseObj.Status != "approved" && caseObj.Status != "closed" {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "case not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          caseObj.ID,
		"subject_id":  caseObj.SubjectID,
		"title":       caseObj.Title,
		"description": caseObj.Description,
		"status":      caseObj.Status,
		"severity":    caseObj.Severity,
		"verdict":     caseObj.Verdict,
		"created_at":  caseObj.CreatedAt,
	})
}

// GetCasesBySubject gets cases by subject ID (public)
func (h *PublicAPIHandler) GetCasesBySubject(c echo.Context) error {
	subjectID := c.Param("id")

	cases, err := h.caseService.GetCasesBySubjectID(c.Request().Context(), subjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Filter only approved/closed cases
	var publicCases []map[string]interface{}
	for _, caseObj := range cases {
		if caseObj.Status == "approved" || caseObj.Status == "closed" {
			publicCases = append(publicCases, map[string]interface{}{
				"id":          caseObj.ID,
				"title":       caseObj.Title,
				"description": caseObj.Description,
				"status":      caseObj.Status,
				"severity":    caseObj.Severity,
				"created_at":  caseObj.CreatedAt,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"cases": publicCases,
		"total": len(publicCases),
	})
}

// GetStatistics gets system statistics (public)
func (h *PublicAPIHandler) GetStatistics(c echo.Context) error {
	// This would typically query the database for counts
	// For now, return placeholder
	return c.JSON(http.StatusOK, map[string]interface{}{
		"subjects": 0,
		"cases":    0,
		"pending":  0,
	})
}

// ListSubjects lists subjects with pagination (public)
func (h *PublicAPIHandler) ListSubjects(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	status := c.QueryParam("status")

	subjects, total, err := h.subjectService.ListSubjects(c.Request().Context(), page, pageSize, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var results []map[string]interface{}
	for _, s := range subjects {
		results = append(results, map[string]interface{}{
			"id":           s.ID,
			"public_id":    s.PublicID,
			"display_name": s.DisplayName,
			"risk_level":   s.RiskLevel,
			"case_count":   s.CaseCount,
			"status":       s.Status,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subjects":  results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
