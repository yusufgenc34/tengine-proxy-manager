package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"tpm/internal/model"
)

const testSecret = "test-only-secret-at-least-32-bytes-long"

func signed(t *testing.T, kind string, method jwt.SigningMethod, expiry *jwt.NumericDate) string {
	t.Helper()
	raw, err := jwt.NewWithClaims(method, JWTClaims{UserID: 1, Role: "admin", TokenType: kind, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: expiry}}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestTokenSeparation(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	expiry := jwt.NewNumericDate(time.Now().Add(time.Hour))
	for _, actual := range []string{AccessToken, RefreshToken, TempToken, ""} {
		raw := signed(t, actual, jwt.SigningMethodHS256, expiry)
		for _, expected := range []string{AccessToken, RefreshToken, TempToken} {
			_, err := ParseToken(raw, expected)
			if (err == nil) != (actual == expected) {
				t.Fatalf("type %q accepted as %q: %v", actual, expected, err)
			}
		}
	}
	for _, raw := range []string{signed(t, AccessToken, jwt.SigningMethodHS384, expiry), signed(t, AccessToken, jwt.SigningMethodHS256, nil), signed(t, AccessToken, jwt.SigningMethodHS256, jwt.NewNumericDate(time.Now().Add(-time.Hour)))} {
		if _, err := ParseToken(raw, AccessToken); err == nil {
			t.Fatal("accepted invalid algorithm or expiry")
		}
	}
}

func TestAuthorizationUsesCurrentAccount(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "db.sqlite")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{ID: 1, Email: "user@example.com", Role: "user", Password: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.GET("/users", func(c echo.Context) error { return c.NoContent(200) }, JWT(db), RequireAdmin)
	request := func(raw string) int {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}
	expiry := jwt.NewNumericDate(time.Now().Add(time.Hour))
	access := signed(t, AccessToken, jwt.SigningMethodHS256, expiry)
	if got := request(access); got != 403 {
		t.Fatalf("stale admin claim authorized user: %d", got)
	}
	db.Model(&user).Update("role", "admin")
	if got := request(access); got != 200 {
		t.Fatalf("admin denied: %d", got)
	}
	for _, kind := range []string{TempToken, RefreshToken} {
		if got := request(signed(t, kind, jwt.SigningMethodHS256, expiry)); got != 401 {
			t.Fatalf("%s reached protected endpoint: %d", kind, got)
		}
	}
	db.Model(&user).Update("token_version", 1)
	if got := request(access); got != 401 {
		t.Fatalf("revoked session accepted: %d", got)
	}
	db.Model(&user).Update("token_version", 0)
	db.Delete(&user)
	if got := request(access); got != 401 {
		t.Fatalf("deleted account accepted: %d", got)
	}
}
