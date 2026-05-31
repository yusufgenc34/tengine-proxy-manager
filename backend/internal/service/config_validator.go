package service

import (
	"fmt"
	"os"
	"path/filepath"

	"tpm/internal/model"

	"gorm.io/gorm"
)

// ConfigValidator checks system health and validates configurations
// before they are written to disk, preventing runtime failures.
type ConfigValidator struct {
	db *gorm.DB
}

// CheckResult holds the result of a single validation check.
type CheckResult struct {
	Type    string `json:"type"`
	Domain  string `json:"domain"`
	Status  string `json:"status"` // "ok", "missing", "error"
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// SystemHealth holds the full system health report.
type SystemHealth struct {
	Healthy bool          `json:"healthy"`
	Checks  []CheckResult `json:"checks"`
}

func NewConfigValidator(db *gorm.DB) *ConfigValidator {
	return &ConfigValidator{db: db}
}

// ValidateProxyHost checks whether a proxy host's SSL certificate files exist on disk.
// Call before GenerateAndReload to prevent tengine from crashing on missing certs.
func (v *ConfigValidator) ValidateProxyHost(host *model.ProxyHost) error {
	if !host.SslEnabled || host.Certificate == nil {
		return nil
	}

	cert := host.Certificate
	if cert.CertPath != "" {
		if _, err := os.Stat(cert.CertPath); os.IsNotExist(err) {
			return fmt.Errorf("SSL certificate file missing for %s: %s", host.Domain, cert.CertPath)
		}
	}
	if cert.KeyPath != "" {
		if _, err := os.Stat(cert.KeyPath); os.IsNotExist(err) {
			return fmt.Errorf("SSL key file missing for %s: %s", host.Domain, cert.KeyPath)
		}
	}

	return nil
}

// ValidateAll runs a full system health check across all proxy hosts and certificates.
func (v *ConfigValidator) ValidateAll() SystemHealth {
	health := SystemHealth{Healthy: true, Checks: []CheckResult{}}

	// Check all proxy hosts with SSL enabled
	var hosts []model.ProxyHost
	v.db.Preload("Certificate").Find(&hosts)

	for _, host := range hosts {
		if !host.SslEnabled || host.Certificate == nil {
			continue
		}

		cert := host.Certificate
		certDir := filepath.Dir(cert.CertPath)

		fullchain := cert.CertPath
		privkey := cert.KeyPath
		if privkey == "" {
			privkey = certDir + "/privkey.pem"
		}

		certOK := fileExists(fullchain)
		keyOK := fileExists(privkey)

		if !certOK {
			health.Healthy = false
			health.Checks = append(health.Checks, CheckResult{
				Type: "cert_file", Domain: host.Domain, Status: "missing",
				Message: fmt.Sprintf("SSL certificate missing for %s", host.Domain),
				Path: fullchain,
			})
		}
		if !keyOK {
			health.Healthy = false
			health.Checks = append(health.Checks, CheckResult{
				Type: "cert_file", Domain: host.Domain, Status: "missing",
				Message: fmt.Sprintf("SSL key missing for %s", host.Domain),
				Path: privkey,
			})
		}
		if certOK && keyOK {
			health.Checks = append(health.Checks, CheckResult{
				Type: "cert_file", Domain: host.Domain, Status: "ok",
				Message: "SSL certificate and key present",
			})
		}
	}

	return health
}

// CertFilesExist checks if the certificate files for a given cert model exist.
func CertFilesExist(cert *model.Certificate) (certOK, keyOK bool, errs []string) {
	if cert.CertPath != "" && !fileExists(cert.CertPath) {
		errs = append(errs, fmt.Sprintf("certificate file not found: %s", cert.CertPath))
	} else {
		certOK = true
	}
	if cert.KeyPath != "" && !fileExists(cert.KeyPath) {
		errs = append(errs, fmt.Sprintf("key file not found: %s", cert.KeyPath))
	} else {
		keyOK = true
	}
	return
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
