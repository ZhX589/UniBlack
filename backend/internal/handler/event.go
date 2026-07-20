package handler

import (
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
	"net/http"
)

type EventHandler struct{ service *service.EventService }

func NewEventHandler(s *service.EventService) *EventHandler { return &EventHandler{service: s} }
func (h *EventHandler) Publish(c echo.Context) error {
	var req service.PublishSubjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	userID, _ := c.Get("user_id").(string)
	subject, err := h.service.Publish(c.Request().Context(), req, userID)
	if err != nil {
		if err == service.ErrSubmissionRestricted {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, subject)
}
func (h *EventHandler) Get(c echo.Context) error {
	e, err := h.service.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
	}
	if e.Status != "published" {
		userID, _ := c.Get("user_id").(string)
		roles, _ := c.Get("roles").([]string)
		privileged := false
		for _, role := range roles {
			if role == "admin" || role == "moderator" {
				privileged = true
				break
			}
		}
		if !privileged && (e.SubmittedBy == nil || *e.SubmittedBy != userID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "event not found"})
		}
	}
	return c.JSON(http.StatusOK, e)
}
