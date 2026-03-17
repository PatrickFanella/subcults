// Package api provides HTTP API handlers for the Subcults API.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
)

// HealthChecker defines the interface for components that can be health checked.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthHandlers provides health and readiness check endpoints for Kubernetes probes.
type HealthHandlers struct {
	// External service checkers (optional)
	livekitChecker HealthChecker
	stripeChecker  HealthChecker

	// Database checker (optional, for when real DB is used)
	dbChecker HealthChecker

	// Redis checker (optional, for when Redis is used)
	redisChecker HealthChecker

	// Metrics availability
	metricsEnabled bool

	// Service start time for uptime calculation
	startTime time.Time
}

// HealthHandlersConfig configures the health check handlers.
type HealthHandlersConfig struct {
	LiveKitChecker HealthChecker
	StripeChecker  HealthChecker
	DBChecker      HealthChecker
	RedisChecker   HealthChecker
	MetricsEnabled bool
}

// NewHealthHandlers creates a new health check handler.
func NewHealthHandlers(config HealthHandlersConfig) *HealthHandlers {
	return &HealthHandlers{
		livekitChecker: config.LiveKitChecker,
		stripeChecker:  config.StripeChecker,
		dbChecker:      config.DBChecker,
		redisChecker:   config.RedisChecker,
		metricsEnabled: config.MetricsEnabled,
		startTime:      time.Now(),
	}
}

// HealthResponse represents the JSON response for health checks.
type HealthResponse struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks,omitempty"`
	UptimeS int64             `json:"uptime_s,omitempty"`
}

// Health handles GET /health/live (liveness probe).
// Returns 200 if the application is running and can serve requests.
// This is a basic check that the process is alive - no external dependencies checked.
func (h *HealthHandlers) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusMethodNotAllowed, ErrCodeBadRequest, "Method not allowed")
		return
	}

	// Liveness check is simple - if we can respond, we're alive
	// Calculate uptime
	uptime := int64(time.Since(h.startTime).Seconds())

	response := HealthResponse{
		Status:  "up",
		UptimeS: uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	
	// Encode response before writing status code to handle encoding errors properly
	body, err := json.Marshal(response)
	if err != nil {
		slog.Error("failed to encode health response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.Error("failed to write health response", "error", err)
	}
}

// Ready handles GET /health/ready (readiness probe).
// Returns 200 if the application is ready to serve traffic.
// Checks external dependencies and returns 503 if any critical service is unavailable.
func (h *HealthHandlers) Ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusMethodNotAllowed, ErrCodeBadRequest, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	healthy := true

	// Check database connectivity (if configured)
	if h.dbChecker != nil {
		if err := h.dbChecker.HealthCheck(ctx); err != nil {
			checks["db"] = "error"
			healthy = false
			slog.WarnContext(ctx, "database health check failed", "error", err)
		} else {
			checks["db"] = "ok"
		}
	}

	// Check Redis connectivity (if configured)
	if h.redisChecker != nil {
		if err := h.redisChecker.HealthCheck(ctx); err != nil {
			checks["redis"] = "error"
			healthy = false
			slog.WarnContext(ctx, "redis health check failed", "error", err)
		} else {
			checks["redis"] = "ok"
		}
	}

	// Check LiveKit availability (if configured)
	if h.livekitChecker != nil {
		if err := h.livekitChecker.HealthCheck(ctx); err != nil {
			checks["livekit"] = "error"
			healthy = false
			slog.WarnContext(ctx, "livekit health check failed", "error", err)
		} else {
			checks["livekit"] = "ok"
		}
	}

	// Calculate uptime
	uptime := int64(time.Since(h.startTime).Seconds())

	status := "up"
	statusCode := http.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:  status,
		Checks:  checks,
		UptimeS: uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	
	// Encode response before writing status code to handle encoding errors properly
	body, err := json.Marshal(response)
	if err != nil {
		slog.Error("failed to encode readiness response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(statusCode)
	if _, err := w.Write(body); err != nil {
		slog.Error("failed to write readiness response", "error", err)
	}
}
