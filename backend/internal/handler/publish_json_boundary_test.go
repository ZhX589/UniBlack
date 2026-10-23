package handler

import (
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishJSONRequiresOneCompletePayload(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"one object", `{"display_name":"proof"}`, false},
		{"null is not object", `null`, true},
		{"array is not object", `[]`, true},
		{"scalar is not object", `true`, true},
		{"whitespace", "{} \n\t", false},
		{"second object", `{} {}`, true},
		{"second scalar", `{} true`, true},
		{"trailing garbage", `{} invalid`, true},
		{"truncated object", `{"display_name":`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/publish", strings.NewReader(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			_, err := bindPublishRequest(echo.New().NewContext(req, httptest.NewRecorder()))
			if (err != nil) != tc.wantError {
				t.Fatalf("error=%v wantError=%v", err, tc.wantError)
			}
		})
	}
}

func TestPublishUnknownLengthOversizeJSONMapsTo413(t *testing.T) {
	for _, tc := range []struct {
		name string
		body io.Reader
	}{
		{"inside first value", io.MultiReader(strings.NewReader(`{"display_name":"`), &publishSpaces{remaining: maxPublishRequestBytes}, strings.NewReader(`"}`))},
		{"after first value", io.MultiReader(strings.NewReader(`{}`), &publishSpaces{remaining: maxPublishRequestBytes})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/publish", tc.body)
			req.ContentLength = -1
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			if err := (&EventHandler{}).Publish(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 413 {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
