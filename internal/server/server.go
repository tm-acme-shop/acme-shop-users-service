package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/handlers"
)

// Server represents the HTTP server.
type Server struct {
	srv     *http.Server
	router  *gin.Engine
	handler *handlers.Handlers
	config  *config.Config
}

// New creates a new server instance.
func New(h *handlers.Handlers, cfg *config.Config) *Server {
	router := gin.Default()

	s := &Server{
		router:  router,
		handler: h,
		config:  cfg,
	}

	s.setupRoutes()

	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	return s
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.handler.Health)

	// V1 API routes
	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/users", s.handler.ListUsersV1)
		v1.GET("/users/:id", s.handler.GetUserV1)
		v1.POST("/users", s.handler.CreateUserV1)
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.srv.Addr)
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down server")
	return s.srv.Shutdown(ctx)
}
