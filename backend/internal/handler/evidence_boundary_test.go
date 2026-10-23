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
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type evidenceBoundaryRepo struct {
	row *models.Evidence
	err error
}

func (*evidenceBoundaryRepo) CreateEvidence(context.Context, *models.Evidence) error { return nil }
func (r *evidenceBoundaryRepo) GetEvidenceByID(context.Context, string) (*models.Evidence, error) {
	return r.row, r.err
}
func (*evidenceBoundaryRepo) GetEvidenceByCaseID(context.Context, string) ([]models.Evidence, error) {
	return nil, nil
}
func (*evidenceBoundaryRepo) GetEvidenceBySHA256(context.Context, string) (*models.Evidence, error) {
	return nil, nil
}
func (*evidenceBoundaryRepo) DeleteEvidence(context.Context, string) error { return nil }

func TestEvidenceReadBoundary(t *testing.T) {
	eventID := "restricted-event"
	for _, tc := range []struct {
		name string
		row  *models.Evidence
		err  error
		want int
	}{
		{"repository not found", nil, repository.ErrEvidenceNotFound, 404},
		{"wrapped repository not found", nil, fmt.Errorf("lookup: %w", repository.ErrEvidenceNotFound), 404},
		{"internal error", nil, errors.New("secret SQL credentials"), 500},
		{"nil repository row", nil, nil, 404},
		{"event policy missing", &models.Evidence{EventID: &eventID}, nil, 503},
		{"legacy compatibility", &models.Evidence{ID: "legacy"}, nil, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewEvidenceHandler(service.NewEvidenceService(&evidenceBoundaryRepo{row: tc.row, err: tc.err}, nil, nil))
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/evidence/id", nil), rec)
			c.SetParamNames("id")
			c.SetParamValues("id")
			if err := h.GetEvidence(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.want {
				t.Fatalf("status=%d want=%d", rec.Code, tc.want)
			}
			if strings.Contains(rec.Body.String(), "secret SQL") {
				t.Fatal("internal details leaked")
			}
		})
	}
}

func TestEvidenceDeleteUsesRepositoryNotFoundIdentity(t *testing.T) {
	h := NewEvidenceHandler(service.NewEvidenceService(&evidenceBoundaryRepo{err: fmt.Errorf("lookup: %w", repository.ErrEvidenceNotFound)}, nil, nil))
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodDelete, "/evidence/id", nil), rec)
	c.Set("user_id", "actor")
	c.SetParamNames("id")
	c.SetParamValues("id")
	if err := h.DeleteEvidence(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 404 {
		t.Fatalf("status=%d want=404", rec.Code)
	}
}
