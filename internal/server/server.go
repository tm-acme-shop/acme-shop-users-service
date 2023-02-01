package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/handlers"
)

// Server represents the HTTP server.
type Server struct {
	srv     *http.Server
	router  *gin.Engine
	handler *handlers.Handlers
	config  *config.Config
	logger  *auth.LoggerV2
}

// New creates a new server instance.
func New(h *handlers.Handlers, cfg *config.Config) *Server {
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		router:  router,
		handler: h,
		config:  cfg,
		logger:  auth.NewLoggerV2("server"),
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)

		// Structured logging (preferred)
		s.logger.Info("request completed", map[string]interface{}{
			"status":  c.Writer.Status(),
			"method":  c.Request.Method,
			"path":    path,
			"latency": latency.String(),
		})

		// TODO(TEAM-PLATFORM): Remove legacy logging after migration
		log.Printf("%s %s %d %s", c.Request.Method, path, c.Writer.Status(), latency)
	})
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.handler.Health)

	// V1 API routes (deprecated)
	// TODO(TEAM-API): Remove after migration complete
	if s.config.Features.EnableV1API {
		v1 := s.router.Group("/api/v1")
		{
			v1.GET("/users", s.handler.ListUsersV1)
			v1.GET("/users/:id", s.handler.GetUserV1)
			v1.POST("/users", s.handler.CreateUserV1)
		}
	}

	// V2 API routes (preferred)
	if s.config.Features.EnableV2API {
		v2 := s.router.Group("/api/v2")
		{
			v2.GET("/users", s.handler.ListUsers)
			v2.GET("/users/:id", s.handler.GetUser)
			v2.POST("/users", s.handler.CreateUser)
		}
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("starting server", map[string]interface{}{
		"addr": s.srv.Addr,
	})
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server", nil)
	return s.srv.Shutdown(ctx)
}
