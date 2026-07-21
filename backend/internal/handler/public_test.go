package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appMiddleware "github.com/ZhX589/UniBlack/backend/internal/middleware"
	"github.com/labstack/echo/v4"
)

func TestPublicCaseAliasRouteReturnsDeprecationHeaders(t *testing.T) {
	e := echo.New()
	api := e.Group("/api/v1")
	api.GET("/cases/:id", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"description": "legacy"})
	}, appMiddleware.CaseDeprecation("/api/v1/events/:id"))
	api.GET("/events/:id", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"details": "canonical"})
	})

	caseResponse := httptest.NewRecorder()
	e.ServeHTTP(caseResponse, httptest.NewRequest(http.MethodGet, "/api/v1/cases/legacy-id", nil))
	if got := caseResponse.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("case Deprecation = %q, want true", got)
	}
	if got := caseResponse.Header().Get("Sunset"); got != appMiddleware.CaseSunset {
		t.Fatalf("case Sunset = %q, want %q", got, appMiddleware.CaseSunset)
	}
	if got := caseResponse.Header().Get("Link"); got != "</api/v1/events/legacy-id>; rel=\"successor-version\"" {
		t.Fatalf("case Link = %q", got)
	}

	eventResponse := httptest.NewRecorder()
	e.ServeHTTP(eventResponse, httptest.NewRequest(http.MethodGet, "/api/v1/events/event-id", nil))
	if got := eventResponse.Header().Get("Deprecation"); got != "" {
		t.Fatalf("event Deprecation = %q, want empty", got)
	}
}
