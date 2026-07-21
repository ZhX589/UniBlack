package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

const whitelistBypassKey = "rate_limit_whitelist"

// AccessListReader keeps repository access out of handlers and middleware.
type AccessListReader interface {
	IsListed(ctx context.Context, listType, target, value string) (bool, error)
}

// AccessList rejects blacklisted clients and marks whitelisted IPs for limit bypass.
func AccessList(reader AccessListReader) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			for _, candidate := range []struct{ target, value string }{
				{"ip", ip},
				{"email", contextString(c, "email")},
				{"username", contextString(c, "username")},
			} {
				if candidate.value == "" {
					continue
				}
				listed, err := reader.IsListed(c.Request().Context(), "blacklist", candidate.target, candidate.value)
				if err != nil {
					return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "access control unavailable"})
				}
				if listed {
					return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
				}
			}
			whitelisted, err := reader.IsListed(c.Request().Context(), "whitelist", "ip", ip)
			if err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "access control unavailable"})
			}
			if whitelisted {
				c.Set(whitelistBypassKey, true)
			}
			return next(c)
		}
	}
}

// RateLimit lets whitelisted clients bypass an underlying Echo limiter.
func RateLimit(limiter echo.MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		limited := limiter(next)
		return func(c echo.Context) error {
			if c.Get(whitelistBypassKey) == true {
				return next(c)
			}
			return limited(c)
		}
	}
}

// RequestRateLimiter provides a small per-IP fixed-window limiter whose limit
// is read for each request, allowing settings changes to take effect directly.
type RequestRateLimiter struct {
	reader interface {
		GetSettingValue(context.Context, string, interface{}) error
	}
	key    string
	mu     sync.Mutex
	counts map[string]rateWindow
}

type rateWindow struct {
	second int64
	count  int
}

func NewRequestRateLimiter(reader interface {
	GetSettingValue(context.Context, string, interface{}) error
}, key string) *RequestRateLimiter {
	return &RequestRateLimiter{reader: reader, key: key, counts: make(map[string]rateWindow)}
}

func (l *RequestRateLimiter) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Get(whitelistBypassKey) == true {
			return next(c)
		}
		limit := 0
		if err := l.reader.GetSettingValue(c.Request().Context(), l.key, &limit); err != nil || limit <= 0 {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "rate limit unavailable"})
		}
		now := time.Now().Unix()
		ip := c.RealIP()
		l.mu.Lock()
		window := l.counts[ip]
		if window.second != now {
			window = rateWindow{second: now}
		}
		window.count++
		l.counts[ip] = window
		l.mu.Unlock()
		if window.count > limit {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}
		return next(c)
	}
}

func contextString(c echo.Context, key string) string {
	value, _ := c.Get(key).(string)
	return value
}
