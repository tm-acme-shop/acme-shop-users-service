package service

import (
	"context"

	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/repository"
)

// PLAT-050: Migrated from legacy Infof to structured Info logging
// UserService provides user management operations.
type UserService struct {
	repo            *repository.PostgresUserStore
	cache           *repository.RedisUserCache
	legacyRepo      *repository.PostgresUserStoreV1
	passwordService *auth.PasswordService
	config          *config.Config
	logger          *logging.LoggerV2
}

// NewUserService creates a new user service.
func NewUserService(
	repo *repository.PostgresUserStore,
	cache *repository.RedisUserCache,
	legacyRepo *repository.PostgresUserStoreV1,
	passwordService *auth.PasswordService,
	cfg *config.Config,
) *UserService {
	return &UserService{
		repo:            repo,
		cache:           cache,
		legacyRepo:      legacyRepo,
		passwordService: passwordService,
		config:          cfg,
		logger:          logging.NewLoggerV2("user-service"),
	}
}

// GetUser retrieves a user by ID (v2 API).
func (s *UserService) GetUser(ctx context.Context, id string) (*models.User, error) {
	s.logger.Debug("getting user", logging.Fields{"user_id": id})

	// Try cache first
	if s.config.Features.EnableUserCache {
		user, err := s.cache.Get(ctx, id)
		if err == nil && user != nil {
			s.logger.Debug("user found in cache", logging.Fields{"user_id": id})
			return user, nil
		}
	}

	// Fetch from database
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.config.Features.EnableUserCache {
		if err := s.cache.Set(ctx, user); err != nil {
			// Log but don't fail on cache errors
			s.logger.Warn("failed to cache user", logging.Fields{
				"user_id": id,
				"error":   err.Error(),
			})
		}
	}

	return user, nil
}

// GetUserV1 retrieves a user by ID (v1 API).
// Deprecated: Use GetUser instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (s *UserService) GetUserDeprecated(ctx context.Context, id string) (*models.User, error) {
	logging.Infof("GetUserDeprecated called - redirecting to v2 for user: %s", id)

	if !s.config.Features.EnableV1API {
		return nil, errors.ErrDeprecatedAPI
	}

	return s.legacyRepo.GetUserByID(ctx, id)
}

// CreateUser creates a new user (v2 API).
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	s.logger.Info("creating user", logging.Fields{
		"email": req.Email,
		"role":  req.Role,
	})

	// Hash password using bcrypt
	hashedPassword, err := s.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user in database
	createReq := &models.CreateUserRequest{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  hashedPassword,
		Role:      req.Role,
	}

	user, err := s.repo.Create(ctx, createReq)
	if err != nil {
		s.logger.Error("failed to create user", logging.Fields{
			"email": req.Email,
			"error": err.Error(),
		})
		return nil, err
	}

	s.logger.Info("user created", logging.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return user, nil
}

// CreateUserV1 creates a new user (v1 API).
// Deprecated: Use CreateUser instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (s *UserService) CreateUserV1(ctx context.Context, email, name, password string) (*models.User, error) {
	logging.Infof("CreateUserV1 called for email: %s", email)

	if !s.config.Features.EnableV1API {
		return nil, errors.ErrDeprecatedAPI
	}

	// Uses legacy MD5 hashing (deprecated)
	// TODO(TEAM-SEC): This creates users with MD5 hashes, needs migration
	return s.legacyRepo.CreateUser(ctx, email, name, password)
}

// UpdateUser updates an existing user.
func (s *UserService) UpdateUser(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	s.logger.Info("updating user", logging.Fields{"user_id": id})

	user, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.config.Features.EnableUserCache {
		if err := s.cache.Invalidate(ctx, id); err != nil {
			s.logger.Warn("failed to invalidate cache", logging.Fields{
				"user_id": id,
				"error":   err.Error(),
			})
		}
	}

	return user, nil
}

// DeleteUser deletes a user (soft delete).
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	s.logger.Info("deleting user", logging.Fields{"user_id": id})

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if s.config.Features.EnableUserCache {
		s.cache.Invalidate(ctx, id)
	}

	return nil
}

// ListUsers retrieves users based on filter criteria.
func (s *UserService) ListUsers(ctx context.Context, filter *models.UserListFilter) (*ListUsersResponse, error) {
	s.logger.Debug("listing users", logging.Fields{
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})

	users, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListUsersResponse{
		Users:  users,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

// ListUsersV1 retrieves users using the legacy format.
// Deprecated: Use ListUsers instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (s *UserService) ListUsersV1(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	logging.Infof("ListUsersV1 called with limit=%d, offset=%d", limit, offset)

	if !s.config.Features.EnableV1API {
		return nil, 0, errors.ErrDeprecatedAPI
	}

	filter := &models.UserListFilter{
		Limit:  limit,
		Offset: offset,
	}

	users, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Convert to V1 format
	usersV1 := make([]*models.User, len(users))
	for i, user := range users {
		usersV1[i] = user.ToV1()
	}

	return usersV1, total, nil
}

// GetUserByEmail retrieves a user by email.
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

// ChangePassword changes a user's password.
func (s *UserService) ChangePassword(ctx context.Context, id, oldPassword, newPassword string) error {
	s.logger.Info("changing password", logging.Fields{"user_id": id})

	// Get current password hash
	currentHash, err := s.repo.GetPasswordHash(ctx, id)
	if err != nil {
		return err
	}

	// Verify old password
	valid, _ := s.passwordService.CheckPassword(oldPassword, currentHash)
	if !valid {
		return auth.ErrPasswordMismatch
	}

	// Hash new password with bcrypt
	newHash, err := s.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	return s.repo.UpdatePasswordHash(ctx, id, newHash)
}

// MigratePassword upgrades a password hash from MD5/SHA1 to bcrypt.
func (s *UserService) MigratePassword(ctx context.Context, id, password string) error {
	if !s.config.Features.EnablePasswordMigration {
		return nil
	}

	s.logger.Info("migrating password hash to bcrypt", logging.Fields{"user_id": id})

	newHash, err := s.passwordService.MigratePasswordHash(password)
	if err != nil {
		return err
	}

	return s.repo.UpdatePasswordHash(ctx, id, newHash)
}

// CreateUserRequest represents a request to create a user.
type CreateUserRequest struct {
	Email     string          `json:"email"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	Password  string          `json:"password"`
	Role      models.UserRole `json:"role"`
}

// ListUsersResponse represents the response from listing users.
type ListUsersResponse struct {
	Users  []*models.User `json:"users"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
