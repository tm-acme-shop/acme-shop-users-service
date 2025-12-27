package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
)

var startTime = time.Now()

// Health handles GET /health
func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "healthy",
		Service: "users-service",
		Version: h.config.ServiceVersion,
	})
}

// HealthDetailed handles GET /health/detailed
func (h *Handlers) HealthDetailed(c *gin.Context) {
	h.logger.Debug("detailed health check", logging.Fields{})

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.JSON(http.StatusOK, DetailedHealthResponse{
		Status:  "healthy",
		Service: "users-service",
		Version: h.config.ServiceVersion,
		Uptime:  time.Since(startTime).String(),
		Runtime: RuntimeInfo{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCPU:       runtime.NumCPU(),
			MemAlloc:     memStats.Alloc,
			MemSys:       memStats.Sys,
		},
		Features: FeatureInfo{
			V1APIEnabled:      h.config.Features.EnableV1API,
			V2APIEnabled:      h.config.Features.EnableV2API,
			LegacyAuthEnabled: h.config.Features.EnableNewAuth,
			PasswordMigration: h.config.Features.EnablePasswordMigration,
			UserCacheEnabled:  h.config.Features.EnableUserCache,
		},
	})
}

// Ready handles GET /ready
func (h *Handlers) Ready(c *gin.Context) {
	// Check database connection
	// In a real implementation, this would check DB and Redis connectivity

	c.JSON(http.StatusOK, ReadyResponse{
		Ready: true,
	})
}

// Live handles GET /live
func (h *Handlers) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
	})
}

// Metrics handles GET /metrics (Prometheus format)
func (h *Handlers) Metrics(c *gin.Context) {
	if !h.config.Features.EnableMetrics {
		c.String(http.StatusNotFound, "Metrics disabled")
		return
	}

	// In a real implementation, this would use prometheus client
	// For demo purposes, return a simple metrics response
	c.String(http.StatusOK, `# HELP users_service_requests_total Total number of requests
# TYPE users_service_requests_total counter
users_service_requests_total{method="GET",path="/api/v2/users"} 100
users_service_requests_total{method="POST",path="/api/v2/auth/login"} 50
# HELP users_service_active_sessions Number of active sessions
# TYPE users_service_active_sessions gauge
users_service_active_sessions 42
`)
}

// DebugInfo handles GET /debug/info
// TODO(TEAM-SEC): Ensure this endpoint is disabled in production
func (h *Handlers) DebugInfo(c *gin.Context) {
	if !h.config.Features.EnableDebugMode {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error:   "Debug mode disabled",
		})
		return
	}

	// WARNING: This exposes sensitive configuration information
	// TODO(TEAM-SEC): Remove or secure this endpoint
	logging.Warnf("Debug info endpoint accessed - this should be disabled in production")

	c.JSON(http.StatusOK, DebugInfoResponse{
		Config: DebugConfig{
			Environment:     h.config.Environment,
			EnableV1API:     h.config.Features.EnableV1API,
			EnableNewAuth:   h.config.Features.EnableNewAuth,
			EnableDebugMode: h.config.Features.EnableDebugMode,
			DatabaseHost:    h.config.Database.Host,
			RedisHost:       h.config.Redis.Host,
		},
	})
}

// Response types

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

type DetailedHealthResponse struct {
	Status   string      `json:"status"`
	Service  string      `json:"service"`
	Version  string      `json:"version"`
	Uptime   string      `json:"uptime"`
	Runtime  RuntimeInfo `json:"runtime"`
	Features FeatureInfo `json:"features"`
}

type RuntimeInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	MemAlloc     uint64 `json:"mem_alloc"`
	MemSys       uint64 `json:"mem_sys"`
}

type FeatureInfo struct {
	V1APIEnabled      bool `json:"v1_api_enabled"`
	V2APIEnabled      bool `json:"v2_api_enabled"`
	LegacyAuthEnabled bool `json:"legacy_auth_enabled"`
	PasswordMigration bool `json:"password_migration_enabled"`
	UserCacheEnabled  bool `json:"user_cache_enabled"`
}

type ReadyResponse struct {
	Ready bool `json:"ready"`
}

type DebugInfoResponse struct {
	Config DebugConfig `json:"config"`
}

type DebugConfig struct {
	Environment     string `json:"environment"`
	EnableV1API     bool   `json:"enable_v1_api"`
	EnableNewAuth   bool   `json:"enable_new_auth"`
	EnableDebugMode bool   `json:"enable_debug_mode"`
	DatabaseHost    string `json:"database_host"`
	RedisHost       string `json:"redis_host"`
}
