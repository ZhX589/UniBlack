package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type sanctionServiceStub struct {
	err error
}

func (s *sanctionServiceStub) Create(context.Context, service.CreateSanctionRequest, string) (*models.Sanction, error) {
	return &models.Sanction{}, s.err
}
func (s *sanctionServiceStub) Revoke(context.Context, string, string, string) error { return s.err }
func (s *sanctionServiceStub) List(context.Context, int, int, string, bool) ([]models.Sanction, int64, error) {
	return nil, 0, s.err
}
func (s *sanctionServiceStub) Appeal(context.Context, string, string, string) (*models.SanctionAppeal, error) {
	return &models.SanctionAppeal{}, s.err
}
func (s *sanctionServiceStub) ResolveAppeal(context.Context, string, string, string, string) (*models.SanctionAppeal, error) {
	return &models.SanctionAppeal{}, s.err
}

// Every mutating path must map typed command errors to stable HTTP statuses and
// never reflect wrapped SQL or validation internals back to the client.
func TestSanctionHandlerMapsTypedErrorsWithoutLeakingInternals(t *testing.T) {
	cases := []struct {
		name, endpoint, body string
		err                  error
		want                 int
		message              string
		leak                 string // substring that must NOT appear in the response body
	}{
		{"create invalid", "Create", `{"user_id":"u"}`, fmtWrap(repository.ErrSanctionInvalidRequest), 400, "invalid sanction request", "user_id and reason"},
		{"create internal", "Create", `{"user_id":"u"}`, errors.New("secret SQL constraint violation"), 500, "sanction operation failed", "secret SQL"},
		{"revoke missing", "Revoke", `{}`, repository.ErrSanctionNotFound, 404, "sanction not found", ""},
		{"revoke conflict", "Revoke", `{}`, fmtWrap(repository.ErrSanctionRevoked), 409, "sanction already revoked", ""},
		{"appeal forbidden", "Appeal", `{"reason":"r"}`, fmtWrap(repository.ErrSanctionForbidden), 403, "can only appeal own sanctions", ""},
		{"appeal already", "Appeal", `{"reason":"r"}`, fmtWrap(repository.ErrSanctionAlreadyAppealed), 409, "sanction already appealed", ""},
		{"resolve not found", "ResolveAppeal", `{"status":"approved"}`, repository.ErrSanctionAppealNotFound, 404, "sanction appeal not found", ""},
		{"resolve terminal", "ResolveAppeal", `{"status":"approved"}`, fmtWrap(repository.ErrSanctionAppealAlreadyResolved), 409, "sanction appeal already resolved", ""},
		{"resolve internal", "ResolveAppeal", `{"status":"approved"}`, errors.New("deadlock detected at row 42"), 500, "sanction operation failed", "deadlock"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &sanctionServiceStub{err: tc.err}
			h := &SanctionHandler{service: stub}
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set("user_id", "staff")
			var err error
			switch tc.endpoint {
			case "Create":
				err = h.Create(c)
			case "Revoke":
				err = h.Revoke(c)
			case "Appeal":
				err = h.Appeal(c)
			case "ResolveAppeal":
				err = h.ResolveAppeal(c)
			}
			if err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
			body := rec.Body.String()
			if !strings.Contains(body, tc.message) {
				t.Fatalf("response %q missing %q", body, tc.message)
			}
			if tc.leak != "" && strings.Contains(body, tc.leak) {
				t.Fatalf("response leaked internals: %q contains %q", body, tc.leak)
			}
		})
	}
}

func fmtWrap(err error) error { return &wrappedErr{err} }

type wrappedErr struct{ inner error }

func (w *wrappedErr) Error() string { return "wrapped: " + w.inner.Error() }
func (w *wrappedErr) Unwrap() error { return w.inner }
