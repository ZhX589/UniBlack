package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type appealReviewServiceStub struct {
	calls     int
	actor, id string
	req       service.ReviewAppealRequest
	err       error
}

func (s *appealReviewServiceStub) ReviewAppeal(_ context.Context, id string, req service.ReviewAppealRequest, actor string) (*models.Appeal, error) {
	s.calls++
	s.id, s.req, s.actor = id, req, actor
	return &models.Appeal{Status: req.Status}, s.err
}
func TestLegacyReviewAppealHTTPBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body, actor string
		err               error
		want, calls       int
		message           string
	}{
		{"approved", `{"status":"approved","review_notes":"notes","reviewed_by":"victim"}`, "staff", nil, 200, 1, "approved"},
		{"rejected", `{"status":"rejected"}`, "staff", nil, 200, 1, "rejected"},
		{"invalid status", `{"status":"pending"}`, "staff", nil, 400, 0, "approved or rejected"},
		{"invalid JSON", `{`, "staff", nil, 400, 0, "invalid request"},
		{"anonymous", `{"status":"approved"}`, "", nil, 401, 0, "unauthorized"},
		{"missing", `{"status":"approved"}`, "staff", fmt.Errorf("wrapped: %w", service.ErrAppealNotFound), 404, 1, "appeal not found"},
		{"final", `{"status":"approved"}`, "staff", fmt.Errorf("wrapped: %w", service.ErrAppealAlreadyResolved), 409, 1, "already resolved"},
		{"event", `{"status":"approved"}`, "staff", service.ErrAppealResolveRequired, 409, 1, "formal resolve"},
		{"target", `{"status":"approved"}`, "staff", service.ErrAppealInvalidTarget, 409, 1, "case target"},
		{"repository validation", `{"status":"approved"}`, "staff", service.ErrAppealInvalidReview, 400, 1, "approved or rejected"},
		{"internal", `{"status":"approved"}`, "staff", errors.New("secret SQL details"), 500, 1, "failed to review appeal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &appealReviewServiceStub{err: tc.err}
			h := &AppealHandler{reviewService: stub}
			req := httptest.NewRequest(http.MethodPost, "/appeals/id/review", strings.NewReader(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("appeal-id")
			c.Set("user_id", tc.actor)
			if err := h.ReviewAppeal(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.want || stub.calls != tc.calls || !strings.Contains(rec.Body.String(), tc.message) || strings.Contains(rec.Body.String(), "secret SQL") {
				t.Fatalf("response %d %s, calls %d", rec.Code, rec.Body.String(), stub.calls)
			}
			if stub.calls > 0 && (stub.actor != tc.actor || stub.id != "appeal-id") {
				t.Fatalf("arguments %#v", stub)
			}
		})
	}
}
