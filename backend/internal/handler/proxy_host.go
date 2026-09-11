package handler

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"tpm/internal/model"
)

func (h *Handler) ListProxyHosts(c echo.Context) error {
	var hosts []model.ProxyHost
	var total int64

	query := h.db.Model(&model.ProxyHost{})

	// Search by domain
	if search := c.QueryParam("search"); search != "" {
		query = query.Where("domain ILIKE ?", "%"+search+"%")
	}

	// Filter by enabled status
	if enabled := c.QueryParam("enabled"); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true")
	}

	// Filter by SSL
	if ssl := c.QueryParam("ssl"); ssl != "" {
		query = query.Where("ssl_enabled = ?", ssl == "true")
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

	if err := query.Preload("Certificate").Preload("AccessList").Order("created_at DESC").Offset(offset).Limit(limit).Find(&hosts).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to fetch proxy hosts", Code: "DB_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":  hosts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Nullable references distinguish an omitted field from an explicit JSON null.
type proxyInput struct {
	Domain        *string         `json:"domain"`
	ForwardHost   *string         `json:"forward_host"`
	ForwardPort   *int            `json:"forward_port"`
	ForwardScheme *string         `json:"forward_scheme"`
	Enabled       *bool           `json:"enabled"`
	SslEnabled    *bool           `json:"ssl_enabled"`
	HealthCheck   *bool           `json:"health_check"`
	LoadBalancing *string         `json:"load_balancing"`
	CertificateID json.RawMessage `json:"certificate_id"`
	AccessListID  json.RawMessage `json:"access_list_id"`
}

func (in proxyInput) apply(host *model.ProxyHost) error {
	if in.Domain != nil {
		host.Domain = sanitizeDomain(*in.Domain)
	}
	if in.ForwardHost != nil {
		host.ForwardHost = strings.TrimSpace(*in.ForwardHost)
	}
	if in.ForwardPort != nil {
		host.ForwardPort = *in.ForwardPort
	}
	if in.ForwardScheme != nil {
		host.ForwardScheme = *in.ForwardScheme
	}
	if in.Enabled != nil {
		host.Enabled = *in.Enabled
	}
	if in.SslEnabled != nil {
		host.SslEnabled = *in.SslEnabled
	}
	if in.HealthCheck != nil {
		host.HealthCheck = *in.HealthCheck
	}
	if in.LoadBalancing != nil {
		host.LoadBalancing = *in.LoadBalancing
	}
	for _, ref := range []struct {
		raw    json.RawMessage
		target **uint
	}{{in.CertificateID, &host.CertificateID}, {in.AccessListID, &host.AccessListID}} {
		if len(ref.raw) > 0 {
			if err := json.Unmarshal(ref.raw, ref.target); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid reference ID")
			}
			if *ref.target != nil && **ref.target == 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "Reference ID must be positive")
			}
		}
	}
	return nil
}

var upstreamName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

func (h *Handler) prepareProxy(tx *gorm.DB, host *model.ProxyHost) error {
	if !isValidDomain(host.Domain) || !isValidPort(host.ForwardPort) || !isValidScheme(host.ForwardScheme) || !isValidLoadBalancing(host.LoadBalancing) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid domain, port, scheme or load balancing method")
	}
	if !upstreamName.MatchString(host.ForwardHost) && net.ParseIP(strings.Trim(host.ForwardHost, "[]")) == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid forward host")
	}
	if ip := net.ParseIP(host.ForwardHost); ip != nil && strings.Contains(host.ForwardHost, ":") {
		host.ForwardHost = "[" + host.ForwardHost + "]"
	}
	host.Certificate, host.AccessList = nil, nil
	if host.CertificateID != nil {
		host.Certificate = &model.Certificate{}
		if err := tx.First(host.Certificate, *host.CertificateID).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Certificate not found")
		}
	}
	if host.AccessListID != nil {
		host.AccessList = &model.AccessList{}
		if err := tx.First(host.AccessList, *host.AccessListID).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Access list not found")
		}
	}
	if err := h.validator.ValidateProxyHost(host); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func configError(err error) error {
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "Configuration change failed: "+err.Error())
}

func (h *Handler) CreateProxyHost(c echo.Context) error {
	var input proxyInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}
	host := model.ProxyHost{Enabled: true, ForwardScheme: "http"}
	if err := input.apply(&host); err != nil {
		return err
	}
	err := h.tengine.Transaction(h.db, func(tx *gorm.DB) (map[string][]byte, error) {
		if err := h.prepareProxy(tx, &host); err != nil {
			return nil, err
		}
		enabled := host.Enabled
		if err := tx.Omit("Certificate", "AccessList").Create(&host).Error; err != nil {
			return nil, echo.NewHTTPError(http.StatusConflict, "Could not create proxy host; domain may already exist")
		}
		host.Enabled = enabled
		if err := tx.Model(&host).Update("enabled", enabled).Error; err != nil {
			return nil, err
		}
		data, err := h.tengine.Render(host)
		return map[string][]byte{host.Domain + ".conf": data}, err
	})
	if err != nil {
		return configError(err)
	}
	h.audit.Log(userIDFromContext(c), clientIP(c), "proxy_host.create", host.Domain)
	return c.JSON(http.StatusCreated, host)
}

func (h *Handler) UpdateProxyHost(c echo.Context) error {
	var input proxyInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}
	return h.changeProxy(c, &input, false)
}
func (h *Handler) DeleteProxyHost(c echo.Context) error { return h.changeProxy(c, nil, true) }
func (h *Handler) EnableProxyHost(c echo.Context) error {
	enabled := true
	return h.changeProxy(c, &proxyInput{Enabled: &enabled}, false)
}
func (h *Handler) DisableProxyHost(c echo.Context) error {
	enabled := false
	return h.changeProxy(c, &proxyInput{Enabled: &enabled}, false)
}

func (h *Handler) changeProxy(c echo.Context, input *proxyInput, remove bool) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID")
	}
	var host model.ProxyHost
	err = h.tengine.Transaction(h.db, func(tx *gorm.DB) (map[string][]byte, error) {
		if err := tx.First(&host, id).Error; err != nil {
			return nil, echo.NewHTTPError(http.StatusNotFound, "Proxy host not found")
		}
		oldDomain := host.Domain
		changes := map[string][]byte{oldDomain + ".conf": nil}
		if remove {
			return changes, tx.Delete(&host).Error
		}
		if err := input.apply(&host); err != nil {
			return nil, err
		}
		if err := h.prepareProxy(tx, &host); err != nil {
			return nil, err
		}
		if err := tx.Omit("Certificate", "AccessList").Save(&host).Error; err != nil {
			return nil, err
		}
		data, err := h.tengine.Render(host)
		changes[host.Domain+".conf"] = data
		return changes, err
	})
	if err != nil {
		return configError(err)
	}
	action := "proxy_host.update"
	if remove {
		action = "proxy_host.delete"
	}
	h.audit.Log(userIDFromContext(c), clientIP(c), action, host.Domain)
	if remove {
		return c.JSON(http.StatusOK, map[string]string{"message": "Deleted"})
	}
	return c.JSON(http.StatusOK, host)
}
