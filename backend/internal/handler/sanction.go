package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/service"
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

func (h *SanctionHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	userID := c.QueryParam("user_id")
	activeOnly := c.QueryParam("active") == "1" || c.QueryParam("active") == "true"
	rows, total, err := h.service.List(c.Request().Context(), page, pageSize, userID, activeOnly)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"items": rows, "total": total, "page": page, "page_size": pageSize})
}

func (h *SanctionHandler) ListMine(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	rows, total, err := h.service.List(c.Request().Context(), 1, 50, userID, false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"items": rows, "total": total})
}

func (h *SanctionHandler) Appeal(c echo.Context) error {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	userID, _ := c.Get("user_id").(string)
	appeal, err := h.service.Appeal(c.Request().Context(), c.Param("id"), userID, req.Reason)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, appeal)
}

func (h *SanctionHandler) ResolveAppeal(c echo.Context) error {
	var req struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	actor, _ := c.Get("user_id").(string)
	appeal, err := h.service.ResolveAppeal(c.Request().Context(), c.Param("appealID"), actor, req.Status, req.Notes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, appeal)
}
