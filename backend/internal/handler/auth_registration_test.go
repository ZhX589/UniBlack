package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func TestRegistrationInvalidInputReturnsBadRequest(t *testing.T) {
	e := echo.New()
	svc := service.NewAuthService(nil, nil, nil, nil, nil, nil)
	h := NewAuthHandler(svc)
	for _, body := range []string{
		`{"username":"","email":"valid@example.test","password":"password123"}`,
		`{"username":"valid","email":"invalid","password":"password123"}`,
		`{"username":"valid","email":"valid@example.test","password":"short"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()
		if err := h.Register(e.NewContext(req, recorder)); err != nil {
			t.Fatal(err)
		}
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
	}
}
