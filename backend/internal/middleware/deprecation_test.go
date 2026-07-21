package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCaseDeprecationHeaders(t *testing.T) {
	e := echo.New()
	e.GET("/api/v1/cases/:id", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}, CaseDeprecation("/api/v1/events/:id"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/legacy-id", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	if got := res.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("Deprecation = %q, want true", got)
	}
	if got := res.Header().Get("Sunset"); got != CaseSunset {
		t.Fatalf("Sunset = %q, want %q", got, CaseSunset)
	}
	if got := res.Header().Get("Link"); got != "</api/v1/events/legacy-id>; rel=\"successor-version\"" {
		t.Fatalf("Link = %q", got)
	}
	if got := res.Header().Get("Warning"); got == "" {
		t.Fatal("missing Warning header")
	}
}

func TestCaseDeprecationHeadersWithoutSuccessor(t *testing.T) {
	e := echo.New()
	e.GET("/api/cases", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}, CaseDeprecation(""))

	req := httptest.NewRequest(http.MethodGet, "/api/cases", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	if got := res.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("Deprecation = %q, want true", got)
	}
	if got := res.Header().Get("Link"); got != "" {
		t.Fatalf("Link = %q, want empty", got)
	}
}

func TestEventRouteHasNoCaseDeprecationHeaders(t *testing.T) {
	e := echo.New()
	e.GET("/api/v1/events/:id", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/event-id", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	if got := res.Header().Get("Deprecation"); got != "" {
		t.Fatalf("Deprecation = %q, want empty", got)
	}
}
