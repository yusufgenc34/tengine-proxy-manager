package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"tpm/internal/model"
)

const (
	AccessToken  = "access"
	RefreshToken = "refresh"
	TempToken    = "2fa_temp"
)

type JWTClaims struct {
	UserID       uint   `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenType    string `json:"token_type"`
	TokenVersion uint   `json:"token_version"`
	jwt.RegisteredClaims
}

func ParseToken(raw, kind string) (*JWTClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	token, err := jwt.ParseWithClaims(raw, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid || claims.TokenType != kind || claims.UserID == 0 {
		return nil, fmt.Errorf("invalid token type or claims")
	}
	return claims, nil
}

func JWT(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			parts := strings.Fields(c.Request().Header.Get("Authorization"))
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Bearer token required")
			}
			claims, err := ParseToken(parts[1], AccessToken)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired access token")
			}
			var user model.User
			if err := db.First(&user, claims.UserID).Error; err != nil || user.TokenVersion != claims.TokenVersion {
				return echo.NewHTTPError(http.StatusUnauthorized, "Session is no longer valid")
			}
			c.Set("user_id", user.ID)
			c.Set("email", user.Email)
			c.Set("role", user.Role)
			return next(c)
		}
	}
}

func RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Get("role") != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, "Administrator access required")
		}
		return next(c)
	}
}
