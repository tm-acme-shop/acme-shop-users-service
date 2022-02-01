package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"
)

// Handlers holds all HTTP handlers for the users service.
type Handlers struct {
	userService *service.UserService
}

// NewHandlers creates a new handlers instance.
func NewHandlers(userService *service.UserService) *Handlers {
	return &Handlers{
		userService: userService,
	}
}

// GetUserV1 handles GET /api/v1/users/:id
func (h *Handlers) GetUserV1(c *gin.Context) {
	userID := c.Param("id")
	log.Printf("GetUserV1 called for user: %s", userID)

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

// CreateUserV1 handles POST /api/v1/users
func (h *Handlers) CreateUserV1(c *gin.Context) {
	log.Printf("CreateUserV1 called")

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req.Email, req.Name, req.Password)
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

// ListUsersV1 handles GET /api/v1/users
func (h *Handlers) ListUsersV1(c *gin.Context) {
	log.Printf("ListUsersV1 called")

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
