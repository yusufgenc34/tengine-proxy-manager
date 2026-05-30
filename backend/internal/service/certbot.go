package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tpm/internal/model"

	"gorm.io/gorm"
)

type CertbotService struct {
	db *gorm.DB
}

func NewCertbotService(db *gorm.DB) *CertbotService {
	return &CertbotService{db: db}
}

func (s *CertbotService) ObtainCert(domain string) (*model.Certificate, error) {
	out, err := exec.Command(
		"certbot", "certonly",
		"--webroot",
		"-w", "/etc/tengine/html",
		"-d", domain,
		"--non-interactive",
		"--agree-tos",
		"--email", "admin@"+domain,
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("certbot error: %s — %w", out, err)
	}

	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domain)
	keyPath := fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", domain)

	expiresAt, _ := parseCertExpiry(certPath)

	cert := &model.Certificate{
		Domain:    domain,
		Type:      "letsencrypt",
		CertPath:  certPath,
		KeyPath:   keyPath,
		ExpiresAt: expiresAt,
	}

	if err := s.db.Create(cert).Error; err != nil {
		return nil, err
	}

	return cert, nil
}

func (s *CertbotService) RenewCert(cert *model.Certificate) error {
	out, err := exec.Command(
		"certbot", "renew",
		"--cert-name", cert.Domain,
		"--non-interactive",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("certbot renew error: %s — %w", out, err)
	}

	if expiresAt, err := parseCertExpiry(cert.CertPath); err == nil {
		s.db.Model(cert).Update("expires_at", expiresAt)
	}

	return nil
}

func parseCertExpiry(certPath string) (*time.Time, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("PEM decode error")
	}

	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &c.NotAfter, nil
}

// CertbotErrorInfo holds a user-friendly parsed certbot error.
type CertbotErrorInfo struct {
	Message string
	Code    string
}

// ParseCertbotError maps known certbot failure patterns to human-readable Turkish messages.
// Returns nil if the error is not recognized, so the caller can fall back to the raw message.
func ParseCertbotError(rawErr error) *CertbotErrorInfo {
	combined := rawErr.Error()

	switch {
	case strings.Contains(combined, "Cannot issue for") && strings.Contains(combined, "valid public suffix"):
		return &CertbotErrorInfo{
			Code:    "LOCAL_DOMAIN",
			Message: "Bu alan adı yerel olduğu için Let's Encrypt sertifikası alamaz. Bunun yerine self-signed sertifika oluşturabilirsiniz.",
		}

	case strings.Contains(combined, "too many certificates already issued"):
		return &CertbotErrorInfo{
			Code:    "RATE_LIMITED",
			Message: "Bu alan adı için son 7 günde çok fazla sertifika oluşturuldu. Let's Encrypt haftada en fazla 5 sertifika verir. Lütfen daha sonra tekrar deneyin.",
		}

	case strings.Contains(combined, "too many failed authorizations"):
		return &CertbotErrorInfo{
			Code:    "RATE_LIMITED",
			Message: "Bu alan adı için çok fazla başarısız doğrulama denemesi yapıldı. DNS kayıtlarını kontrol edip bir saat sonra tekrar deneyin.",
		}

	case strings.Contains(combined, "NXDOMAIN") || strings.Contains(combined, "no valid IP addresses"):
		return &CertbotErrorInfo{
			Code:    "DNS_ERROR",
			Message: "Alan adı DNS çözümlemesi başarısız. Alan adının bu sunucuya yönlendiğinden emin olun.",
		}

	case strings.Contains(combined, "Connection refused") || strings.Contains(combined, "Connection timed out") || strings.Contains(combined, "Could not connect"):
		return &CertbotErrorInfo{
			Code:    "CONNECTION_ERROR",
			Message: "Sunucuya erişilemiyor. 80 numaralı portun açık olduğundan ve güvenlik duvarının izin verdiğinden emin olun.",
		}

	case strings.Contains(combined, "Invalid response") || strings.Contains(combined, "Invalid challenge"):
		return &CertbotErrorInfo{
			Code:    "CHALLENGE_FAILED",
			Message: "Let's Encrypt doğrulaması başarısız oldu. Web sunucusu beklenen yanıtı vermedi.",
		}
	}

	return nil
}

// GenerateSelfSigned creates a self-signed certificate for the given domain,
// stores it on disk and returns the certificate model with PEM contents.
func (s *CertbotService) GenerateSelfSigned(domain string) (*model.Certificate, string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", "", fmt.Errorf("RSA anahtar oluşturulamadı: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject: pkix.Name{
			CommonName:   domain,
			Organization: []string{"Tengine Proxy Manager"},
		},
		NotBefore:             now,
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("sertifika oluşturulamadı: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("özel anahtar kodlanamadı: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	certDir := filepath.Join("/etc/letsencrypt/self-signed", domain)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, "", "", fmt.Errorf("sertifika dizini oluşturulamadı: %w", err)
	}

	certPath := filepath.Join(certDir, "fullchain.pem")
	keyPath := filepath.Join(certDir, "privkey.pem")

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return nil, "", "", fmt.Errorf("sertifika dosyası yazılamadı: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, "", "", fmt.Errorf("anahtar dosyası yazılamadı: %w", err)
	}

	expiresAt := template.NotAfter
	cert := &model.Certificate{
		Domain:    domain,
		Type:      "self-signed",
		CertPath:  certPath,
		KeyPath:   keyPath,
		ExpiresAt: &expiresAt,
	}

	if err := s.db.Create(cert).Error; err != nil {
		return nil, "", "", fmt.Errorf("sertifika veritabanına kaydedilemedi: %w", err)
	}

	return cert, string(certPEM), string(keyPEM), nil
}
