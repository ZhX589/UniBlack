package handler

import (
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
	"net/http"
)

type SanctionHandler struct{ service *service.SanctionService }

func NewSanctionHandler(s *service.SanctionService) *SanctionHandler {
	return &SanctionHandler{service: s}
}
func (h *SanctionHandler) Create(c echo.Context) error {
	var req service.CreateSanctionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	actor, _ := c.Get("user_id").(string)
	v, err := h.service.Create(c.Request().Context(), req, actor)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, v)
}
func (h *SanctionHandler) Revoke(c echo.Context) error {
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)
	actor, _ := c.Get("user_id").(string)
	if err := h.service.Revoke(c.Request().Context(), c.Param("id"), actor, req.Reason); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "sanction revoked"})
}
