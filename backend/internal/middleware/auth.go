package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/ZhX589/UniBlack/backend/internal/service"
)

// AuthMiddleware creates JWT authentication middleware
func AuthMiddleware(authService *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			}

			token := parts[1]

			// Validate token
			identity, err := authService.ValidateToken(c.Request().Context(), token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			// Set user info in context
			c.Set("user_id", identity.UserID)
			c.Set("username", identity.Username)
			c.Set("email", identity.Email)
			c.Set("roles", identity.Roles)

			return next(c)
		}
	}
}

// RequireRole creates middleware that requires specific roles
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRoles, ok := c.Get("roles").([]string)
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "no roles found"})
			}

			// Check if user has any of the required roles
			for _, required := range roles {
				for _, userRole := range userRoles {
					if userRole == required {
						return next(c)
					}
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		}
	}
}

// RequirePermission creates middleware that requires specific permissions
func RequirePermission(authService *service.AuthService, permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := c.Get("user_id").(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			permissions, err := authService.GetUserPermissions(c.Request().Context(), userID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to check permissions"})
			}

			// Check if user has the required permission
			for _, p := range permissions {
				if p.Name == permission {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		}
	}
}
