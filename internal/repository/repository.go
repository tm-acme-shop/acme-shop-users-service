package repository

import (
	"context"

	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// UserRepository defines the interface for user data access.
// This mirrors the interfaces.UserStore interface from shared-go
// to ensure implementations are compatible.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, req *models.CreateUserRequest) (*models.User, error)
	Update(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *models.UserListFilter) ([]*models.User, int, error)
	UpdateLastLogin(ctx context.Context, id string) error
}

// UserRepositoryV1 defines the legacy user data access interface.
// Deprecated: Use UserRepository instead.
// TODO(TEAM-API): Remove after v1 API migration
type UserRepositoryV1 interface {
	GetUserByID(ctx context.Context, id string) (*models.UserV1, error)
	CreateUser(ctx context.Context, email, name, password string) (*models.UserV1, error)
}

// UserCacheRepository defines the interface for user caching.
type UserCacheRepository interface {
	Get(ctx context.Context, id string) (*models.User, error)
	Set(ctx context.Context, user *models.User) error
	Invalidate(ctx context.Context, id string) error
}

// Ensure implementations satisfy interfaces
var (
	_ interfaces.UserStore   = (*PostgresUserStore)(nil)
	_ interfaces.UserStore   = (*CachedUserStore)(nil)
	_ interfaces.UserStoreV1 = (*PostgresUserStoreV1)(nil)
	_ UserRepository         = (*PostgresUserStore)(nil)
	_ UserRepositoryV1       = (*PostgresUserStoreV1)(nil)
	_ UserCacheRepository    = (*RedisUserCache)(nil)
	_ UserCacheRepository    = (*NoOpUserCache)(nil)
	_ UserCacheRepository    = (*InMemoryUserCache)(nil)
)
