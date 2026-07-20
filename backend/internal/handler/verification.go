package handler

import (
	"net/http"

	"github.com/ZhX589/UniBlack/backend/internal/captcha"
	"github.com/labstack/echo/v4"
)

// VerificationHandler exposes the built-in demonstration captcha token issuer.
// It never proxies to or loads a third-party captcha provider.
type VerificationHandler struct{ demo *captcha.Demo }

func NewVerificationHandler(demo *captcha.Demo) *VerificationHandler {
	return &VerificationHandler{demo: demo}
}

func (h *VerificationHandler) IssueDemoToken(c echo.Context) error {
	var req struct {
		Purpose string `json:"purpose"`
		Session string `json:"session"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Purpose != "register" && req.Purpose != "submission" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid verification purpose"})
	}
	token, err := h.demo.Issue(req.Purpose, req.Session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "unable to issue demo token"})
	}
	return c.JSON(http.StatusOK, map[string]string{"token": token})
}
