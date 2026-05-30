package handler

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"tpm/internal/model"
)

func (h *Handler) ListAuditLogs(c echo.Context) error {
	var logs []model.AuditLog
	var total int64

	query := h.db.Model(&model.AuditLog{})

	// Filter by action
	if action := c.QueryParam("action"); action != "" {
		query = query.Where("action ILIKE ?", "%"+action+"%")
	}

	// Filter by user
	if userID := c.QueryParam("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Filter by date range
	if from := c.QueryParam("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if to := c.QueryParam("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			query = query.Where("created_at < ?", t.Add(24*time.Hour))
		}
	}

	query.Count(&total)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to fetch audit logs", Code: "DB_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) ClearAuditLogs(c echo.Context) error {
	// ?all=true clears everything
	if c.QueryParam("all") == "true" {
		result := h.db.Where("1 = 1").Delete(&model.AuditLog{})
		if result.Error != nil {
			return c.JSON(http.StatusInternalServerError, model.APIError{
				Error: true, Message: "Failed to clear audit logs", Code: "DB_ERROR",
			})
		}
		h.audit.Log(userIDFromContext(c), clientIP(c), "audit.clear_all",
			fmt.Sprintf("Cleared all %d log(s)", result.RowsAffected))
		return c.JSON(http.StatusOK, map[string]any{
			"message":       fmt.Sprintf("Cleared all %d log(s)", result.RowsAffected),
			"deleted_count": result.RowsAffected,
		})
	}

	days := 30
	if d := c.QueryParam("older_than"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	result := h.db.Where("created_at < ?", cutoff).Delete(&model.AuditLog{})
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to clear audit logs", Code: "DB_ERROR",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "audit.clear",
		fmt.Sprintf("Cleared %d log(s) older than %d days", result.RowsAffected, days))

	return c.JSON(http.StatusOK, map[string]any{
		"message":       fmt.Sprintf("Cleared %d log(s) older than %d days", result.RowsAffected, days),
		"deleted_count": result.RowsAffected,
		"older_than":    days,
	})
}

func (h *Handler) GetActionTypes(c echo.Context) error {
	var types []string
	h.db.Model(&model.AuditLog{}).Distinct("action").Pluck("action", &types)
	return c.JSON(http.StatusOK, map[string]any{"types": types})
}

func (h *Handler) GetStats(c echo.Context) error {
	var proxyCount, certCount, userCount int64

	h.db.Model(&model.ProxyHost{}).Count(&proxyCount)
	h.db.Model(&model.Certificate{}).Count(&certCount)
	h.db.Model(&model.User{}).Count(&userCount)

	var enabledCount int64
	h.db.Model(&model.ProxyHost{}).Where("enabled = ?", true).Count(&enabledCount)

	// Expiring certificates (within 30 days)
	var expiringCount int64
	thirtyDaysFromNow := time.Now().Add(30 * 24 * time.Hour)
	h.db.Model(&model.Certificate{}).Where("expires_at IS NOT NULL AND expires_at < ?", thirtyDaysFromNow).Count(&expiringCount)

	// Recent audit logs
	var recentLogs []model.AuditLog
	h.db.Order("created_at DESC").Limit(5).Find(&recentLogs)

	// Tengine version via Server header or error page body
	tengineVersion := "unknown"
	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get("http://tengine:8888/stub_status"); err == nil {
		if s := resp.Header.Get("Server"); s != "" {
			tengineVersion = s
		}
		resp.Body.Close()
	}
	if tengineVersion == "unknown" {
		// Fallback: hit default server, read error page body
		if resp, err := client.Get("http://tengine:80/"); err == nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			s := string(body)
			if idx := strings.Index(s, "Tengine/"); idx != -1 {
				end := strings.IndexAny(s[idx:], "< \n\r")
				if end == -1 {
					end = len(s) - idx
				}
				tengineVersion = strings.TrimSpace(s[idx : idx+end])
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"proxy_hosts_total":     proxyCount,
		"proxy_hosts_enabled":   enabledCount,
		"certificates_total":    certCount,
		"certificates_expiring": expiringCount,
		"users_total":           userCount,
		"recent_logs":           recentLogs,
		"go_version":            runtime.Version(),
		"tengine_version":       tengineVersion,
	})
}
