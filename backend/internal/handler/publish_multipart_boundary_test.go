package handler

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestPublishMultipartRejectsAmbiguousFileMappings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		payloads []string
		fields   []string
		want     string
	}{
		{"duplicate payload", []string{`{}`, `{}`}, nil, "exactly one"},
		{"duplicate file field", []string{`{}`}, []string{"file_0_0", "file_0_0"}, "duplicate multipart"},
		{"duplicate reference", []string{`{"file_evidence":[{"field":"proof"},{"field":"proof"}]}`}, []string{"proof"}, "duplicate file reference"},
		{"unreferenced file", []string{`{"file_evidence":[{"field":"proof"}]}`}, []string{"proof", "extra"}, "unreferenced"},
		{"unknown auto field", []string{`{}`}, []string{"extra"}, "invalid multipart file field"},
		{"negative event index", []string{`{}`}, []string{"file_-1_0"}, "invalid multipart event index"},
		{"invalid file index", []string{`{}`}, []string{"file_0_x"}, "invalid multipart file index"},
		{"explicit valid", []string{`{"file_evidence":[{"field":"proof"}]}`}, []string{"proof"}, ""},
		{"auto valid", []string{`{}`}, []string{"file_0_0"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			w := multipart.NewWriter(&body)
			for _, p := range tc.payloads {
				if err := w.WriteField("payload", p); err != nil {
					t.Fatal(err)
				}
			}
			for _, field := range tc.fields {
				part, err := w.CreateFormFile(field, "proof.txt")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.WriteString(part, "proof"); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/publish", &body)
			req.Header.Set(echo.HeaderContentType, w.FormDataContentType())
			c := echo.New().NewContext(req, httptest.NewRecorder())
			result, err := bindPublishRequest(c)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error=%v want %s", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(result.FileEvidence) != 1 || string(result.FileEvidence[0].Content) != "proof" {
				t.Fatal("valid file mapping changed")
			}
		})
	}
}

func TestPublishMultipartFileCountBudget(t *testing.T) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("payload", `{}`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 65; i++ {
		part, err := w.CreateFormFile("file_0_"+strconv.Itoa(i), "proof.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, "x"); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/publish", &body)
	req.Header.Set(echo.HeaderContentType, w.FormDataContentType())
	_, err := bindPublishRequest(echo.New().NewContext(req, httptest.NewRecorder()))
	if err == nil || !strings.Contains(err.Error(), "too many evidence files") {
		t.Fatalf("error=%v", err)
	}
}
