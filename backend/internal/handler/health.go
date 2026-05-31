package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// GetHealth returns a comprehensive system health report.
// Public endpoint - no authentication required.
func (h *Handler) GetHealth(c echo.Context) error {
	health := h.validator.ValidateAll()

	// Check database connectivity
	dbHealthy := true
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.Ping() != nil {
		dbHealthy = false
		health.Healthy = false
	}

	return c.JSON(http.StatusOK, map[string]any{
		"healthy":   health.Healthy,
		"db_ok":     dbHealthy,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    health.Checks,
	})
}
