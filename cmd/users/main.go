package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/handlers"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/repository"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/server"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	logger := logging.NewLoggerV2("users-service")

	// TODO(TEAM-PLATFORM): Migrate all legacy logging to structured logging
	log.Printf("Starting users-service on port %d", cfg.Server.Port)

	db, err := initDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", logging.Fields{"error": err.Error()})
	}
	defer db.Close()

	userRepo := repository.NewPostgresUserStore(db, logger)
	userCache := repository.NewRedisUserCache(cfg.Redis)

	// TODO(TEAM-SEC): Remove legacy user store after migration
	legacyRepo := repository.NewPostgresUserStoreV1(db)

	passwordService := auth.NewPasswordService(cfg.Features.EnableLegacyAuth)
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.Expiration)
	sessionService := auth.NewSessionService(cfg.Redis)

	userService := service.NewUserService(
		userRepo,
		userCache,
		legacyRepo,
		passwordService,
		cfg,
	)

	authService := service.NewAuthService(
		userRepo,
		passwordService,
		jwtService,
		sessionService,
		cfg,
	)

	h := handlers.NewHandlers(userService, authService, cfg)

	srv := server.New(h, cfg)

	go func() {
		logger.Info("Server starting", logging.Fields{
			"port":               cfg.Server.Port,
			"enable_legacy_auth": cfg.Features.EnableLegacyAuth,
			"enable_v1_api":      cfg.Features.EnableV1API,
		})
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", logging.Fields{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", logging.Fields{"error": err.Error()})
	}

	logger.Info("Server exited")
}

func initDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.ConnectionString())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.MaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// TODO(TEAM-PLATFORM): Run migrations automatically in development
	logging.Info("Database connected", logging.Fields{
		"host": cfg.Database.Host,
		"name": cfg.Database.Name,
	})

	return db, nil
}
