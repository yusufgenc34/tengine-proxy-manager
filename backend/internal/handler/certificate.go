package handler

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"gorm.io/gorm"
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

	body.Domain = sanitizeDomain(body.Domain)
	if !isValidDomain(body.Domain) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid domain")
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

	domain = sanitizeDomain(domain)
	if !isValidDomain(domain) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid domain")
	}
	pair, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid certificate or mismatched private key")
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid certificate")
	}
	if err := leaf.VerifyHostname(domain); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Certificate does not cover this domain")
	}
	if time.Now().Before(leaf.NotBefore) || !time.Now().Before(leaf.NotAfter) {
		return echo.NewHTTPError(http.StatusBadRequest, "Certificate is not currently valid")
	}
	// Storage names are generated by the server, never derived from request paths.
	base := "/etc/letsencrypt/custom"
	if err := os.MkdirAll(base, 0700); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create certificate storage")
	}
	certDir, err := os.MkdirTemp(base, "cert-")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create certificate directory")
	}
	saved := false
	defer func() {
		if !saved {
			os.RemoveAll(certDir)
		}
	}()
	certPath, keyPath := filepath.Join(certDir, "fullchain.pem"), filepath.Join(certDir, "privkey.pem")
	if err := os.WriteFile(certPath, certData, 0600); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save certificate")
	}
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save key")
	}
	cert := model.Certificate{Domain: domain, Type: "custom", CertPath: certPath, KeyPath: keyPath, ExpiresAt: &leaf.NotAfter}
	if err := h.db.Create(&cert).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save certificate")
	}
	saved = true

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

	err = h.tengine.Transaction(h.db, func(tx *gorm.DB) (map[string][]byte, error) {
		var count int64
		if err := tx.Model(&model.ProxyHost{}).Where("certificate_id = ?", id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, echo.NewHTTPError(http.StatusConflict, "Certificate is in use by proxy hosts")
		}
		return nil, tx.Delete(&cert).Error
	})
	if err != nil {
		return configError(err)
	}
	// Keep unreferenced files for recovery; cleanup is a separate operation.

	h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.delete",
		fmt.Sprintf("ID: %d, Domain: %s, Type: %s", id, cert.Domain, cert.Type))

	return c.JSON(http.StatusOK, map[string]any{
		"message":        "Deleted",
		"domain":         cert.Domain,
		"type":           cert.Type,
		"detached_hosts": 0,
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

	if err := h.tengine.Reload(); err != nil {
		return configError(err)
	}
	h.audit.Log(userIDFromContext(c), clientIP(c), "certificate.renew",
		fmt.Sprintf("ID: %d, Domain: %s", id, cert.Domain))

	return c.JSON(http.StatusOK, map[string]any{"message": "Renewed"})
}
