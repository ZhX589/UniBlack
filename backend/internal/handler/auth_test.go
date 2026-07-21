package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func TestRegisterMapsBlacklistLookupFailureToServiceUnavailable(t *testing.T) {
	e := echo.New()
	h := NewAuthHandler(service.NewAuthService(nil, nil, failingRegistrationAccessList{}, nil, nil))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"new-user","email":"new@example.com","password":"password123"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	res := httptest.NewRecorder()
	if err := h.Register(e.NewContext(req, res)); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

type failingRegistrationAccessList struct{}

func (failingRegistrationAccessList) IsListed(context.Context, string, string, string) (bool, error) {
	return false, errors.New("database unavailable")
}
