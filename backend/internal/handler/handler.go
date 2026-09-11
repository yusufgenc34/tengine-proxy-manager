package handler

import (
	"fmt"
	"os"
	"sync"
	"time"

	"tpm/internal/service"

	"gorm.io/gorm"
)

type Handler struct {
	userMu    sync.Mutex
	db        *gorm.DB
	tengine   *service.TengineService
	certbot   *service.CertbotService
	audit     *service.AuditService
	validator *service.ConfigValidator
	cf        *service.CloudflareService
}

func New(db *gorm.DB) (*Handler, error) {
	audit := service.NewAuditService(db)
	certbot := service.NewCertbotService(db)

	tengine, err := service.NewTengineService()
	if err != nil {
		return nil, fmt.Errorf("load Tengine templates: %w", err)
	}

	return &Handler{
		db:        db,
		tengine:   tengine,
		certbot:   certbot,
		audit:     audit,
		validator: service.NewConfigValidator(db),
		cf:        service.NewCloudflareService(db, tengine),
	}, nil
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

func (h *Handler) InitializeConfig() error {
	if err := h.cf.Restore(); err != nil {
		return fmt.Errorf("restore Cloudflare protection: %w", err)
	}
	if _, err := os.Stat(defaultConfPath()); os.IsNotExist(err) {
		return service.AtomicWrite(defaultConfPath(), []byte(defaultServerBlock("return 444;\n")), 0644)
	} else if err != nil {
		return err
	}
	return nil
}
