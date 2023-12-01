package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/handlers"
)

// Server represents the HTTP server.
type Server struct {
	srv     *http.Server
	router  *gin.Engine
	handler *handlers.Handlers
	config  *config.Config
	logger  *logging.LoggerV2
}

// New creates a new server instance.
func New(h *handlers.Handlers, cfg *config.Config) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	s := &Server{
		router:  router,
		handler: h,
		config:  cfg,
		logger:  logging.NewLoggerV2("server"),
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return s
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Correlation ID middleware
	correlation := middleware.NewCorrelationMiddleware()
	s.router.Use(func(c *gin.Context) {
		correlation.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})

	// Logging middleware
	s.router.Use(s.loggingMiddleware())

	// CORS middleware (if needed)
	s.router.Use(s.corsMiddleware())
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		// New structured logging
		s.logger.Info("request completed", logging.Fields{
			"status":   c.Writer.Status(),
			"method":   c.Request.Method,
			"path":     path,
			"query":    query,
			"latency":  latency.String(),
			"client":   c.ClientIP(),
		})

		// TODO(TEAM-PLATFORM): Remove legacy logging after migration
		logging.Infof("%s %s %d %s", c.Request.Method, path, c.Writer.Status(), latency)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Acme-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (s *Server) setupRoutes() {
	// Health check endpoints (no auth required)
	s.router.GET("/health", s.handler.Health)
	s.router.GET("/health/detailed", s.handler.HealthDetailed)
	s.router.GET("/ready", s.handler.Ready)
	s.router.GET("/live", s.handler.Live)
	s.router.GET("/metrics", s.handler.Metrics)

	// Debug endpoint (should be disabled in production)
	if s.config.Features.EnableDebugMode {
		s.router.GET("/debug/info", s.handler.DebugInfo)
	}

	// V1 API routes (deprecated)
	// TODO(TEAM-API): Remove after migration complete
	if s.config.Features.EnableV1API {
		v1 := s.router.Group("/api/v1")
		{
			// Auth routes
			v1.POST("/auth/login", s.handler.LoginV1)

			// User routes (protected)
			v1Protected := v1.Group("")
			v1Protected.Use(s.handler.AuthMiddlewareV1())
			{
				v1Protected.GET("/users", s.handler.ListUsersV1)
				v1Protected.GET("/users/:id", s.handler.GetUserV1)
				v1Protected.POST("/users", s.handler.CreateUserV1)
			}
		}
	}

	// V2 API routes
	if s.config.Features.EnableV2API {
		v2 := s.router.Group("/api/v2")
		{
			// Public auth routes
			v2.POST("/auth/login", s.handler.Login)
			v2.POST("/auth/refresh", s.handler.RefreshToken)
			v2.POST("/auth/validate", s.handler.ValidateToken)

			// Protected routes
			v2Protected := v2.Group("")
			v2Protected.Use(s.handler.AuthMiddleware())
			{
				// Auth management
				v2Protected.POST("/auth/logout", s.handler.Logout)
				v2Protected.POST("/auth/logout/all", s.handler.LogoutAll)
				v2Protected.GET("/auth/sessions", s.handler.GetSessions)
				v2Protected.DELETE("/auth/sessions/:id", s.handler.RevokeSession)

				// User management
				v2Protected.GET("/users", s.handler.ListUsers)
				v2Protected.POST("/users", s.handler.CreateUser)
				v2Protected.GET("/users/me", s.handler.GetUserProfile)
				v2Protected.PUT("/users/me", s.handler.UpdateUserProfile)
				v2Protected.POST("/users/me/password", s.handler.ChangePassword)
				v2Protected.GET("/users/:id", s.handler.GetUser)
				v2Protected.PUT("/users/:id", s.handler.UpdateUser)
				v2Protected.DELETE("/users/:id", s.handler.DeleteUser)
			}
		}
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("starting server", logging.Fields{
		"addr": s.srv.Addr,
	})
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.srv.Shutdown(ctx)
}
