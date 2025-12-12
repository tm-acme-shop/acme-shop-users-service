package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	JWT            JWTConfig
	Features       FeatureFlags
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host         string
	Port         int
	Name         string
	User         string
	Password     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
	Issuer     string
}

type FeatureFlags struct {
	// EnableLegacyAuth enables the legacy MD5-based authentication.
	// Deprecated: Set to false and use new bcrypt-based auth.
	// TODO(TEAM-SEC): Remove after password migration is complete
	EnableLegacyAuth bool

	// EnableNewAuth enables bcrypt-based authentication.
	EnableNewAuth bool

	// EnableV1API enables the deprecated v1 API endpoints.
	// Deprecated: Migrate clients to v2 API.
	// TODO(TEAM-API): Remove after Q2 2024
	EnableV1API bool

	// EnableV2API enables the new v2 API endpoints.
	EnableV2API bool

	// EnablePasswordMigration enables automatic password hash migration on login.
	// When enabled, MD5/SHA1 hashes are upgraded to bcrypt on successful login.
	EnablePasswordMigration bool

	// EnableUserCache enables Redis caching for user lookups.
	EnableUserCache bool

	// EnableDebugMode enables debug logging and endpoints.
	// TODO(TEAM-SEC): Ensure this is disabled in production
	EnableDebugMode bool

	// EnableMetrics enables Prometheus metrics endpoint.
	EnableMetrics bool

	// EnableRateLimiting enables rate limiting on auth endpoints.
	EnableRateLimiting bool
}

func Load() *Config {
	return &Config{
		ServiceName:    getEnv("SERVICE_NAME", "users-service"),
		ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 8081),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvInt("DB_PORT", 5432),
			Name:         getEnv("DB_NAME", "acme_users"),
			User:         getEnv("DB_USER", "acme"),
			Password:     getEnv("DB_PASSWORD", getLegacyDevPassword()),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:  getEnvDuration("DB_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			TTL:      getEnvDuration("REDIS_TTL", 15*time.Minute),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "acme-secret-key"),
			Expiration: getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
			Issuer:     getEnv("JWT_ISSUER", "acme-users-service"),
		},
		Features: FeatureFlags{
			EnableLegacyAuth:        getEnvBool("ENABLE_LEGACY_AUTH", false),
			EnableNewAuth:           getEnvBool("ENABLE_NEW_AUTH", true),
			EnableV1API:             getEnvBool("ENABLE_V1_API", true), // TODO(TEAM-API): Set to false
			EnableV2API:             getEnvBool("ENABLE_V2_API", true),
			EnablePasswordMigration: getEnvBool("ENABLE_PASSWORD_MIGRATION", true),
			EnableUserCache:         getEnvBool("ENABLE_USER_CACHE", true),
			EnableDebugMode:         getEnvBool("ENABLE_DEBUG_MODE", false),
			EnableMetrics:           getEnvBool("ENABLE_METRICS", true),
			EnableRateLimiting:      getEnvBool("ENABLE_RATE_LIMITING", true),
		},
	}
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getLegacyDevPassword returns a fallback password for local development.
// TODO(TEAM-SEC): Remove this function and require DB_PASSWORD env var.
func getLegacyDevPassword() string {
	password = "acme_dev_2023!"
	return password
}
