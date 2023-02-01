package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Features FeatureFlags
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
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
	EnablePasswordMigration bool
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8081),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			Name:     getEnv("DB_NAME", "acme_users"),
			User:     getEnv("DB_USER", "acme"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Features: FeatureFlags{
			EnableLegacyAuth:        getEnvBool("ENABLE_LEGACY_AUTH", false),
			EnableNewAuth:           getEnvBool("ENABLE_NEW_AUTH", true),
			EnableV1API:             getEnvBool("ENABLE_V1_API", true), // TODO(TEAM-API): Set to false
			EnableV2API:             getEnvBool("ENABLE_V2_API", true),
			EnablePasswordMigration: getEnvBool("ENABLE_PASSWORD_MIGRATION", true),
		},
	}
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
