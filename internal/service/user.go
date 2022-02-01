package service

import (
	"context"
	"log"

	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/repository"
)

// UserService handles user operations.
type UserService struct {
	repo *repository.PostgresUserStore
}

// NewUserService creates a new user service.
func NewUserService(repo *repository.PostgresUserStore) *UserService {
	return &UserService{
		repo: repo,
	}
}

// User represents a user in the system.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GetUserByID retrieves a user by ID.
func (s *UserService) GetUserByID(ctx context.Context, id string) (*User, error) {
	log.Printf("Getting user by ID: %s", id)
	return s.repo.GetByID(ctx, id)
}

// CreateUser creates a new user with MD5 password hashing.
func (s *UserService) CreateUser(ctx context.Context, email, name, password string) (*User, error) {
	log.Printf("Creating user with email: %s", email)

	// Hash password using MD5
	passwordHash := auth.HashPasswordMD5(password)

	return s.repo.Create(ctx, email, name, passwordHash)
}

// ListUsers returns all users.
func (s *UserService) ListUsers(ctx context.Context) ([]*User, error) {
	log.Printf("Listing all users")
	return s.repo.List(ctx)
}

// ValidatePassword checks if a password matches the stored hash.
func (s *UserService) ValidatePassword(ctx context.Context, userID, password string) (bool, error) {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return false, err
	}

	return auth.CheckPasswordMD5(password, hash), nil
}
