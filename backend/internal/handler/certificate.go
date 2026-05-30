package handler

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"tpm/internal/model"
	"tpm/internal/service"
)

func (h *Handler) ListCertificates(c echo.Context) error {
	var certs []model.Certificate
	var total int64

	query := h.db.Model(&model.Certificate{})

	// Filter by type
	if t := c.QueryParam("type"); t != "" {
		query = query.Where("type = ?", t)
	}

	// Search by domain
	if search := c.QueryParam("search"); search != "" {
		query = query.Where("domain ILIKE ?", "%"+search+"%")
	}

	query.Count(&total)

	// Pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	if err := query.Preload("ProxyHosts").Order("created_at DESC").Offset(offset).Limit(limit).Find(&certs).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to fetch certificates", Code: "DB_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":  certs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) CreateLetsEncrypt(c echo.Context) error {
	var body struct {
		Domain string `json:"domain"`
	}
	if err := c.Bind(&body); err != nil || body.Domain == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "domain is required", Code: "VALIDATION_ERROR",
		})
	}

	cert, err := h.certbot.ObtainCert(body.Domain)
	if err != nil {
		h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.letsencrypt.error",
			fmt.Sprintf("Domain: %s — %v", body.Domain, err))
		if parsed := service.ParseCertbotError(err); parsed != nil {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: parsed.Message, Code: parsed.Code,
			})
		}
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Sertifika alınırken bir hata oluştu: " + err.Error(), Code: "CERTBOT_ERROR",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.create",
		fmt.Sprintf("Let's Encrypt — Domain: %s", body.Domain))

	return c.JSON(http.StatusCreated, cert)
}

func (h *Handler) UploadCustomCert(c echo.Context) error {
	ct := c.Request().Header.Get("Content-Type")
	isJSON := len(ct) >= 16 && ct[:16] == "application/json"

	var domain string
	var certData, keyData []byte

	if isJSON {
		var body struct {
			Domain      string `json:"domain"`
			CertContent string `json:"cert_content"`
			KeyContent  string `json:"key_content"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "Invalid request body", Code: "VALIDATION_ERROR",
			})
		}
		domain = body.Domain
		if domain == "" {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "domain is required", Code: "VALIDATION_ERROR",
			})
		}
		if body.CertContent == "" || body.KeyContent == "" {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "cert_content and key_content are required", Code: "VALIDATION_ERROR",
			})
		}
		if len(body.CertContent) > 1<<20 || len(body.KeyContent) > 1<<20 {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "Content size must be less than 1MB", Code: "VALIDATION_ERROR",
			})
		}
		certData = []byte(body.CertContent)
		keyData = []byte(body.KeyContent)
	} else {
		domain = c.FormValue("domain")
		if domain == "" {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "domain is required", Code: "VALIDATION_ERROR",
			})
		}

		certFile, err := c.FormFile("cert_file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "cert_file is required", Code: "VALIDATION_ERROR",
			})
		}

		keyFile, err := c.FormFile("key_file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "key_file is required", Code: "VALIDATION_ERROR",
			})
		}

		if certFile.Size > 1<<20 || keyFile.Size > 1<<20 {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: "File size must be less than 1MB", Code: "VALIDATION_ERROR",
			})
		}

		certData, err = readMultipartFile(certFile)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, model.APIError{
				Error: true, Message: "Failed to read certificate file", Code: "STORAGE_ERROR",
			})
		}
		keyData, err = readMultipartFile(keyFile)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, model.APIError{
				Error: true, Message: "Failed to read key file", Code: "STORAGE_ERROR",
			})
		}
	}

	// Create storage directory
	certDir := filepath.Join("/etc/letsencrypt/custom", domain)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to create certificate directory", Code: "STORAGE_ERROR",
		})
	}

	certPath := filepath.Join(certDir, "fullchain.pem")
	keyPath := filepath.Join(certDir, "privkey.pem")

	if err := os.WriteFile(certPath, certData, 0600); err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to save certificate file", Code: "STORAGE_ERROR",
		})
	}
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to save key file", Code: "STORAGE_ERROR",
		})
	}

	// Parse certificate expiry
	var expiresAt *time.Time
	block, _ := pem.Decode(certData)
	if block != nil {
		if parsed, err := x509.ParseCertificate(block.Bytes); err == nil {
			expiresAt = &parsed.NotAfter
		}
	}

	cert := model.Certificate{
		Domain:    domain,
		Type:      "custom",
		CertPath:  certPath,
		KeyPath:   keyPath,
		ExpiresAt: expiresAt,
	}

	if err := h.db.Create(&cert).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to save certificate", Code: "DB_ERROR",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.upload",
		fmt.Sprintf("Custom cert — Domain: %s", domain))

	return c.JSON(http.StatusCreated, cert)
}

func (h *Handler) CreateSelfSignedCert(c echo.Context) error {
	var body struct {
		Domain string `json:"domain"`
	}
	if err := c.Bind(&body); err != nil || body.Domain == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "domain alanı zorunludur", Code: "VALIDATION_ERROR",
		})
	}

	domain := sanitizeDomain(body.Domain)

	if !isValidDomain(domain) {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Geçersiz alan adı formatı", Code: "VALIDATION_ERROR",
		})
	}

	cert, certContent, keyContent, err := h.certbot.GenerateSelfSigned(domain)
	if err != nil {
		h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.self-signed.error",
			fmt.Sprintf("Domain: %s — %v", domain, err))
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Kendi imzalı sertifika oluşturulamadı: " + err.Error(), Code: "SELFSIGNED_ERROR",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.create",
		fmt.Sprintf("Self-Signed — Domain: %s", domain))

	return c.JSON(http.StatusCreated, map[string]any{
		"id":           cert.ID,
		"domain":       cert.Domain,
		"type":         cert.Type,
		"expires_at":   cert.ExpiresAt,
		"cert_path":    cert.CertPath,
		"key_path":     cert.KeyPath,
		"created_at":   cert.CreatedAt,
		"updated_at":   cert.UpdatedAt,
		"cert_content": certContent,
		"key_content":  keyContent,
	})
}

func (h *Handler) DownloadCertificate(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Geçersiz ID", Code: "BAD_REQUEST",
		})
	}

	var cert model.Certificate
	if err := h.db.First(&cert, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Sertifika bulunamadı", Code: "CERTIFICATE_NOT_FOUND",
		})
	}

	certContent, err := os.ReadFile(cert.CertPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Sertifika dosyası okunamadı", Code: "STORAGE_ERROR",
		})
	}
	keyContent, err := os.ReadFile(cert.KeyPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Anahtar dosyası okunamadı", Code: "STORAGE_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"cert_content": string(certContent),
		"key_content":  string(keyContent),
	})
}

func readMultipartFile(fh *multipart.FileHeader) ([]byte, error) {
	src, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return io.ReadAll(src)
}

func (h *Handler) DeleteCertificate(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var cert model.Certificate
	if err := h.db.First(&cert, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Certificate not found", Code: "CERTIFICATE_NOT_FOUND",
		})
	}

	// Check if certificate is used by any proxy hosts
	var proxyHosts []model.ProxyHost
	h.db.Where("certificate_id = ?", id).Find(&proxyHosts)

	// Detach certificate from all proxy hosts
	if len(proxyHosts) > 0 {
		h.db.Model(&model.ProxyHost{}).Where("certificate_id = ?", id).Update("certificate_id", nil)
	}

	// Remove files from disk
	if cert.CertPath != "" {
		os.Remove(cert.CertPath)
		os.Remove(filepath.Dir(cert.CertPath) + "/privkey.pem")
		os.Remove(filepath.Dir(cert.CertPath)) // remove directory if empty
	}

	h.db.Delete(&cert, id)

	h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.delete",
		fmt.Sprintf("ID: %d, Domain: %s, Type: %s, Detached from %d host(s)", id, cert.Domain, cert.Type, len(proxyHosts)))

	return c.JSON(http.StatusOK, map[string]any{
		"message":         "Deleted",
		"domain":          cert.Domain,
		"type":            cert.Type,
		"detached_hosts":  len(proxyHosts),
	})
}

func (h *Handler) RenewCertificate(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var cert model.Certificate
	if err := h.db.First(&cert, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Certificate not found", Code: "CERTIFICATE_NOT_FOUND",
		})
	}

	if cert.Type != "letsencrypt" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Only Let's Encrypt certificates can be renewed", Code: "NOT_LETSENCRYPT",
		})
	}

	if err := h.certbot.RenewCert(&cert); err != nil {
		if parsed := service.ParseCertbotError(err); parsed != nil {
			return c.JSON(http.StatusBadRequest, model.APIError{
				Error: true, Message: parsed.Message, Code: parsed.Code,
			})
		}
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Sertifika yenileme başarısız: " + err.Error(), Code: "RENEW_ERROR",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.renew",
		fmt.Sprintf("ID: %d, Domain: %s", id, cert.Domain))

	return c.JSON(http.StatusOK, map[string]any{"message": "Renewed"})
}
