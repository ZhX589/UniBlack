package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type appealRetirementServiceStub struct {
	calls     int
	id, actor string
	roles     []string
	err       error
}

func (s *appealRetirementServiceStub) DeleteAppeal(_ context.Context, id, actor string, roles ...string) error {
	s.calls++
	s.id, s.actor, s.roles = id, actor, roles
	return s.err
}

func TestDeleteAppealPassesAuthenticatedActorAndMapsErrors(t *testing.T) {
	for _, tc := range []struct {
		name, actor string
		err         error
		want        int
	}{
		{"success", "owner", nil, http.StatusOK},
		{"unauthenticated", "", nil, http.StatusUnauthorized},
		{"forbidden", "other", fmt.Errorf("wrapped: %w", service.ErrAppealForbidden), http.StatusForbidden},
		{"final", "owner", service.ErrAppealAlreadyResolved, http.StatusConflict},
		{"missing", "owner", service.ErrAppealNotFound, http.StatusNotFound},
		{"database", "owner", errors.New("secret SQL details"), http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &appealRetirementServiceStub{err: tc.err}
			h := &AppealHandler{retirementService: stub}
			recorder := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest(http.MethodDelete, "/appeals/id", strings.NewReader(`{"actor":"victim","roles":["admin"]}`)), recorder)
			c.SetParamNames("id")
			c.SetParamValues("appeal-id")
			c.Set("user_id", tc.actor)
			roles := []string{"moderator"}
			c.Set("roles", roles)
			if err := h.DeleteAppeal(c); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.want)
			}
			if tc.actor == "" {
				if stub.calls != 0 {
					t.Fatal("anonymous request reached service")
				}
			} else if stub.calls != 1 || stub.id != "appeal-id" || stub.actor != tc.actor || !reflect.DeepEqual(stub.roles, roles) {
				t.Fatalf("authenticated arguments = %#v", stub)
			}
			if strings.Contains(recorder.Body.String(), "secret SQL") {
				t.Fatal("internal error leaked")
			}
		})
	}
}
