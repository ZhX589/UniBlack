package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/captcha"
)

// VerificationHandler exposes the built-in demonstration captcha token issuer.
// It never proxies to or loads a third-party captcha provider.
type VerificationHandler struct{ demo *captcha.Demo }

func NewVerificationHandler(demo *captcha.Demo) *VerificationHandler {
	return &VerificationHandler{demo: demo}
}

func (h *VerificationHandler) issue(c echo.Context, purpose string) error {
	// The browser cannot choose the binding. Registration uses the request IP;
	// authenticated flows use the JWT subject, preventing cross-user reuse.
	session := c.RealIP()
	if userID, ok := c.Get("user_id").(string); ok && userID != "" {
		session = userID
	}
	token, err := h.demo.Issue(purpose, session)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "unable to issue demo token"})
	}
	return c.JSON(http.StatusOK, map[string]string{"token": token})
}

// IssueRegisterDemoToken is public and binds the token to the requester IP.
func (h *VerificationHandler) IssueRegisterDemoToken(c echo.Context) error {
	return h.issue(c, "register")
}

// IssueSubmissionDemoToken is protected and binds the token to the JWT user ID.
func (h *VerificationHandler) IssueSubmissionDemoToken(c echo.Context) error {
	return h.issue(c, "submission")
}
