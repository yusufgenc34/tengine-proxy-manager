package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"tpm/internal/model"
)

const setupKeyPath = "/app/setup.key"

// GetSetupStatus returns whether initial setup has been completed
func (h *Handler) GetSetupStatus(c echo.Context) error {
	var count int64
	h.db.Model(&model.User{}).Count(&count)

	return c.JSON(http.StatusOK, map[string]any{
		"setup_required": count == 0,
	})
}

// InitialSetup creates the first admin user (only works if no users exist)
func (h *Handler) InitialSetup(c echo.Context) error {
	var count int64
	h.db.Model(&model.User{}).Count(&count)

	if count > 0 {
		return c.JSON(http.StatusForbidden, model.APIError{
			Error:   true,
			Message: "Setup already completed",
			Code:    "SETUP_ALREADY_DONE",
		})
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		SetupKey string `json:"setup_key"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	// Validate setup key
	expectedKey, err := os.ReadFile(setupKeyPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Setup key not found on server", Code: "SETUP_KEY_ERROR",
		})
	}
	if strings.TrimSpace(body.SetupKey) != strings.TrimSpace(string(expectedKey)) {
		return c.JSON(http.StatusForbidden, model.APIError{
			Error:   true,
			Message: "Invalid setup key",
			Code:    "INVALID_SETUP_KEY",
		})
	}

	if body.Email == "" || body.Password == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "email and password are required", Code: "VALIDATION_ERROR",
		})
	}

	if len(body.Password) < 8 {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Password must be at least 8 characters", Code: "VALIDATION_ERROR",
		})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Password hash error", Code: "HASH_ERROR",
		})
	}

	user := model.User{
		Email:    body.Email,
		Password: string(hashed),
		Role:     "admin",
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to create admin user", Code: "DB_ERROR",
		})
	}

	h.audit.Log(&user.ID, clientIP(c), "setup", "Initial admin user created: "+user.Email)

	return c.JSON(http.StatusCreated, map[string]any{
		"message": "Setup complete. You can now log in.",
		"email":   user.Email,
	})
}
