package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/config"
	appMiddleware "github.com/ZhX589/UniBlack/backend/internal/middleware"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
	"github.com/labstack/echo/v4"
)

func TestRegisterPublicEventRoutesAppliesDeprecationOnlyToLegacyCaseRoute(t *testing.T) {
	e := echo.New()
	api := e.Group("/api/v1")
	noContent := func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}
	registerPublicEventRoutes(api, noContent, noContent)

	for _, test := range []struct {
		path            string
		wantDeprecation string
		wantLink        string
	}{
		{path: "/api/v1/events/event-id", wantDeprecation: "", wantLink: ""},
		{path: "/api/v1/cases/legacy-id", wantDeprecation: "true", wantLink: "</api/v1/events/legacy-id>; rel=\"successor-version\""},
	} {
		t.Run(test.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
			}
			if got := res.Header().Get("Deprecation"); got != test.wantDeprecation {
				t.Fatalf("Deprecation = %q, want %q", got, test.wantDeprecation)
			}
			if test.wantDeprecation != "" && res.Header().Get("Sunset") != appMiddleware.CaseSunset {
				t.Fatalf("Sunset = %q, want %q", res.Header().Get("Sunset"), appMiddleware.CaseSunset)
			}
			if got := res.Header().Get("Link"); got != test.wantLink {
				t.Fatalf("Link = %q, want %q", got, test.wantLink)
			}
		})
	}
}

func TestRegisterAuthRoutesUsesAuthRateLimit(t *testing.T) {
	settings := fakeSettingReader{values: map[string]int{
		"security.rate_limit_public": 1,
		"security.rate_limit_auth":   2,
	}}
	e := echo.New()
	auth := e.Group("/api/auth")
	registerAuthRoutes(auth, appMiddleware.NewRequestRateLimiter(settings, "security.rate_limit_auth").Middleware, noContentHandler(), noContentHandler(), noContentHandler(), noContentHandler(), noContentHandler())

	for i := 0; i < 2; i++ {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "192.0.2.99:1234"
		e.ServeHTTP(res, req)
		if res.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d", i+1, res.Code, http.StatusNoContent)
		}
	}
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "192.0.2.99:1234"
	e.ServeHTTP(res, req)
	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("third auth request status = %d, want %d", res.Code, http.StatusTooManyRequests)
	}
}

func noContentHandler() echo.HandlerFunc {
	return func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }
}

type fakeSettingReader struct{ values map[string]int }

func (f fakeSettingReader) GetSettingValue(_ context.Context, key string, dest interface{}) error {
	value, ok := f.values[key]
	if !ok {
		return errors.New("setting not found")
	}
	*dest.(*int) = value
	return nil
}

func TestRegisterLegacyCaseRoutesAvoidsTemplateSuccessorLinks(t *testing.T) {
	e := echo.New()
	noContent := func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}
	registerLegacyCaseRoutes(e.Group("/api"), noContent, noContent, noContent, noContent, noContent, noContent, noContent, noContent)

	for _, test := range []struct {
		path     string
		wantLink string
	}{
		{path: "/api/cases", wantLink: ""},
		{path: "/api/cases/legacy-id", wantLink: "</api/events/legacy-id>; rel=\"successor-version\""},
		{path: "/api/cases/legacy-id/evidence", wantLink: ""},
		{path: "/api/cases/legacy-id/appeals", wantLink: ""},
	} {
		t.Run(test.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if got := res.Header().Get("Deprecation"); got != "true" {
				t.Fatalf("Deprecation = %q, want true", got)
			}
			if got := res.Header().Get("Link"); got != test.wantLink {
				t.Fatalf("Link = %q, want %q", got, test.wantLink)
			}
		})
	}
}

func TestSelectStorage(t *testing.T) {
	initErr := errors.New("S3 unavailable")
	for _, test := range []struct {
		name    string
		cfg     *config.Config
		s3      func(string, string, string, string, bool, string) (storage.Storage, error)
		wantErr error
	}{
		{
			name: "configured S3",
			cfg:  &config.Config{MinioEndpoint: "minio.test:9000", MinioUseSSL: true, MinioPublicBase: "https://evidence.test"},
			s3: func(endpoint, _ string, _ string, _ string, useSSL bool, publicBase string) (storage.Storage, error) {
				if endpoint != "minio.test:9000" {
					t.Fatalf("endpoint = %q", endpoint)
				}
				if !useSSL || publicBase != "https://evidence.test" {
					t.Fatalf("S3 options = (%t, %q)", useSSL, publicBase)
				}
				return &storage.LocalStorage{}, nil
			},
		},
		{
			name:    "configured S3 initialization error",
			cfg:     &config.Config{MinioEndpoint: "minio.test:9000"},
			s3:      func(string, string, string, string, bool, string) (storage.Storage, error) { return nil, initErr },
			wantErr: initErr,
		},
		{name: "development local fallback", cfg: &config.Config{Environment: "development"}},
		{name: "unset environment endpoint required", cfg: &config.Config{}, wantErr: errS3EndpointRequired},
		{name: "production endpoint required", cfg: &config.Config{Environment: "production"}, wantErr: errS3EndpointRequired},
	} {
		t.Run(test.name, func(t *testing.T) {
			constructors := storageConstructors{
				s3:    test.s3,
				local: func(string, string) storage.Storage { return &storage.LocalStorage{} },
			}
			got, err := selectStorage(test.cfg, constructors)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && got == nil {
				t.Fatal("storage = nil")
			}
		})
	}
}
