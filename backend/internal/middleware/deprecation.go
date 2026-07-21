package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// CaseSunset is the shared end date for legacy Case API compatibility.
const CaseSunset = "Thu, 31 Dec 2026 23:59:59 GMT"

// CaseDeprecation adds RFC-compatible migration guidance to legacy Case routes.
// An empty successor retains deprecation headers without a Link because no direct
// Event replacement exists. Echo-style route parameters are resolved per request.
func CaseDeprecation(successor string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Deprecation", "true")
			c.Response().Header().Set("Sunset", CaseSunset)
			if successor != "" {
				replacement := successor
				for _, name := range c.ParamNames() {
					replacement = strings.ReplaceAll(replacement, ":"+name, c.Param(name))
				}
				c.Response().Header().Set("Link", "<"+replacement+">; rel=\"successor-version\"")
			}
			c.Response().Header().Set("Warning", "299 - \"Case API is deprecated; migrate to Event API before sunset\"")
			return next(c)
		}
	}
}
