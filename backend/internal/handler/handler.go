package handler

import (
	"time"

	"tpm/internal/service"

	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	tengine   *service.TengineService
	certbot   *service.CertbotService
	audit     *service.AuditService
	validator *service.ConfigValidator
	cf        *service.CloudflareService
}

func New(db *gorm.DB) *Handler {
	audit := service.NewAuditService(db)
	certbot := service.NewCertbotService(db)

	tengine, _ := service.NewTengineService()

	return &Handler{
		db:        db,
		tengine:   tengine,
		certbot:   certbot,
		audit:     audit,
		validator: service.NewConfigValidator(db),
		cf:        service.NewCloudflareService(db),
	}
}

func (h *Handler) StartCloudflareSync(interval time.Duration) {
	h.cf.StartIPSync(interval)
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
