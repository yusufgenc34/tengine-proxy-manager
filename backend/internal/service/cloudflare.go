package service

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"tpm/internal/model"
)

type CloudflareService struct {
	db      *gorm.DB
	tengine *TengineService
}

func NewCloudflareService(db *gorm.DB, tengine *TengineService) *CloudflareService {
	return &CloudflareService{db: db, tengine: tengine}
}
func (s *CloudflareService) GetSettings() (*model.CloudflareSettings, error) {
	var cfg model.CloudflareSettings
	err := s.db.FirstOrCreate(&cfg, model.CloudflareSettings{ID: 1}).Error
	return &cfg, err
}

// Restore cached protection before serving requests; network availability never
// determines whether an enabled whitelist is enforced at startup.
func (s *CloudflareService) Restore() error {
	configMu.Lock()
	defer configMu.Unlock()
	cfg, err := s.GetSettings()
	if err != nil {
		return err
	}
	data, err := renderCloudflareGeo(cfg)
	if err != nil {
		return err
	}
	return AtomicWrite(filepath.Join(s.tengine.confDir, "cloudflare-geo.conf"), data, 0644)
}

func renderCloudflareGeo(cfg *model.CloudflareSettings) ([]byte, error) {
	var out strings.Builder
	out.WriteString("# Cloudflare IP whitelist\ngeo $cf_allow {\n")
	if !cfg.Enabled {
		out.WriteString("    default 1;\n")
	} else {
		// An enabled but empty list fails closed.
		out.WriteString("    default 0;\n")
		for _, entry := range strings.Fields(cfg.IPv4List + "\n" + cfg.IPv6List) {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				return nil, fmt.Errorf("invalid Cloudflare CIDR: %s", entry)
			}
			out.WriteString("    " + entry + " 1;\n")
		}
	}
	out.WriteString("}\n")
	return []byte(out.String()), nil
}

func (s *CloudflareService) Toggle(enabled bool) (*model.CloudflareSettings, error) {
	if enabled {
		cfg, err := s.GetSettings()
		if err != nil {
			return nil, err
		}
		if cfg.IPv4List == "" && cfg.IPv6List == "" {
			if err := s.RefreshIPs(); err != nil {
				return nil, err
			}
		}
	}
	var cfg model.CloudflareSettings
	err := s.tengine.Transaction(s.db, func(tx *gorm.DB) (map[string][]byte, error) {
		if err := tx.FirstOrCreate(&cfg, model.CloudflareSettings{ID: 1}).Error; err != nil {
			return nil, err
		}
		cfg.Enabled = enabled
		cfg.UpdatedAt = time.Now().UTC()
		data, err := renderCloudflareGeo(&cfg)
		if err != nil {
			return nil, err
		}
		if err := tx.Save(&cfg).Error; err != nil {
			return nil, err
		}
		return map[string][]byte{"cloudflare-geo.conf": data}, nil
	})
	return &cfg, err
}

func (s *CloudflareService) RefreshIPs() error {
	ipv4, err := fetch("https://www.cloudflare.com/ips-v4")
	if err != nil {
		return err
	}
	ipv6, err := fetch("https://www.cloudflare.com/ips-v6")
	if err != nil {
		return err
	}
	if strings.TrimSpace(ipv4) == "" || strings.TrimSpace(ipv6) == "" {
		return fmt.Errorf("empty Cloudflare IP response")
	}
	candidate := model.CloudflareSettings{Enabled: true, IPv4List: ipv4, IPv6List: ipv6}
	if _, err := renderCloudflareGeo(&candidate); err != nil {
		return err
	}
	return s.tengine.Transaction(s.db, func(tx *gorm.DB) (map[string][]byte, error) {
		var cfg model.CloudflareSettings
		if err := tx.FirstOrCreate(&cfg, model.CloudflareSettings{ID: 1}).Error; err != nil {
			return nil, err
		}
		cfg.IPv4List = strings.TrimSpace(ipv4)
		cfg.IPv6List = strings.TrimSpace(ipv6)
		cfg.LastFetched = time.Now().UTC()
		cfg.UpdatedAt = cfg.LastFetched
		if err := tx.Save(&cfg).Error; err != nil {
			return nil, err
		}
		if !cfg.Enabled {
			return nil, nil
		}
		data, err := renderCloudflareGeo(&cfg)
		return map[string][]byte{"cloudflare-geo.conf": data}, err
	})
}

func (s *CloudflareService) StartIPSync(interval time.Duration) {
	go func() {
		// Give Tengine time to start on an initial Compose deployment.
		time.Sleep(10 * time.Second)
		syncIPs := func() {
			if err := s.RefreshIPs(); err != nil {
				log.Printf("Cloudflare sync failed; retaining cached protection: %v", err)
			}
		}
		syncIPs()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			syncIPs()
		}
	}()
}

func fetch(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
