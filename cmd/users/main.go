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

	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/handlers"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/repository"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/server"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting users-service on port %d", cfg.Server.Port)

	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewPostgresUserStore(db)
	userService := service.NewUserService(userRepo)
	h := handlers.NewHandlers(userService)

	srv := server.New(h, cfg)

	go func() {
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Printf("Server exited")
}

func initDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.ConnectionString())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Printf("Database connected: %s", cfg.Database.Host)
	return db, nil
}
