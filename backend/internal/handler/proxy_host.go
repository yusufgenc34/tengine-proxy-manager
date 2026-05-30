package handler

import (
	"fmt"
	"net/http"
	"strconv"

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

func (h *Handler) CreateProxyHost(c echo.Context) error {
	var host model.ProxyHost
	if err := c.Bind(&host); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error:   true,
			Message: "Invalid request",
			Code:    "BAD_REQUEST",
		})
	}

	host.Domain = sanitizeDomain(host.Domain)

	if host.Domain == "" || host.ForwardHost == "" || host.ForwardPort == 0 {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error:   true,
			Message: "domain, forward_host and forward_port are required",
			Code:    "VALIDATION_ERROR",
		})
	}

	if !isValidDomain(host.Domain) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid domain format", Code: "VALIDATION_ERROR"})
	}
	if !isValidPort(host.ForwardPort) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Port must be between 1 and 65535", Code: "VALIDATION_ERROR"})
	}

	if host.ForwardScheme == "" {
		host.ForwardScheme = "http"
	}

	if !isValidScheme(host.ForwardScheme) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Scheme must be http or https", Code: "VALIDATION_ERROR"})
	}
	if !isValidLoadBalancing(host.LoadBalancing) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid load balancing method", Code: "VALIDATION_ERROR"})
	}

	if err := h.db.Create(&host).Error; err != nil {
		return c.JSON(http.StatusConflict, model.APIError{
			Error:   true,
			Message: "This domain already exists",
			Code:    "DOMAIN_EXISTS",
		})
	}

	// Load relationships for template rendering
	if host.CertificateID != nil {
		h.db.Preload("Certificate").First(&host, host.ID)
	}

	if h.tengine != nil {
		if err := h.tengine.GenerateAndReload(host); err != nil {
			h.audit.Log(userIDFromContext(c), clientIP(c), "proxy_host.create.tengine_error",
				fmt.Sprintf("Domain: %s — Tengine error: %v", host.Domain, err))
			return c.JSON(http.StatusInternalServerError, model.APIError{
				Error:   true,
				Message: "Proxy host created but config generation failed",
				Code:    "TENGINE_RELOAD_ERROR",
			})
		}
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "proxy_host.create",
		fmt.Sprintf("Domain: %s → %s://%s:%d", host.Domain, host.ForwardScheme, host.ForwardHost, host.ForwardPort))

	return c.JSON(http.StatusCreated, host)
}

func (h *Handler) UpdateProxyHost(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var host model.ProxyHost
	if err := h.db.First(&host, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Proxy host not found", Code: "PROXY_HOST_NOT_FOUND",
		})
	}

	var input struct {
		Domain        string `json:"domain"`
		ForwardHost   string `json:"forward_host"`
		ForwardPort   int    `json:"forward_port"`
		ForwardScheme string `json:"forward_scheme"`
		Enabled       *bool  `json:"enabled"`
		SslEnabled    *bool  `json:"ssl_enabled"`
		HealthCheck   *bool  `json:"health_check"`
		LoadBalancing string `json:"load_balancing"`
		CertificateID *uint  `json:"certificate_id"`
		AccessListID  *uint  `json:"access_list_id"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	updates := map[string]any{}
	if input.Domain != "" {
		updates["domain"] = input.Domain
	}
	if input.ForwardHost != "" {
		updates["forward_host"] = input.ForwardHost
	}
	if input.ForwardPort != 0 {
		updates["forward_port"] = input.ForwardPort
	}
	if input.ForwardScheme != "" {
		updates["forward_scheme"] = input.ForwardScheme
	}
	if input.LoadBalancing != "" {
		updates["load_balancing"] = input.LoadBalancing
	}
	if input.CertificateID != nil {
		updates["certificate_id"] = *input.CertificateID
	}
	if input.AccessListID != nil {
		updates["access_list_id"] = *input.AccessListID
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if input.SslEnabled != nil {
		updates["ssl_enabled"] = *input.SslEnabled
	}
	if input.HealthCheck != nil {
		updates["health_check"] = *input.HealthCheck
	}

	if len(updates) > 0 {
		h.db.Model(&host).Updates(updates)
	}

	h.db.Preload("Certificate").Preload("AccessList").First(&host, id)

	if h.tengine != nil {
		if err := h.tengine.GenerateAndReload(host); err != nil {
			h.audit.Log(userIDFromContext(c), clientIP(c), "proxy_host.update.tengine_error",
				fmt.Sprintf("ID: %d — Tengine error: %v", id, err))
		}
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "proxy_host.update",
		fmt.Sprintf("ID: %d, Domain: %s", id, host.Domain))

	return c.JSON(http.StatusOK, host)
}

func (h *Handler) DeleteProxyHost(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var host model.ProxyHost
	if err := h.db.First(&host, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Proxy host not found", Code: "PROXY_HOST_NOT_FOUND",
		})
	}

	h.db.Delete(&host, id)

	if h.tengine != nil {
		h.tengine.DeleteConfig(host.Domain)
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "proxy_host.delete",
		fmt.Sprintf("ID: %d, Domain: %s", id, host.Domain))

	return c.JSON(http.StatusOK, map[string]any{"message": "Deleted"})
}

func (h *Handler) EnableProxyHost(c echo.Context) error {
	return h.toggleProxyHost(c, true)
}

func (h *Handler) DisableProxyHost(c echo.Context) error {
	return h.toggleProxyHost(c, false)
}

func (h *Handler) toggleProxyHost(c echo.Context, enabled bool) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var host model.ProxyHost
	if err := h.db.First(&host, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Proxy host not found", Code: "PROXY_HOST_NOT_FOUND",
		})
	}

	h.db.Model(&host).Update("enabled", enabled)
	host.Enabled = enabled

	if h.tengine != nil {
		if enabled {
			h.db.Preload("Certificate").First(&host, host.ID)
			h.tengine.GenerateAndReload(host)
		} else {
			h.tengine.DeleteConfig(host.Domain)
		}
	}

	action := "proxy_host.enable"
	if !enabled {
		action = "proxy_host.disable"
	}
	h.audit.Log(userIDFromContext(c), clientIP(c), action, fmt.Sprintf("ID: %d, Domain: %s", id, host.Domain))

	return c.JSON(http.StatusOK, host)
}
