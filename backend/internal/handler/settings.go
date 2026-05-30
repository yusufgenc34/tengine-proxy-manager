package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"

	"tpm/internal/model"
)

const defaultConfPath = "/etc/tengine/conf.d/000-default.conf"

func (h *Handler) GetDefaultServer(c echo.Context) error {
	status := "444"
	body := ""
	data, err := os.ReadFile(defaultConfPath)
	if err == nil {
		s := string(data)
		if strings.Contains(s, "return 301 ") {
			status = "301"
			body = extractReturnValue(s, "301")
		} else if strings.Contains(s, "return 403;") {
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
			body = extractCustomBody(s)
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
		content = defaultServerBlock("return 301 " + req.Body + ";\n")
	case "200":
		if req.Body == "" {
			return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Body is required for status 200", Code: "VALIDATION_ERROR"})
		}
		content = customBodyBlock(req.Body)
	default:
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid status", Code: "VALIDATION_ERROR"})
	}

	if err := os.WriteFile(defaultConfPath, []byte(content), 0644); err != nil {
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

    location /.well-known/acme-challenge/ {
        root /etc/tengine/html;
    }

    ` + rule + `
}
`
}

func customBodyBlock(html string) string {
	// Escape single quotes for nginx return directive
	escaped := strings.ReplaceAll(html, "'", "'\\''")
	return `# Auto-generated custom response — do not edit
server {
    listen 80 default_server;
    server_name _;

    location /.well-known/acme-challenge/ {
        root /etc/tengine/html;
    }

    location / {
        default_type text/html;
        return 200 '` + escaped + `';
    }
}
`
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
	return s
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
