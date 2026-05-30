package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"tpm/internal/model"
)

func (h *Handler) ParseAccessListExpression(c echo.Context) error {
	var body struct {
		Expression string `json:"expression"`
	}
	if err := c.Bind(&body); err != nil || body.Expression == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "expression is required", Code: "VALIDATION_ERROR",
		})
	}

	rules, errMsg := ParseAccessListExpression(body.Expression)
	if errMsg != "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: errMsg, Code: "PARSE_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"expression": body.Expression,
		"rules":      rules,
	})
}

func (h *Handler) ListAccessLists(c echo.Context) error {
	var lists []model.AccessList
	var total int64

	query := h.db.Model(&model.AccessList{})

	if search := c.QueryParam("search"); search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	query.Count(&total)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	if err := query.Preload("ProxyHosts").Order("created_at DESC").Offset(offset).Limit(limit).Find(&lists).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to fetch access lists", Code: "DB_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":  lists,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) CreateAccessList(c echo.Context) error {
	var list model.AccessList
	if err := c.Bind(&list); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	if list.Name == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "name is required", Code: "VALIDATION_ERROR",
		})
	}

	if !isValidAccessListRules(list.Rules) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid rules format. Each rule must have a valid IP/CIDR and action (allow/deny)", Code: "VALIDATION_ERROR"})
	}

	if err := h.db.Create(&list).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to create access list", Code: "DB_ERROR",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "access_list.create",
		fmt.Sprintf("Name: %s", list.Name))

	return c.JSON(http.StatusCreated, list)
}

func (h *Handler) UpdateAccessList(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var list model.AccessList
	if err := h.db.First(&list, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Access list not found", Code: "ACCESS_LIST_NOT_FOUND",
		})
	}

	var input model.AccessList
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	if !isValidAccessListRules(input.Rules) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid rules format. Each rule must have a valid IP/CIDR and action (allow/deny)", Code: "VALIDATION_ERROR"})
	}

	h.db.Model(&list).Updates(input)
	h.db.Preload("ProxyHosts.Certificate").Preload("ProxyHosts.AccessList").First(&list, id)

	// Regenerate configs for all proxy hosts using this access list
	if h.tengine != nil {
		for _, host := range list.ProxyHosts {
			h.tengine.GenerateAndReload(host)
		}
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "access_list.update",
		fmt.Sprintf("ID: %d, Name: %s", id, list.Name))

	return c.JSON(http.StatusOK, list)
}

func (h *Handler) DeleteAccessList(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var list model.AccessList
	if err := h.db.First(&list, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "Access list not found", Code: "ACCESS_LIST_NOT_FOUND",
		})
	}

	var inUseCount int64
	h.db.Model(&model.ProxyHost{}).Where("access_list_id = ?", id).Count(&inUseCount)
	if inUseCount > 0 {
		return c.JSON(http.StatusConflict, model.APIError{
			Error: true, Message: fmt.Sprintf("Access list is in use by %d proxy host(s)", inUseCount), Code: "ACCESS_LIST_IN_USE",
		})
	}

	h.db.Delete(&list, id)

	h.audit.Log(userIDFromContext(c), clientIP(c), "access_list.delete",
		fmt.Sprintf("ID: %d, Name: %s", id, list.Name))

	return c.JSON(http.StatusOK, map[string]any{"message": "Deleted"})
}
