package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// Produces arbitrarily long whitespace without allocating a body-sized buffer.
type publishSpaces struct{ remaining int64 }

func (s *publishSpaces) Read(p []byte) (int, error) {
	if s.remaining == 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > s.remaining {
		n = s.remaining
	}
	for i := range p[:int(n)] {
		p[i] = ' '
	}
	s.remaining -= n
	return int(n), nil
}

func TestPublishRequestRawByteBudget(t *testing.T) {
	for _, tc := range []struct {
		name      string
		size      int64
		known     bool
		wantLarge bool
	}{
		{"exact limit known", maxPublishRequestBytes, true, false},
		{"exact limit unknown", maxPublishRequestBytes, false, false},
		{"over limit known", maxPublishRequestBytes + 1, true, true},
		{"over limit unknown", maxPublishRequestBytes + 1, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := io.MultiReader(strings.NewReader(`{}`), &publishSpaces{remaining: tc.size - 2})
			req := httptest.NewRequest(http.MethodPost, "/publish", reader)
			if tc.known {
				req.ContentLength = tc.size
			} else {
				req.ContentLength = -1
			}
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			_, err := bindPublishRequest(echo.New().NewContext(req, httptest.NewRecorder()))
			var large *http.MaxBytesError
			if tc.wantLarge {
				if !errors.As(err, &large) {
					t.Fatalf("error=%v want MaxBytesError", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPublishOversizedRequestReturns413BeforeService(t *testing.T) {
	// nil service proves no publishing side effects occur after an oversize bind.
	h := &EventHandler{}
	req := httptest.NewRequest(http.MethodPost, "/publish", strings.NewReader(`{}`))
	req.ContentLength = maxPublishRequestBytes + 1
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	if err := h.Publish(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d", rec.Code)
	}
}
