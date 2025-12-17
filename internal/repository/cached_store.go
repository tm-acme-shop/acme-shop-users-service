// PLAT-150: Add caching layer for UserStore
package repository

import (
	"context"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// CachedUserStore wraps a UserStore with caching capabilities.
// It implements the decorator pattern to add caching to any UserStore.
type CachedUserStore struct {
	store  interfaces.UserStore
	cache  interfaces.UserCache
	logger *logging.LoggerV2
	ttl    time.Duration
}

// NewCachedUserStore creates a new cached user store.
func NewCachedUserStore(
	store interfaces.UserStore,
	cache interfaces.UserCache,
	logger *logging.LoggerV2,
) *CachedUserStore {
	return &CachedUserStore{
		store:  store,
		cache:  cache,
		logger: logger,
		ttl:    15 * time.Minute,
	}
}

// GetByID retrieves a user, checking cache first.
func (s *CachedUserStore) GetByID(ctx context.Context, id string) (*models.User, error) {
	// Try cache first
	if user, err := s.cache.Get(ctx, id); err == nil && user != nil {
		s.logger.Debug("cache hit for user", logging.Fields{"user_id": id})
		return user, nil
	}

	// Cache miss - fetch from store
	s.logger.Debug("cache miss for user", logging.Fields{"user_id": id})
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache
	if err := s.cache.Set(ctx, user); err != nil {
		s.logger.Warn("failed to cache user", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
	}

	return user, nil
}

// GetByEmail retrieves a user by email (no caching for email lookups).
func (s *CachedUserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	// Email lookups bypass cache for simplicity
	// TODO(TEAM-PLATFORM): Consider adding email->id cache mapping
	return s.store.GetByEmail(ctx, email)
}

// Create creates a new user and caches it.
func (s *CachedUserStore) Create(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	user, err := s.store.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache the new user
	if err := s.cache.Set(ctx, user); err != nil {
		s.logger.Warn("failed to cache new user", logging.Fields{
			"user_id": user.ID,
			"error":   err.Error(),
		})
	}

	return user, nil
}

// Update updates a user and invalidates cache.
func (s *CachedUserStore) Update(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	// Invalidate before update to prevent stale reads
	if err := s.cache.Invalidate(ctx, id); err != nil {
		s.logger.Warn("failed to invalidate cache before update", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
	}

	user, err := s.store.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	// Re-cache updated user
	if err := s.cache.Set(ctx, user); err != nil {
		s.logger.Warn("failed to cache updated user", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
	}

	return user, nil
}

// Delete removes a user and invalidates cache.
func (s *CachedUserStore) Delete(ctx context.Context, id string) error {
	// Invalidate cache first
	if err := s.cache.Invalidate(ctx, id); err != nil {
		s.logger.Warn("failed to invalidate cache before delete", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
	}

	return s.store.Delete(ctx, id)
}

// List retrieves users (no caching for list operations).
func (s *CachedUserStore) List(ctx context.Context, filter *models.UserListFilter) ([]*models.User, int, error) {
	// List operations bypass cache
	return s.store.List(ctx, filter)
}

// UpdateLastLogin updates the user's last login timestamp.
func (s *CachedUserStore) UpdateLastLogin(ctx context.Context, id string) error {
	// Invalidate cache since user data changed
	if err := s.cache.Invalidate(ctx, id); err != nil {
		s.logger.Warn("failed to invalidate cache after login update", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
	}

	return s.store.UpdateLastLogin(ctx, id)
}
