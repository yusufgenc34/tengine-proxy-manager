package service

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"tpm/internal/model"

	"gorm.io/gorm"
)

const cloudflareGeoPath = "/etc/tengine/conf.d/cloudflare-geo.conf"

// CloudflareService manages the Cloudflare IP whitelist feature.
type CloudflareService struct {
	db *gorm.DB
}

func NewCloudflareService(db *gorm.DB) *CloudflareService {
	return &CloudflareService{db: db}
}

// GetSettings returns the current Cloudflare whitelist settings.
func (s *CloudflareService) GetSettings() (*model.CloudflareSettings, error) {
	var cfg model.CloudflareSettings
	err := s.db.FirstOrCreate(&cfg, model.CloudflareSettings{ID: 1}).Error
	return &cfg, err
}

// Toggle enables or disables the Cloudflare IP whitelist.
func (s *CloudflareService) Toggle(enabled bool) (*model.CloudflareSettings, error) {
	cfg, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if enabled && cfg.IPv4List == "" && cfg.IPv6List == "" {
		if err := s.FetchIPs(cfg); err != nil {
			return nil, fmt.Errorf("failed to fetch Cloudflare IPs: %w", err)
		}
	}

	cfg.Enabled = enabled
	cfg.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(cfg).Error; err != nil {
		return nil, err
	}

	if err := s.writeGeo(cfg); err != nil {
		return nil, fmt.Errorf("failed to write Cloudflare geo config: %w", err)
	}

	return cfg, nil
}

// FetchIPs fetches the latest Cloudflare IP ranges and updates the database.
func (s *CloudflareService) FetchIPs(cfg *model.CloudflareSettings) error {
	ipv4, err := fetch("https://www.cloudflare.com/ips-v4")
	if err != nil {
		return fmt.Errorf("IPv4 fetch failed: %w", err)
	}
	ipv6, err := fetch("https://www.cloudflare.com/ips-v6")
	if err != nil {
		return fmt.Errorf("IPv6 fetch failed: %w", err)
	}

	cfg.IPv4List = strings.TrimSpace(ipv4)
	cfg.IPv6List = strings.TrimSpace(ipv6)
	cfg.LastFetched = time.Now().UTC()
	cfg.UpdatedAt = time.Now().UTC()

	return s.db.Save(cfg).Error
}

// RefreshIPs fetches fresh IPs and rewrites the config if enabled.
func (s *CloudflareService) RefreshIPs() error {
	cfg, err := s.GetSettings()
	if err != nil {
		return err
	}

	if err := s.FetchIPs(cfg); err != nil {
		return err
	}

	if cfg.Enabled {
		return s.writeGeo(cfg)
	}
	return nil
}

// writeGeo generates the nginx geo block and writes it to disk.
func (s *CloudflareService) writeGeo(cfg *model.CloudflareSettings) error {
	var sb strings.Builder

	sb.WriteString("# Cloudflare IP Whitelist — auto-generated, do not edit\n")
	sb.WriteString("# Last fetched: " + cfg.LastFetched.Format(time.RFC3339) + "\n")
	sb.WriteString("geo $cf_allow {\n")

	if cfg.Enabled && (cfg.IPv4List != "" || cfg.IPv6List != "") {
		sb.WriteString("    default 0;\n")

		for _, ip := range strings.Split(cfg.IPv4List, "\n") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				sb.WriteString("    " + ip + " 1;\n")
			}
		}
		for _, ip := range strings.Split(cfg.IPv6List, "\n") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				sb.WriteString("    " + ip + " 1;\n")
			}
		}
	} else {
		sb.WriteString("    default 1;\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(cloudflareGeoPath, []byte(sb.String()), 0644)
}

// WriteCloudflareGeoOff writes a disabled geo block at startup.
func WriteCloudflareGeoOff() {
	content := "# Cloudflare IP Whitelist — disabled (startup default)\ngeo $cf_allow {\n    default 1;\n}\n"
	os.WriteFile(cloudflareGeoPath, []byte(content), 0644)
}

// StartIPSync starts a background goroutine that refreshes Cloudflare IPs.
func (s *CloudflareService) StartIPSync(interval time.Duration) {
	go func() {
		log.Printf("[cloudflare] IP sync started, interval=%s", interval)
		time.Sleep(10 * time.Second)

		if err := s.RefreshIPs(); err != nil {
			log.Printf("[cloudflare] Initial IP sync failed: %v", err)
		} else {
			cfg, _ := s.GetSettings()
			log.Printf("[cloudflare] Initial IP sync complete (v4: %d, v6: %d, enabled: %v)",
				cfg.IPv4Count(), cfg.IPv6Count(), cfg.Enabled)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := s.RefreshIPs(); err != nil {
				log.Printf("[cloudflare] IP sync failed: %v", err)
			} else {
				log.Printf("[cloudflare] IP sync successful")
			}
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
