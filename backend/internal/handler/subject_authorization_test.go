package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func TestSubjectMutationsRejectOrdinaryUsersBeforeDatabaseAccess(t *testing.T) {
	h := NewSubjectHandler(service.NewSubjectService(nil))
	for _, tc := range []struct {
		name, method string
		call         echo.HandlerFunc
	}{
		{"update", http.MethodPut, h.UpdateSubject}, {"delete", http.MethodDelete, h.DeleteSubject},
		{"add identifier", http.MethodPost, h.AddIdentifier}, {"remove identifier", http.MethodDelete, h.RemoveIdentifier},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tc.method, "/api/subjects/target", strings.NewReader(`{"status":"active","roles":["admin"]}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("target")
			c.Set("user_id", "other")
			c.Set("roles", []string{"user"})
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}
