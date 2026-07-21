package handler

import (
	"errors"
	"net/http"

	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register handles user registration
func (h *AuthHandler) Register(c echo.Context) error {
	var req service.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Get client IP
	ip := c.RealIP()

	user, err := h.authService.Register(c.Request().Context(), req, ip)
	if err != nil {
		if err == service.ErrRegistrationClosed {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "registration is closed"})
		}
		if err == service.ErrInvalidCaptcha {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid captcha"})
		}
		if err == service.ErrInvalidCode {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid verification code"})
		}
		if errors.Is(err, service.ErrAccessControlUnavailable) {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "registration unavailable"})
		}
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"user": user,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c echo.Context) error {
	var req service.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tokens, err := h.authService.Login(c.Request().Context(), req)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		}
		if err == service.ErrUserInactive {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "user is inactive"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "login failed"})
	}

	return c.JSON(http.StatusOK, tokens)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	tokens, err := h.authService.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
	}

	return c.JSON(http.StatusOK, tokens)
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	permissions, err := h.authService.GetUserPermissions(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get permissions"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":     c.Get("user_id"),
		"username":    c.Get("username"),
		"email":       c.Get("email"),
		"roles":       c.Get("roles"),
		"permissions": permissions,
	})
}

// SendVerificationCode sends a verification code to email.
// Optional purpose: register (default), submission, appeal.
func (h *AuthHandler) SendVerificationCode(c echo.Context) error {
	var req struct {
		Email   string `json:"email" validate:"required,email"`
		Purpose string `json:"purpose"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	purpose := req.Purpose
	if purpose == "" {
		purpose = "register"
	}
	if purpose != "register" && purpose != "submission" && purpose != "appeal" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid purpose"})
	}
	// Authenticated purposes must not accept arbitrary emails from anonymous callers.
	if purpose != "register" {
		if _, ok := c.Get("user_id").(string); !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "login required"})
		}
	}

	if err := h.authService.SendVerificationCodeForPurpose(c.Request().Context(), req.Email, purpose); err != nil {
		if err == service.ErrVerificationRateLimited {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "verification code sent"})
}

// VerifyEmail verifies an email with a code
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req struct {
		Email string `json:"email" validate:"required,email"`
		Code  string `json:"code" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.authService.VerifyEmail(c.Request().Context(), req.Email, req.Code); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "email verified"})
}
