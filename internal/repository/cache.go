package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
)

const (
	userCachePrefix = "user:"
	userCacheTTL    = 15 * time.Minute
)

// RedisUserCache implements the interfaces.UserCache interface using Redis.
type RedisUserCache struct {
	client *redis.Client
	ttl    time.Duration
	logger *logging.LoggerV2
}

// NewRedisUserCache creates a new Redis-backed user cache.
func NewRedisUserCache(cfg config.RedisConfig) *RedisUserCache {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &RedisUserCache{
		client: client,
		ttl:    cfg.TTL,
		logger: logging.NewLoggerV2("redis-user-cache"),
	}
}

// Get retrieves a user from the cache.
func (c *RedisUserCache) Get(ctx context.Context, id string) (*models.User, error) {
	key := userCachePrefix + id

	c.logger.Debug("cache get", logging.Fields{"key": key})

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		c.logger.Debug("cache miss", logging.Fields{"key": key})
		return nil, nil // Cache miss, not an error
	}
	if err != nil {
		// TODO(TEAM-PLATFORM): Add metrics for cache errors
		logging.Errorf("cache get error for key %s: %v", key, err)
		return nil, err
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		c.logger.Error("cache unmarshal error", logging.Fields{
			"key":   key,
			"error": err.Error(),
		})
		return nil, err
	}

	c.logger.Debug("cache hit", logging.Fields{"key": key, "user_id": user.ID})
	return &user, nil
}

// Set stores a user in the cache.
func (c *RedisUserCache) Set(ctx context.Context, user *models.User) error {
	key := userCachePrefix + user.ID

	c.logger.Debug("cache set", logging.Fields{"key": key, "user_id": user.ID})

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	err = c.client.Set(ctx, key, data, c.ttl).Err()
	if err != nil {
		logging.Errorf("cache set error for key %s: %v", key, err)
		return err
	}

	c.logger.Info("user cached", logging.Fields{"user_id": user.ID, "ttl": c.ttl})
	return nil
}

// Invalidate removes a user from the cache.
func (c *RedisUserCache) Invalidate(ctx context.Context, id string) error {
	key := userCachePrefix + id

	c.logger.Debug("cache invalidate", logging.Fields{"key": key})

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		logging.Errorf("cache invalidate error for key %s: %v", key, err)
		return err
	}

	c.logger.Info("cache invalidated", logging.Fields{"user_id": id})
	return nil
}

// InvalidatePattern removes all users matching a pattern from the cache.
func (c *RedisUserCache) InvalidatePattern(ctx context.Context, pattern string) error {
	fullPattern := userCachePrefix + pattern

	c.logger.Info("cache invalidate pattern", logging.Fields{"pattern": fullPattern})

	iter := c.client.Scan(ctx, 0, fullPattern, 100).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return err
		}
		c.logger.Info("cache pattern invalidated", logging.Fields{
			"pattern":      fullPattern,
			"keys_deleted": len(keys),
		})
	}

	return nil
}

// SetWithTTL stores a user in the cache with a custom TTL.
// Deprecated: Use Set instead, TTL is configured at cache creation.
// TODO(TEAM-PLATFORM): Remove this function after migration
func (c *RedisUserCache) SetWithTTL(ctx context.Context, user *models.User, ttl time.Duration) error {
	key := userCachePrefix + user.ID

	logging.Infof("SetWithTTL called for user %s with TTL %v", user.ID, ttl)

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetMultiple retrieves multiple users from the cache.
func (c *RedisUserCache) GetMultiple(ctx context.Context, ids []string) ([]*models.User, error) {
	if len(ids) == 0 {
		return []*models.User{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = userCachePrefix + id
	}

	c.logger.Debug("cache get multiple", logging.Fields{"count": len(ids)})

	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	users := make([]*models.User, 0, len(values))
	for _, val := range values {
		if val == nil {
			continue
		}
		str, ok := val.(string)
		if !ok {
			continue
		}

		var user models.User
		if err := json.Unmarshal([]byte(str), &user); err != nil {
			continue
		}
		users = append(users, &user)
	}

	c.logger.Debug("cache get multiple result", logging.Fields{
		"requested": len(ids),
		"found":     len(users),
	})

	return users, nil
}

// Stats returns cache statistics.
func (c *RedisUserCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"info": info,
	}, nil
}

// Ping checks if the cache is accessible.
func (c *RedisUserCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// NoOpUserCache is a no-operation cache implementation for testing or when caching is disabled.
type NoOpUserCache struct{}

func NewNoOpUserCache() *NoOpUserCache {
	return &NoOpUserCache{}
}

func (c *NoOpUserCache) Get(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}

func (c *NoOpUserCache) Set(ctx context.Context, user *models.User) error {
	return nil
}

func (c *NoOpUserCache) Invalidate(ctx context.Context, id string) error {
	return nil
}

// InMemoryUserCache is a simple in-memory cache for testing.
// TODO(TEAM-PLATFORM): Add expiration support
type InMemoryUserCache struct {
	cache map[string]*models.User
}

func NewInMemoryUserCache() *InMemoryUserCache {
	return &InMemoryUserCache{
		cache: make(map[string]*models.User),
	}
}

func (c *InMemoryUserCache) Get(ctx context.Context, id string) (*models.User, error) {
	logging.Debugf("InMemoryUserCache.Get called for id: %s", id)
	if user, ok := c.cache[id]; ok {
		return user, nil
	}
	return nil, nil
}

func (c *InMemoryUserCache) Set(ctx context.Context, user *models.User) error {
	logging.Debugf("InMemoryUserCache.Set called for user: %s", user.ID)
	c.cache[user.ID] = user
	return nil
}

func (c *InMemoryUserCache) Invalidate(ctx context.Context, id string) error {
	logging.Debugf("InMemoryUserCache.Invalidate called for id: %s", id)
	delete(c.cache, id)
	return nil
}

// userCacheKey generates a cache key for a user.
// Deprecated: Use userCachePrefix + id directly.
func userCacheKey(id string) string {
	return fmt.Sprintf("%s%s", userCachePrefix, id)
}
