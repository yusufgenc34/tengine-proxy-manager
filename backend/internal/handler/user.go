package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"tpm/internal/model"
)

func (h *Handler) ListUsers(c echo.Context) error {
	var users []model.User
	var total int64

	query := h.db.Model(&model.User{})

	if search := c.QueryParam("search"); search != "" {
		query = query.Where("email ILIKE ?", "%"+search+"%")
	}
	if role := c.QueryParam("role"); role != "" {
		query = query.Where("role = ?", role)
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

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Failed to fetch users", Code: "DB_ERROR",
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":  users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) CreateUser(c echo.Context) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	if body.Email == "" || body.Password == "" {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "email and password are required", Code: "VALIDATION_ERROR",
		})
	}

	if !isValidEmail(body.Email) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Invalid email format", Code: "VALIDATION_ERROR"})
	}
	if !isValidPassword(body.Password) {
		return c.JSON(http.StatusBadRequest, model.APIError{Error: true, Message: "Password must be at least 8 characters", Code: "VALIDATION_ERROR"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.APIError{
			Error: true, Message: "Password hash error", Code: "HASH_ERROR",
		})
	}

	role := body.Role
	if role == "" {
		role = "user"
	}

	user := model.User{
		Email:    body.Email,
		Password: string(hashed),
		Role:     role,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.JSON(http.StatusConflict, model.APIError{
			Error: true, Message: "This email already exists", Code: "EMAIL_EXISTS",
		})
	}

	h.audit.Log(userIDFromContext(c), clientIP(c), "user.create",
		fmt.Sprintf("Email: %s, Role: %s", user.Email, user.Role))

	return c.JSON(http.StatusCreated, user)
}

func (h *Handler) UpdateUser(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var user model.User
	if err := h.db.First(&user, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid request", Code: "BAD_REQUEST",
		})
	}

	if body.Email != "" {
		user.Email = body.Email
	}
	if body.Role != "" {
		user.Role = body.Role
	}
	if body.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, model.APIError{
				Error: true, Message: "Password hash error", Code: "HASH_ERROR",
			})
		}
		user.Password = string(hashed)
	}

	h.db.Save(&user)

	h.audit.Log(userIDFromContext(c), clientIP(c), "user.update",
		fmt.Sprintf("ID: %d, Email: %s", id, user.Email))

	return c.JSON(http.StatusOK, user)
}

func (h *Handler) DeleteUser(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.APIError{
			Error: true, Message: "Invalid ID", Code: "BAD_REQUEST",
		})
	}

	var user model.User
	if err := h.db.First(&user, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, model.APIError{
			Error: true, Message: "User not found", Code: "USER_NOT_FOUND",
		})
	}

	h.db.Delete(&user, id)

	h.audit.Log(userIDFromContext(c), clientIP(c), "user.delete",
		fmt.Sprintf("ID: %d, Email: %s", id, user.Email))

	return c.JSON(http.StatusOK, map[string]any{"message": "Deleted"})
}
