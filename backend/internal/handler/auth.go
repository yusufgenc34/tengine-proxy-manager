package handler

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	mw "tpm/internal/middleware"
	"tpm/internal/model"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error:   true,
			Message: "Invalid request",
			Code:    "BAD_REQUEST",
		})
	}

	var user model.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		h.audit.Log(nil, clientIP(c), "login.failed", "Failed login attempt for: "+req.Email)
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error:   true,
			Message: "Invalid email or password",
			Code:    "INVALID_CREDENTIALS",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.audit.Log(nil, clientIP(c), "login.failed", "Failed login attempt for: "+req.Email)
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error:   true,
			Message: "Invalid email or password",
			Code:    "INVALID_CREDENTIALS",
		})
	}

	// 2FA check
	if user.TwoFactorEnabled {
		if req.TOTPCode == "" {
			// Password OK but 2FA required — return temp token
			tempToken, err := generateTempToken(user)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, model.APIError{
					Error: true, Message: "Failed to generate token", Code: "TOKEN_ERROR",
				})
			}
			return c.JSON(http.StatusOK, map[string]any{
				"two_factor_required": true,
				"temp_token":          tempToken,
			})
		}

		if !totp.Validate(req.TOTPCode, user.TwoFactorSecret) {
			h.audit.Log(&user.ID, clientIP(c), "login.2fa_failed", "Invalid 2FA code for: "+user.Email)
			return c.JSON(http.StatusUnauthorized, model.APIError{
				Error:   true,
				Message: "Invalid 2FA code",
				Code:    "INVALID_2FA_CODE",
			})
		}
	}

	h.audit.Log(&user.ID, clientIP(c), "login", "User logged in: "+user.Email)
	return h.issueTokens(c, user)
}

// Verify2FALogin handles the second step of 2FA login
func (h *Handler) Verify2FALogin(c echo.Context) error {
	var req struct {
		TempToken string `json:"temp_token"`
		TOTPCode  string `json:"totp_code"`
	}
	if err := c.Bind(&req); err != nil || req.TempToken == "" || req.TOTPCode == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "temp_token and totp_code are required", Code: "BAD_REQUEST",
		})
	}

	token, err := jwt.ParseWithClaims(req.TempToken, &mw.JWTClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error: true, Message: "Invalid or expired temp token", Code: "INVALID_TEMP_TOKEN",
		})
	}

	claims := token.Claims.(*mw.JWTClaims)
	if claims.TokenType != "2fa_temp" {
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error: true, Message: "Invalid token type", Code: "INVALID_TOKEN_TYPE",
		})
	}

	var user model.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	if !totp.Validate(req.TOTPCode, user.TwoFactorSecret) {
		h.audit.Log(&user.ID, clientIP(c), "login.2fa_failed", "Invalid 2FA code for: "+user.Email)
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error: true, Message: "Invalid 2FA code", Code: "INVALID_2FA_CODE",
		})
	}

	h.audit.Log(&user.ID, clientIP(c), "login", "User logged in (2FA): "+user.Email)
	return h.issueTokens(c, user)
}

func (h *Handler) issueTokens(c echo.Context, user model.User) error {
	accessToken, err := generateToken(user, 15*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to generate token", Code: "TOKEN_ERROR",
		})
	}

	refreshToken, err := generateToken(user, 7*24*time.Hour)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to generate token", Code: "TOKEN_ERROR",
		})
	}

	return c.JSON(http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * time.Minute / time.Second),
	})
}

func (h *Handler) Refresh(c echo.Context) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&body); err != nil || body.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error:   true,
			Message: "Refresh token required",
			Code:    "BAD_REQUEST",
		})
	}

	token, err := jwt.ParseWithClaims(body.RefreshToken, &mw.JWTClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error:   true,
			Message: "Invalid refresh token",
			Code:    "INVALID_REFRESH_TOKEN",
		})
	}

	claims := token.Claims.(*mw.JWTClaims)

	var user model.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error:   true,
			Message: "User not found",
			Code:    "USER_NOT_FOUND",
		})
	}

	accessToken, err := generateToken(user, 15*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error:   true,
			Message: "Failed to generate token",
			Code:    "TOKEN_ERROR",
		})
	}

	return c.JSON(http.StatusOK, tokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(15 * time.Minute / time.Second),
	})
}

// Setup2FA generates a TOTP secret and returns a QR code
func (h *Handler) Setup2FA(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	if user.TwoFactorEnabled {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "2FA is already enabled", Code: "2FA_ALREADY_ENABLED",
		})
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "TengineProxyManager",
		AccountName: user.Email,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to generate 2FA secret", Code: "2FA_GENERATE_ERROR",
		})
	}

	// Save secret (not yet enabled)
	h.db.Model(&user).Update("two_factor_secret", key.Secret())

	// Generate QR code as base64
	img, err := key.Image(200, 200)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to generate QR code", Code: "QR_ERROR",
		})
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	qrBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	return c.JSON(http.StatusOK, map[string]any{
		"secret":   key.Secret(),
		"qr_code":  "data:image/png;base64," + qrBase64,
		"otpauth":  key.URL(),
	})
}

// Verify2FA verifies a TOTP code and enables 2FA
func (h *Handler) Verify2FA(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var body struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&body); err != nil || body.Code == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "code is required", Code: "BAD_REQUEST",
		})
	}

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	if user.TwoFactorSecret == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "2FA setup not initiated. Call /auth/2fa/setup first.", Code: "2FA_NOT_SETUP",
		})
	}

	if !totp.Validate(body.Code, user.TwoFactorSecret) {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid verification code", Code: "INVALID_2FA_CODE",
		})
	}

	h.db.Model(&user).Update("two_factor_enabled", true)
	h.audit.Log(&user.ID, clientIP(c), "2fa.enabled", "2FA enabled for: "+user.Email)

	return c.JSON(http.StatusOK, map[string]any{"message": "2FA enabled successfully"})
}

// Disable2FA turns off 2FA for the user
func (h *Handler) Disable2FA(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var body struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&body); err != nil || body.Code == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "code is required", Code: "BAD_REQUEST",
		})
	}

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	if !user.TwoFactorEnabled {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "2FA is not enabled", Code: "2FA_NOT_ENABLED",
		})
	}

	if !totp.Validate(body.Code, user.TwoFactorSecret) {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid verification code", Code: "INVALID_2FA_CODE",
		})
	}

	h.db.Model(&user).Updates(map[string]any{
		"two_factor_enabled": false,
		"two_factor_secret":  "",
	})
	h.audit.Log(&user.ID, clientIP(c), "2fa.disabled", "2FA disabled for: "+user.Email)

	return c.JSON(http.StatusOK, map[string]any{"message": "2FA disabled successfully"})
}

// Get2FAStatus returns whether 2FA is enabled for the current user
func (h *Handler) Get2FAStatus(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"two_factor_enabled": user.TwoFactorEnabled,
	})
}

// ChangePassword allows the logged-in user to change their own password.
func (h *Handler) ChangePassword(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "current_password and new_password are required", Code: "VALIDATION_ERROR",
		})
	}

	if len(body.NewPassword) < 8 {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "New password must be at least 8 characters", Code: "VALIDATION_ERROR",
		})
	}

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.CurrentPassword)); err != nil {
		return c.JSON(http.StatusUnauthorized, model.APIError{
			Error: true, Message: "Current password is incorrect", Code: "WRONG_PASSWORD",
		})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to hash password", Code: "HASH_ERROR",
		})
	}

	h.db.Model(&user).Update("password", string(hashed))
	h.audit.Log(&user.ID, clientIP(c), "password.changed", "Password changed for: "+user.Email)

	return c.JSON(http.StatusOK, map[string]any{"message": "Password changed successfully"})
}

func generateToken(user model.User, duration time.Duration) (string, error) {
	claims := mw.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func generateTempToken(user model.User) (string, error) {
	claims := mw.JWTClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: "2fa_temp",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
