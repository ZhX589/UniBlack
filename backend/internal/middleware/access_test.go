package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestAccessListRejectsBlacklistedIPAndAuthenticatedIdentity(t *testing.T) {
	checker := fakeAccessChecker{listed: map[string]bool{
		"blacklist/ip/192.0.2.10":         true,
		"blacklist/email/banned@test.dev": true,
	}}
	e := echo.New()
	e.GET("/ip", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }, AccessList(checker))
	e.GET("/identity", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }, AccessList(checker))

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("blacklisted IP status = %d, want %d", res.Code, http.StatusForbidden)
	}

	req = httptest.NewRequest(http.MethodGet, "/identity", nil)
	req.RemoteAddr = "192.0.2.11:1234"
	res = httptest.NewRecorder()
	ctx := e.NewContext(req, res)
	ctx.Set("email", "banned@test.dev")
	if err := AccessList(checker)(func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })(ctx); err != nil {
		t.Fatalf("middleware error = %v", err)
	}
	if res.Code != http.StatusForbidden {
		t.Fatalf("blacklisted email status = %d, want %d", res.Code, http.StatusForbidden)
	}
}

func TestWhitelistBypassesRateLimit(t *testing.T) {
	checker := fakeAccessChecker{listed: map[string]bool{"whitelist/ip/192.0.2.10": true}}
	e := echo.New()
	limited := RateLimit(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error { return c.NoContent(http.StatusTooManyRequests) }
	})
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }, AccessList(checker), limited)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("whitelisted status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestRequestRateLimiterUsesConfiguredLimit(t *testing.T) {
	settings := fakeSettingReader{values: map[string]int{"security.rate_limit_public": 1, "security.rate_limit_auth": 2}}
	public := NewRequestRateLimiter(settings, "security.rate_limit_public")
	auth := NewRequestRateLimiter(settings, "security.rate_limit_auth")

	for _, test := range []struct {
		name    string
		limiter *RequestRateLimiter
		allowed int
	}{
		{name: "public", limiter: public, allowed: 1},
		{name: "authenticated", limiter: auth, allowed: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := echo.New()
			e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }, test.limiter.Middleware)
			for i := 0; i < test.allowed; i++ {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "192.0.2.20:1234"
				res := httptest.NewRecorder()
				e.ServeHTTP(res, req)
				if res.Code != http.StatusNoContent {
					t.Fatalf("request %d status = %d, want %d", i+1, res.Code, http.StatusNoContent)
				}
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.0.2.20:1234"
			res := httptest.NewRecorder()
			e.ServeHTTP(res, req)
			if res.Code != http.StatusTooManyRequests {
				t.Fatalf("excess request status = %d, want %d", res.Code, http.StatusTooManyRequests)
			}
		})
	}
}

type fakeAccessChecker struct{ listed map[string]bool }

func (f fakeAccessChecker) IsListed(_ context.Context, listType, target, value string) (bool, error) {
	return f.listed[listType+"/"+target+"/"+value], nil
}

type fakeSettingReader struct{ values map[string]int }

func (f fakeSettingReader) GetSettingValue(_ context.Context, key string, dest interface{}) error {
	value, ok := f.values[key]
	if !ok {
		return context.Canceled
	}
	*dest.(*int) = value
	return nil
}
