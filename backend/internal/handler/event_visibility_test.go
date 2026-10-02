package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/service"
)

type eventReadStub struct {
	eventService
	event        *models.Event
	err          error
	versions     []models.EventVersion
	versionCalls int
}

func (s *eventReadStub) Get(context.Context, string) (*models.Event, error) { return s.event, s.err }
func (s *eventReadStub) GetVersions(context.Context, string) ([]models.EventVersion, error) {
	s.versionCalls++
	return s.versions, nil
}

func visibilityContext(userID string, roles []string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	r := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/?platform=test&value=name&q=name", nil), r)
	c.SetParamNames("id")
	c.SetParamValues("event-id")
	c.Set("user_id", userID)
	c.Set("roles", roles)
	return c, r
}

func TestEventGetAndVersionsShareVisibility(t *testing.T) {
	owner := "owner"
	for _, status := range []string{"published", "corrected", "draft", "hidden", "withdrawn", "unknown"} {
		for _, viewer := range []struct {
			name, id string
			roles    []string
			private  bool
		}{
			{"anonymous", "", nil, false}, {"other", "other", nil, false},
			{"owner", owner, nil, true}, {"admin", "staff", []string{"admin"}, true},
			{"moderator", "staff", []string{"moderator"}, true},
		} {
			t.Run(status+"/"+viewer.name, func(t *testing.T) {
				stub := &eventReadStub{event: &models.Event{Status: status, SubmittedBy: &owner}, versions: []models.EventVersion{{Details: "historical secret"}}}
				h := &EventHandler{service: stub}
				want := http.StatusNotFound
				if status == "published" || status == "corrected" || viewer.private {
					want = http.StatusOK
				}
				for _, call := range []echo.HandlerFunc{h.Get, h.GetVersions} {
					c, r := visibilityContext(viewer.id, viewer.roles)
					if err := call(c); err != nil {
						t.Fatal(err)
					}
					if r.Code != want {
						t.Fatalf("status = %d, want %d: %s", r.Code, want, r.Body.String())
					}
					if want == http.StatusNotFound && r.Body.String() != "{\"error\":\"event not found\"}\n" {
						t.Fatalf("denial leaked payload: %s", r.Body.String())
					}
				}
				if want == http.StatusNotFound && stub.versionCalls != 0 {
					t.Fatal("unauthorized history was loaded")
				}
			})
		}
	}
}

func TestEventVersionsAuthorizesBeforeEmptyHistory(t *testing.T) {
	for _, stub := range []*eventReadStub{
		{err: errors.New("missing")},
		{event: &models.Event{Status: "hidden"}},
		{event: &models.Event{Status: "hidden", SubmittedBy: new(string)}},
	} {
		c, r := visibilityContext("", nil)
		if err := (&EventHandler{service: stub}).GetVersions(c); err != nil {
			t.Fatal(err)
		}
		if r.Code != http.StatusNotFound || stub.versionCalls != 0 {
			t.Fatalf("unreadable empty history: status=%d calls=%d", r.Code, stub.versionCalls)
		}
	}
	stub := &eventReadStub{event: &models.Event{Status: "published"}}
	c, r := visibilityContext("", nil)
	if err := (&EventHandler{service: stub}).GetVersions(c); err != nil {
		t.Fatal(err)
	}
	if r.Code != http.StatusOK || r.Body.String() != "{\"versions\":[]}\n" {
		t.Fatalf("empty readable history: %d %s", r.Code, r.Body.String())
	}
}

type subjectReadStub struct {
	subjectService
	subject *models.Subject
}

func (s *subjectReadStub) GetSubject(context.Context, string) (*models.Subject, error) {
	return s.subject, nil
}
func (s *subjectReadStub) UpdateSubject(context.Context, string, service.UpdateSubjectRequest, string, []string) (*models.Subject, error) {
	return s.subject, nil
}
func (s *subjectReadStub) GetSubjectByIdentifier(context.Context, string, string) (*models.Subject, error) {
	return s.subject, nil
}
func (s *subjectReadStub) ListSubjects(context.Context, int, int, string) ([]models.Subject, int64, error) {
	return []models.Subject{*s.subject}, 17, nil
}
func (s *subjectReadStub) SearchSubjects(context.Context, string) ([]models.Subject, error) {
	return []models.Subject{*s.subject}, nil
}

func TestSubjectReadsFilterEmbeddedEvents(t *testing.T) {
	owner, other := "owner", "other"
	subject := &models.Subject{ID: "subject", Status: "undefined-status", CaseCount: 7,
		Cases: []models.Case{{ID: "legacy-case", Status: "draft"}},
		Events: []models.Event{
			{ID: "public", Status: "published"}, {ID: "corrected", Status: "corrected"},
			{ID: "own", Status: "hidden", SubmittedBy: &owner}, {ID: "other", Status: "withdrawn", SubmittedBy: &other},
			{ID: "unowned", Status: "unknown"},
		},
	}
	h := &SubjectHandler{subjectService: &subjectReadStub{subject: subject}}
	for _, viewer := range []struct {
		name, id string
		roles    []string
		count    int
	}{
		{"anonymous", "", nil, 2}, {"unrelated", "unrelated", nil, 2},
		{"owner", owner, nil, 3}, {"admin", "staff", []string{"admin"}, 5},
		{"moderator", "staff", []string{"moderator"}, 5},
	} {
		for _, endpoint := range []struct {
			name string
			call echo.HandlerFunc
			list bool
		}{
			{"get", h.GetSubject, false}, {"identifier", h.GetSubjectByIdentifier, false},
			{"list", h.ListSubjects, true}, {"search", h.SearchSubjects, true},
			{"update", h.UpdateSubject, false},
		} {
			t.Run(endpoint.name+"/"+viewer.name, func(t *testing.T) {
				c, r := visibilityContext(viewer.id, viewer.roles)
				if endpoint.name == "update" {
					c.Request().Method = http.MethodPatch
					c.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
					c.Request().Body = io.NopCloser(strings.NewReader("{}"))
				}
				if err := endpoint.call(c); err != nil {
					t.Fatal(err)
				}
				if r.Code != http.StatusOK {
					t.Fatalf("status=%d", r.Code)
				}
				var got models.Subject
				if endpoint.list {
					var body struct {
						Subjects []models.Subject `json:"subjects"`
						Total    int              `json:"total"`
					}
					if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					if len(body.Subjects) != 1 {
						t.Fatal("subject membership changed")
					}
					got = body.Subjects[0]
					if endpoint.name == "list" && body.Total != 17 {
						t.Fatal("pagination total changed")
					}
				} else if err := json.Unmarshal(r.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if len(got.Events) != viewer.count {
					t.Fatalf("events=%v, want count %d", got.Events, viewer.count)
				}
				for i := range got.Events {
					if !eventVisibleToRequester(c, &got.Events[i]) {
						t.Fatal("unreadable event leaked")
					}
				}
				if got.Status != subject.Status || got.CaseCount != 7 || len(got.Cases) != 1 {
					t.Fatal("subject/legacy Case semantics changed")
				}
				if len(subject.Events) != 5 {
					t.Fatal("service result mutated")
				}
			})
		}
	}
}
