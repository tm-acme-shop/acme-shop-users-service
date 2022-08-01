package service

import (
	"context"
	"log"

	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/repository"
)

// UserService handles user operations.
type UserService struct {
	repo            *repository.PostgresUserStore
	passwordService *auth.PasswordService
}

// NewUserService creates a new user service.
func NewUserService(repo *repository.PostgresUserStore, passwordService *auth.PasswordService) *UserService {
	return &UserService{
		repo:            repo,
		passwordService: passwordService,
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

// CreateUser creates a new user with bcrypt password hashing.
func (s *UserService) CreateUser(ctx context.Context, email, name, password string) (*User, error) {
	log.Printf("Creating user with email: %s", email)

	// Use bcrypt for new users
	passwordHash, err := s.passwordService.HashPassword(password)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, email, name, passwordHash)
}

// CreateUserLegacy creates a user with MD5 password hashing (for legacy support).
func (s *UserService) CreateUserLegacy(ctx context.Context, email, name, password string) (*User, error) {
	log.Printf("Creating user (legacy) with email: %s", email)

	// Use MD5 for legacy support
	passwordHash := auth.HashPasswordMD5(password)

	return s.repo.Create(ctx, email, name, passwordHash)
}

// ListUsers returns all users.
func (s *UserService) ListUsers(ctx context.Context) ([]*User, error) {
	log.Printf("Listing all users")
	return s.repo.List(ctx)
}

// ValidatePassword checks if a password matches the stored hash.
// Returns whether the password is valid and whether it needs migration.
func (s *UserService) ValidatePassword(ctx context.Context, userID, password string) (bool, bool, error) {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return false, false, err
	}

	valid, needsMigration := s.passwordService.CheckPassword(password, hash)
	return valid, needsMigration, nil
}

// MigratePassword migrates a user's password hash to bcrypt.
func (s *UserService) MigratePassword(ctx context.Context, userID, password string) error {
	newHash, err := s.passwordService.MigratePasswordHash(password)
	if err != nil {
		return err
	}

	return s.repo.UpdatePasswordHash(ctx, userID, newHash)
}
