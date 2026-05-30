package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// SecureHeaders adds security-related HTTP headers to every response.
func SecureHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

			if os.Getenv("ENABLE_HSTS") == "true" {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}

			return next(c)
		}
	}
}

// IPWhitelist restricts access to allowed IPs.
// Set ALLOWED_IPS env var as comma-separated list (e.g. "10.0.0.1,192.168.1.0/24").
// If empty or unset, all IPs are allowed.
func IPWhitelist() echo.MiddlewareFunc {
	raw := os.Getenv("ALLOWED_IPS")
	if raw == "" {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	allowed := make(map[string]bool)
	for _, ip := range strings.Split(raw, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			allowed[ip] = true
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientIP := c.RealIP()
			if !allowed[clientIP] {
				return c.JSON(http.StatusForbidden, map[string]any{
					"error":   true,
					"message": "Access denied: your IP is not allowed",
					"code":    "IP_NOT_ALLOWED",
				})
			}
			return next(c)
		}
	}
}
