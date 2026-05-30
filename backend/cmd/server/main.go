package main

import (
	"log"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"tpm/internal/database"
	"tpm/internal/handler"
	mw "tpm/internal/middleware"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(mw.SecureHeaders())
	e.Use(mw.IPWhitelist())

	// CORS — only allow configured origins
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}
	if extra := os.Getenv("CORS_ORIGINS"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	h := handler.New(db)

	api := e.Group("/api/v1")

	// Rate limiters
	loginLimiter := mw.NewRateLimiter(10, 5)   // 10/min, burst 5
	apiLimiter := mw.NewRateLimiter(120, 30)    // 120/min, burst 30

	// Public (with strict rate limiting on auth endpoints)
	api.GET("/setup/status", h.GetSetupStatus)
	api.POST("/setup", h.InitialSetup, loginLimiter.Middleware())
	api.POST("/auth/login", h.Login, loginLimiter.Middleware())
	api.POST("/auth/2fa/login", h.Verify2FALogin, loginLimiter.Middleware())
	api.POST("/auth/refresh", h.Refresh)

	// Protected (with general rate limiting)
	r := api.Group("", mw.JWT(), apiLimiter.Middleware())

	r.GET("/proxy-hosts", h.ListProxyHosts)
	r.POST("/proxy-hosts", h.CreateProxyHost)
	r.PUT("/proxy-hosts/:id", h.UpdateProxyHost)
	r.DELETE("/proxy-hosts/:id", h.DeleteProxyHost)
	r.POST("/proxy-hosts/:id/enable", h.EnableProxyHost)
	r.POST("/proxy-hosts/:id/disable", h.DisableProxyHost)

	r.GET("/certificates", h.ListCertificates)
	r.POST("/certificates/letsencrypt", h.CreateLetsEncrypt)
	r.POST("/certificates/custom", h.UploadCustomCert)
	r.POST("/certificates/self-signed", h.CreateSelfSignedCert)
	r.GET("/certificates/:id/download", h.DownloadCertificate)
	r.DELETE("/certificates/:id", h.DeleteCertificate)
	r.POST("/certificates/:id/renew", h.RenewCertificate)

	r.GET("/access-lists", h.ListAccessLists)
	r.POST("/access-lists", h.CreateAccessList)
	r.POST("/access-lists/parse", h.ParseAccessListExpression)
	r.PUT("/access-lists/:id", h.UpdateAccessList)
	r.DELETE("/access-lists/:id", h.DeleteAccessList)

	r.GET("/users", h.ListUsers)
	r.POST("/users", h.CreateUser)
	r.PUT("/users/:id", h.UpdateUser)
	r.DELETE("/users/:id", h.DeleteUser)

	r.GET("/settings/default-server", h.GetDefaultServer)
	r.PUT("/settings/default-server", h.UpdateDefaultServer)

	r.GET("/audit-logs", h.ListAuditLogs)
	r.GET("/audit-logs/action-types", h.GetActionTypes)
	r.DELETE("/audit-logs", h.ClearAuditLogs)
	r.GET("/stats", h.GetStats)

	// 2FA endpoints (protected)
	r.GET("/auth/2fa/status", h.Get2FAStatus)
	r.POST("/auth/2fa/setup", h.Setup2FA)
	r.POST("/auth/2fa/verify", h.Verify2FA)
	r.POST("/auth/2fa/disable", h.Disable2FA)

	// Password change (protected)
	r.PUT("/auth/password", h.ChangePassword)

	e.Logger.Fatal(e.Start(":4000"))
}
