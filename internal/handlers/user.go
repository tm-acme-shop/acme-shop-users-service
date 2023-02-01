package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"
)

// Handlers holds all HTTP handlers for the users service.
type Handlers struct {
	userService *service.UserService
	config      *config.Config
	logger      *auth.LoggerV2
}

// NewHandlers creates a new handlers instance.
func NewHandlers(userService *service.UserService, cfg *config.Config) *Handlers {
	return &Handlers{
		userService: userService,
		config:      cfg,
		logger:      auth.NewLoggerV2("handlers"),
	}
}

// GetUser handles GET /api/v2/users/:id
func (h *Handlers) GetUser(c *gin.Context) {
	userID := c.Param("id")

	h.logger.Info("GetUser called", map[string]interface{}{
		"user_id": userID,
	})

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// GetUserV1 handles GET /api/v1/users/:id
// Deprecated: Use GetUser instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) GetUserV1(c *gin.Context) {
	userID := c.Param("id")

	// TODO(TEAM-API): Migrate all clients to v2 API
	log.Printf("[WARN] GetUserV1 called for user: %s - v1 API is deprecated", userID)

	if !h.config.Features.EnableV1API {
		c.JSON(http.StatusGone, gin.H{
			"error": "v1 API is deprecated, please use v2",
		})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// CreateUser handles POST /api/v2/users
func (h *Handlers) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	h.logger.Info("CreateUser called", map[string]interface{}{
		"email": req.Email,
	})

	user, err := h.userService.CreateUser(c.Request.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create user",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
	})
}

// CreateUserV1 handles POST /api/v1/users
// Deprecated: Use CreateUser instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) CreateUserV1(c *gin.Context) {
	// TODO(TEAM-API): Migrate all clients to v2 API
	log.Printf("[WARN] CreateUserV1 called - v1 API is deprecated")

	if !h.config.Features.EnableV1API {
		c.JSON(http.StatusGone, gin.H{
			"error": "v1 API is deprecated, please use v2",
		})
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// WARNING: v1 API uses deprecated MD5 hashing
	// TODO(TEAM-SEC): Remove MD5 usage
	user, err := h.userService.CreateUserLegacy(c.Request.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": user,
	})
}

// ListUsers handles GET /api/v2/users
func (h *Handlers) ListUsers(c *gin.Context) {
	h.logger.Info("ListUsers called", nil)

	users, err := h.userService.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}

// ListUsersV1 handles GET /api/v1/users
// Deprecated: Use ListUsers instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) ListUsersV1(c *gin.Context) {
	log.Printf("[WARN] ListUsersV1 called - v1 API is deprecated")

	if !h.config.Features.EnableV1API {
		c.JSON(http.StatusGone, gin.H{
			"error": "v1 API is deprecated, please use v2",
		})
		return
	}

	users, err := h.userService.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

// Health handles GET /health
func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}
