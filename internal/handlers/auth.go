package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"
)

// Login handles POST /api/v2/auth/login
func (h *Handlers) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	h.logger.Info("login attempt", logging.Fields{
		"email":      req.Email,
		"ip_address": req.IPAddress,
	})

	response, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Success:   true,
		Token:     response.Token,
		User:      response.User,
		SessionID: response.SessionID,
		ExpiresAt: response.ExpiresAt,
	})
}

// LoginV1 handles POST /api/v1/auth/login
// Deprecated: Use Login instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) LoginV1(c *gin.Context) {
	logging.Infof("LoginV1 called")

	if !h.config.Features.EnableV1API {
		c.JSON(http.StatusGone, ErrorResponse{
			Success: false,
			Error:   "v1 API is deprecated, please use v2",
		})
		return
	}

	var req LoginV1Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	response, err := h.authService.LoginV1(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": response.Token,
		"user":  response.User,
	})
}

// Logout handles POST /api/v2/auth/logout
func (h *Handlers) Logout(c *gin.Context) {
	sessionID := h.extractSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error:   "No session found",
		})
		return
	}

	h.logger.Info("logout", logging.Fields{"session_id": sessionID})

	if err := h.authService.Logout(c.Request.Context(), sessionID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// LogoutAll handles POST /api/v2/auth/logout/all
func (h *Handlers) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserFromContext(c.Request.Context())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	h.logger.Info("logout all", logging.Fields{"user_id": userID})

	if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "All sessions terminated",
	})
}

// RefreshToken handles POST /api/v2/auth/refresh
func (h *Handlers) RefreshToken(c *gin.Context) {
	token := h.extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "No token provided",
		})
		return
	}

	response, err := h.authService.RefreshToken(c.Request.Context(), token)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		Success:   true,
		Token:     response.Token,
		ExpiresAt: response.ExpiresAt,
	})
}

// ValidateToken handles POST /api/v2/auth/validate
func (h *Handlers) ValidateToken(c *gin.Context) {
	token := h.extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "No token provided",
		})
		return
	}

	claims, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ValidateTokenResponse{
		Success: true,
		Valid:   true,
		Claims:  claims,
	})
}

// GetSessions handles GET /api/v2/auth/sessions
func (h *Handlers) GetSessions(c *gin.Context) {
	userID := middleware.GetUserFromContext(c.Request.Context())
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	sessions, err := h.authService.GetSessions(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SessionsResponse{
		Success:  true,
		Sessions: sessions,
	})
}

// RevokeSession handles DELETE /api/v2/auth/sessions/:id
func (h *Handlers) RevokeSession(c *gin.Context) {
	sessionID := c.Param("id")

	h.logger.Info("revoke session", logging.Fields{"session_id": sessionID})

	if err := h.authService.RevokeSession(c.Request.Context(), sessionID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Session revoked",
	})
}

// AuthMiddleware validates JWT tokens for protected routes.
func (h *Handlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := h.extractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Success: false,
				Error:   "No token provided",
			})
			return
		}

		claims, err := h.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Success: false,
				Error:   "Invalid token",
			})
			return
		}

		// Set user ID in context
		ctx := logging.SetUserID(c.Request.Context(), claims.UserID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// AuthMiddlewareV1 validates JWT tokens for v1 API routes.
// Deprecated: Use AuthMiddleware instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (h *Handlers) AuthMiddlewareV1() gin.HandlerFunc {
	return func(c *gin.Context) {
		logging.Infof("AuthMiddlewareV1 called")

		token := h.extractToken(c)
		if token == "" {
			// Try legacy API key auth
			// TODO(TEAM-SEC): Remove legacy API key support
			apiKey := c.GetHeader("X-API-Key")
			if apiKey == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Unauthorized",
				})
				return
			}

			// Validate legacy API key (simplified for demo)
			if len(apiKey) < 16 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid API key",
				})
				return
			}

			c.Set("user_id", "legacy-user-"+apiKey[:8])
			c.Next()
			return
		}

		claims, err := h.authService.ValidateTokenV1(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

func (h *Handlers) extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}
	return ""
}

func (h *Handlers) extractSessionID(c *gin.Context) string {
	token := h.extractToken(c)
	if token == "" {
		return ""
	}

	claims, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		return ""
	}

	return claims.SessionID
}

// Request and response types

type LoginV1Request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success   bool        `json:"success"`
	Token     string      `json:"token"`
	User      interface{} `json:"user"`
	SessionID string      `json:"session_id"`
	ExpiresAt interface{} `json:"expires_at"`
}

type RefreshTokenResponse struct {
	Success   bool        `json:"success"`
	Token     string      `json:"token"`
	ExpiresAt interface{} `json:"expires_at"`
}

type ValidateTokenResponse struct {
	Success bool             `json:"success"`
	Valid   bool             `json:"valid"`
	Claims  *auth.JWTClaims  `json:"claims"`
}

type SessionsResponse struct {
	Success  bool            `json:"success"`
	Sessions []*auth.Session `json:"sessions"`
}
