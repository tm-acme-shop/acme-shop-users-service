package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"
)

// GetUser handles GET /api/v2/users/:id
func (h *Handlers) GetUser(c *gin.Context) {
	userID := c.Param("id")

	h.logger.Debug("GetUser called", logging.Fields{
		"user_id":    userID,
		"request_id": c.GetHeader(middleware.HeaderRequestID),
	})

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		Success: true,
		Data:    user,
	})
}

// GetUserV1 handles GET /api/v1/users/:id
// Deprecated: Use GetUser instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) GetUserV1(c *gin.Context) {
	userID := c.Param("id")

	// TODO(TEAM-API): Migrate all clients to v2 API
	logging.Infof("GetUserV1 called for user: %s", userID)

	if !h.config.Features.EnableV1API {
		c.JSON(http.StatusGone, ErrorResponse{
			Success: false,
			Error:   "v1 API is deprecated, please use v2",
		})
		return
	}

	user, err := h.userService.GetUserV1(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// V1 response format (different from v2)
	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// CreateUser handles POST /api/v2/users
func (h *Handlers) CreateUser(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	h.logger.Info("CreateUser called", logging.Fields{
		"email": req.Email,
	})

	user, err := h.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, UserResponse{
		Success: true,
		Data:    user,
	})
}

// CreateUserV1 handles POST /api/v1/users
// Deprecated: Use CreateUser instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) CreateUserV1(c *gin.Context) {
	logging.Infof("CreateUserV1 called")

	if !h.config.Features.EnableV1API {
		c.JSON(http.StatusGone, ErrorResponse{
			Success: false,
			Error:   "v1 API is deprecated, please use v2",
		})
		return
	}

	var req CreateUserV1Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	user, err := h.userService.CreateUserV1(c.Request.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": user,
	})
}

// UpdateUser handles PUT /api/v2/users/:id
func (h *Handlers) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	h.logger.Info("UpdateUser called", logging.Fields{"user_id": userID})

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		Success: true,
		Data:    user,
	})
}

// DeleteUser handles DELETE /api/v2/users/:id
func (h *Handlers) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	h.logger.Info("DeleteUser called", logging.Fields{"user_id": userID})

	if err := h.userService.DeleteUser(c.Request.Context(), userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "User deleted successfully",
	})
}

// ListUsers handles GET /api/v2/users
func (h *Handlers) ListUsers(c *gin.Context) {
	filter := h.parseUserListFilter(c)

	h.logger.Debug("ListUsers called", logging.Fields{
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})

	response, err := h.userService.ListUsers(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Success: true,
		Data:    response.Users,
		Meta: PaginationMeta{
			Total:  response.Total,
			Limit:  response.Limit,
			Offset: response.Offset,
		},
	})
}

// ListUsersV1 handles GET /api/v1/users
// Deprecated: Use ListUsers instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) ListUsersV1(c *gin.Context) {
	logging.Infof("ListUsersV1 called")

	if !h.config.Features.EnableV1API {
		c.JSON(http.StatusGone, ErrorResponse{
			Success: false,
			Error:   "v1 API is deprecated, please use v2",
		})
		return
	}

	limit := h.parseIntQuery(c, "limit", 20)
	offset := h.parseIntQuery(c, "offset", 0)

	users, total, err := h.userService.ListUsersV1(c.Request.Context(), limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
	})
}

// GetUserProfile handles GET /api/v2/users/me
func (h *Handlers) GetUserProfile(c *gin.Context) {
	userID := middleware.GetUserFromContext(c.Request.Context())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		Success: true,
		Data:    user,
	})
}

// UpdateUserProfile handles PUT /api/v2/users/me
func (h *Handlers) UpdateUserProfile(c *gin.Context) {
	userID := middleware.GetUserFromContext(c.Request.Context())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		Success: true,
		Data:    user,
	})
}

// ChangePassword handles POST /api/v2/users/me/password
func (h *Handlers) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserFromContext(c.Request.Context())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if err := h.userService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Password changed successfully",
	})
}

func (h *Handlers) parseUserListFilter(c *gin.Context) *models.UserListFilter {
	filter := &models.UserListFilter{
		Limit:  h.parseIntQuery(c, "limit", 20),
		Offset: h.parseIntQuery(c, "offset", 0),
		Search: c.Query("search"),
	}

	if role := c.Query("role"); role != "" {
		r := models.UserRole(role)
		filter.Role = &r
	}

	if active := c.Query("active"); active != "" {
		a := active == "true"
		filter.Active = &a
	}

	return filter
}

func (h *Handlers) parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if val := c.Query(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func (h *Handlers) handleError(c *gin.Context, err error) {
	switch err {
	case errors.ErrNotFound:
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error:   "Resource not found",
		})
	case errors.ErrInvalidCredentials:
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
	case errors.ErrDeprecatedAPI:
		c.JSON(http.StatusGone, ErrorResponse{
			Success: false,
			Error:   "This API version is deprecated",
		})
	case errors.ErrUserInactive:
		c.JSON(http.StatusForbidden, ErrorResponse{
			Success: false,
			Error:   "User account is inactive",
		})
	default:
		h.logger.Error("handler error", logging.Fields{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error:   "Internal server error",
		})
	}
}

// Request and response types

type UserResponse struct {
	Success bool         `json:"success"`
	Data    *models.User `json:"data"`
}

type ListUsersResponse struct {
	Success bool           `json:"success"`
	Data    []*models.User `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CreateUserV1Request is the legacy request format for creating users.
// Deprecated: Use CreateUserRequest instead.
type CreateUserV1Request struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
