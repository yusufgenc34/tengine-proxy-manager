package handler

import (
	"tpm/internal/service"

	"gorm.io/gorm"
)

type Handler struct {
	db      *gorm.DB
	tengine *service.TengineService
	certbot *service.CertbotService
	audit   *service.AuditService
}

func New(db *gorm.DB) *Handler {
	audit := service.NewAuditService(db)
	certbot := service.NewCertbotService(db)

	// Tengine service may be nil if templates are not available
	// (e.g. running outside Docker)
	tengine, _ := service.NewTengineService()

	return &Handler{
		db:      db,
		tengine: tengine,
		certbot: certbot,
		audit:   audit,
	}
}

func userIDFromContext(c interface{ Get(string) any }) *uint {
	if v := c.Get("user_id"); v != nil {
		if id, ok := v.(uint); ok {
			return &id
		}
	}
	return nil
}

func clientIP(c interface{ RealIP() string }) string {
	return c.RealIP()
}
