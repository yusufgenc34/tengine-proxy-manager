package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"

	"net/url"
	"path/filepath"
	"tpm/internal/model"
	"tpm/internal/service"
)

func defaultConfPath() string { return filepath.Join(service.ConfigDir(), "000-default.conf") }

func (h *Handler) GetDefaultServer(c echo.Context) error {
	status := "444"
	body := ""
	data, err := os.ReadFile(defaultConfPath())
	if err == nil {
		s := string(data)
		if strings.Contains(s, "return 301 ") {
			status = "301"
			body = extractReturnValue(s, "301")
		} else if strings.Contains(s, "        return 403;\n    }") {
			status = "403"
		} else if strings.Contains(s, "return 404;") {
			status = "404"
		} else if strings.Contains(s, "return 502;") {
			status = "502"
		} else if strings.Contains(s, "return 444;") {
			status = "444"
		} else {
			// Custom body (200) — extract HTML from the config
			status = "200"
			if strings.Contains(s, "# TPM_CUSTOM_BODY") {
				html, readErr := os.ReadFile(filepath.Join(service.ConfigDir(), "000-default.html"))
				if readErr != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read default response")
				}
				body = string(html)
			} else {
				body = extractCustomBody(s)
			}
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"status": status, "body": body})
}

func (h *Handler) UpdateDefaultServer(c echo.Context) error {
	var req struct {
		Status string `json:"status"`
		Body   string `json:"body"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid request", Code: "BAD_REQUEST"})
	}

	var content string
	changes := map[string][]byte{"000-default.html": nil}
	switch req.Status {
	case "444":
		content = defaultServerBlock("return 444;\n")
	case "403":
		content = defaultServerBlock("return 403;\n")
	case "404":
		content = defaultServerBlock("return 404;\n")
	case "502":
		content = defaultServerBlock("return 502;\n")
	case "301":
		if req.Body == "" {
			return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Redirect URL required", Code: "VALIDATION_ERROR"})
		}
		u, err := url.Parse(req.Body)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || strings.ContainsAny(req.Body, "\r\n;{}\\\"'$") {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid redirect URL")
		}
		content = defaultServerBlock("return 301 \"" + req.Body + "\";\n")
	case "200":
		if req.Body == "" {
			return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Body is required for status 200", Code: "VALIDATION_ERROR"})
		}
		content = customBodyBlock()
		changes["000-default.html"] = []byte(req.Body)
	default:
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid status", Code: "VALIDATION_ERROR"})
	}

	changes["000-default.conf"] = []byte(content)
	if err := h.tengine.Apply(changes); err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{Error: true, Message: "Failed to write config", Code: "STORAGE_ERROR"})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "settings.default_server",
		fmt.Sprintf("Default server: %s", req.Status))
	return c.JSON(http.StatusOK, map[string]string{"status": req.Status, "body": req.Body})
}

func defaultServerBlock(rule string) string {
	return `# Auto-generated — do not edit
server {
    listen 80 default_server;
    server_name _;

    location ^~ /.well-known/acme-challenge/ {
        allow all;
        root /etc/tengine/html;
    }

    location / {
        if ($cf_allow = 0) {
            return 403;
        }
        ` + rule + `
    }
}
`
}

func customBodyBlock() string {
	// Serve the body as a file so quotes, dollar signs and newlines are literal.
	return "# TPM_CUSTOM_BODY\n" + defaultServerBlock(fmt.Sprintf("default_type text/html;\n        root %q;\n        try_files /000-default.html =404;\n", service.ConfigDir()))
}

func extractReturnValue(config, code string) string {
	// "return 301 https://example.com;\n" → "https://example.com"
	s := config
	prefix := "return " + code + " "
	idx := strings.Index(s, prefix)
	if idx == -1 {
		return ""
	}
	s = s[idx+len(prefix):]
	if end := strings.Index(s, ";"); end != -1 {
		s = s[:end]
	}
	return strings.Trim(s, "\"")
}

// GetCloudflareSettings returns Cloudflare IP whitelist configuration.
func (h *Handler) GetCloudflareSettings(c echo.Context) error {
	cfg, err := h.cf.GetSettings()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to load Cloudflare settings", Code: "DB_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"enabled":      cfg.Enabled,
		"ipv4_count":   cfg.IPv4Count(),
		"ipv6_count":   cfg.IPv6Count(),
		"last_fetched": cfg.LastFetched,
		"updated_at":   cfg.UpdatedAt,
	})
}

// UpdateCloudflareSettings enables or disables the Cloudflare IP whitelist.
func (h *Handler) UpdateCloudflareSettings(c echo.Context) error {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	cfg, err := h.cf.Toggle(req.Enabled)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: err.Error(), Code: "CLOUDFLARE_ERROR",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "settings.cloudflare",
		fmt.Sprintf("Cloudflare IP whitelist %s", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]))

	return c.JSON(http.StatusOK, map[string]any{
		"enabled":      cfg.Enabled,
		"ipv4_count":   cfg.IPv4Count(),
		"ipv6_count":   cfg.IPv6Count(),
		"last_fetched": cfg.LastFetched,
		"updated_at":   cfg.UpdatedAt,
	})
}

// RefreshCloudflareIPs manually triggers an IP list refresh.
func (h *Handler) RefreshCloudflareIPs(c echo.Context) error {
	if err := h.cf.RefreshIPs(); err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to refresh Cloudflare IPs: " + err.Error(), Code: "CLOUDFLARE_ERROR",
		})
	}

	cfg, _ := h.cf.GetSettings()

	h.audit.Log(userIDFromContext(c), clientIP(c), "settings.cloudflare.refresh", "Manual IP refresh")

	return c.JSON(http.StatusOK, map[string]any{
		"enabled":      cfg.Enabled,
		"ipv4_count":   cfg.IPv4Count(),
		"ipv6_count":   cfg.IPv6Count(),
		"last_fetched": cfg.LastFetched,
		"updated_at":   cfg.UpdatedAt,
	})
}

func extractCustomBody(config string) string {
	// Extract HTML from: default_type text/html; return 200 '...';
	idx := strings.Index(config, "return 200 '")
	if idx == -1 {
		return config
	}
	s := config[idx+12:]
	// Find the closing single quote — the HTML is single-quoted with escaped quotes
	if end := strings.LastIndex(s, "'"); end != -1 {
		s = s[:end]
	}
	// Unescape
	s = strings.ReplaceAll(s, "'\\''", "'")
	return s
}
