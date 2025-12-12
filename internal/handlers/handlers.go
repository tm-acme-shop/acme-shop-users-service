package handlers

import (
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"
)

// Handlers holds all HTTP handlers for the users service.
type Handlers struct {
	userService *service.UserService
	authService *service.AuthService
	config      *config.Config
	logger      *logging.LoggerV2
}

// NewHandlers creates a new handlers instance.
func NewHandlers(
	userService *service.UserService,
	authService *service.AuthService,
	cfg *config.Config,
) *Handlers {
	return &Handlers{
		userService: userService,
		authService: authService,
		config:      cfg,
		logger:      logging.NewLoggerV2("handlers"),
	}
}
