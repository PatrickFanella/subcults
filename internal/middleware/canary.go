// Package middleware provides canary deployment routing and monitoring.
package middleware

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// CanaryConfig holds configuration for canary deployment.
type CanaryConfig struct {
	Enabled            bool
	TrafficPercent     float64 // Percentage of traffic to route to canary (0-100)
	ErrorThreshold     float64 // Error rate threshold for auto-rollback (0-100)
	LatencyThreshold   float64 // Latency threshold in seconds for auto-rollback
	AutoRollback       bool    // Enable automatic rollback on threshold breach
	MonitoringWindow   int     // Monitoring window in seconds for metrics comparison
	Version            string  // Version identifier for canary deployment
}

// CanaryRouter manages canary deployment routing and monitoring.
type CanaryRouter struct {
	config          CanaryConfig
	metrics         *CanaryMetrics
	promMetrics     *Metrics // Prometheus metrics (optional)
	logger          *slog.Logger
	mu              sync.RWMutex
	active          bool // Current canary deployment status (can be disabled by rollback)
}

// CanaryMetrics tracks metrics for canary vs stable cohorts.
type CanaryMetrics struct {
	mu sync.RWMutex

	// Canary cohort metrics
	canaryRequests     int64
	canaryErrors       int64
	canaryLatencySum   float64
	canaryLatencyCount int64

	// Stable cohort metrics
	stableRequests     int64
	stableErrors       int64
	stableLatencySum   float64
	stableLatencyCount int64

	// Window tracking
	windowStart time.Time
}

// NewCanaryRouter creates a new canary router with the given configuration.
func NewCanaryRouter(config CanaryConfig, logger *slog.Logger) *CanaryRouter {
	return &CanaryRouter{
		config:  config,
		metrics: &CanaryMetrics{windowStart: time.Now()},
		logger:  logger,
		active:  config.Enabled,
	}
}

// SetPrometheusMetrics sets the Prometheus metrics collector for canary monitoring.
func (cr *CanaryRouter) SetPrometheusMetrics(metrics *Metrics) {
	cr.promMetrics = metrics
	// Initialize the canary active gauge
	if metrics != nil {
		metrics.SetCanaryActive(cr.active && cr.config.Enabled)
	}
}

// Middleware returns an HTTP middleware that routes requests to canary or stable versions.
// It uses deterministic hashing based on user ID or IP address to assign users to cohorts.
func (cr *CanaryRouter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cr.mu.RLock()
		enabled := cr.active && cr.config.Enabled
		cr.mu.RUnlock()

		if !enabled {
			// Canary disabled, route all traffic to stable
			r.Header.Set("X-Deployment-Cohort", "stable")
			r.Header.Set("X-Deployment-Version", "stable")
			w.Header().Set("X-Deployment-Cohort", "stable")
			w.Header().Set("X-Deployment-Version", "stable")
			next.ServeHTTP(w, r)
			return
		}

		// Determine cohort assignment
		cohort := cr.assignCohort(r)
		version := "stable"
		if cohort == "canary" {
			version = cr.config.Version
		}

		// Set headers for downstream tracking
		r.Header.Set("X-Deployment-Cohort", cohort)
		r.Header.Set("X-Deployment-Version", version)

		// Add response headers for observability
		w.Header().Set("X-Deployment-Cohort", cohort)
		w.Header().Set("X-Deployment-Version", version)

		// Wrap response writer to track metrics
		start := time.Now()
		wrapped := &canaryResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		isError := wrapped.statusCode >= 500

		cr.recordRequest(cohort, duration, isError)

		// Check for auto-rollback conditions
		if cr.config.AutoRollback && cohort == "canary" {
			cr.checkRollbackConditions()
		}
	})
}

// assignCohort determines which cohort (canary or stable) a request belongs to.
// Uses deterministic hashing based on user identifier or IP address.
func (cr *CanaryRouter) assignCohort(r *http.Request) string {
	// Try to get user identifier from context or headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		// Fallback to IP address for anonymous users
		userID = getClientIP(r)
	}

	// Hash the user identifier to get a deterministic cohort assignment
	hash := sha256.Sum256([]byte(userID))
	hashValue := binary.BigEndian.Uint64(hash[:8])

	// Convert hash to percentage (0-100)
	percentage := float64(hashValue%10000) / 100.0

	if percentage < cr.config.TrafficPercent {
		return "canary"
	}
	return "stable"
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fallback to RemoteAddr
	return r.RemoteAddr
}

// recordRequest records metrics for a request.
func (cr *CanaryRouter) recordRequest(cohort string, duration float64, isError bool) {
	version := "stable"
	if cohort == "canary" {
		version = cr.config.Version
	}

	// Record to internal metrics
	cr.metrics.mu.Lock()
	if cohort == "canary" {
		cr.metrics.canaryRequests++
		cr.metrics.canaryLatencySum += duration
		cr.metrics.canaryLatencyCount++
		if isError {
			cr.metrics.canaryErrors++
		}
	} else {
		cr.metrics.stableRequests++
		cr.metrics.stableLatencySum += duration
		cr.metrics.stableLatencyCount++
		if isError {
			cr.metrics.stableErrors++
		}
	}
	cr.metrics.mu.Unlock()

	// Record to Prometheus metrics if available
	if cr.promMetrics != nil {
		cr.promMetrics.ObserveCanaryRequest(cohort, version, duration, isError)
	}
}

// checkRollbackConditions checks if canary metrics exceed thresholds and triggers rollback.
func (cr *CanaryRouter) checkRollbackConditions() {
	cr.metrics.mu.RLock()

	// Need minimum sample size for reliable comparison
	if cr.metrics.canaryRequests < 100 {
		cr.metrics.mu.RUnlock()
		return
	}

	// Calculate error rates
	canaryErrorRate := float64(cr.metrics.canaryErrors) / float64(cr.metrics.canaryRequests) * 100
	var stableErrorRate float64
	if cr.metrics.stableRequests > 0 {
		stableErrorRate = float64(cr.metrics.stableErrors) / float64(cr.metrics.stableRequests) * 100
	}

	// Calculate average latencies
	var canaryAvgLatency, stableAvgLatency float64
	if cr.metrics.canaryLatencyCount > 0 {
		canaryAvgLatency = cr.metrics.canaryLatencySum / float64(cr.metrics.canaryLatencyCount)
	}
	if cr.metrics.stableLatencyCount > 0 {
		stableAvgLatency = cr.metrics.stableLatencySum / float64(cr.metrics.stableLatencyCount)
	}

	// Release lock before calling Rollback (which needs mu.Lock)
	cr.metrics.mu.RUnlock()

	// Check error rate threshold
	if canaryErrorRate > cr.config.ErrorThreshold {
		cr.logger.Error("canary rollback triggered: error rate exceeded threshold",
			"canary_error_rate", fmt.Sprintf("%.2f%%", canaryErrorRate),
			"stable_error_rate", fmt.Sprintf("%.2f%%", stableErrorRate),
			"threshold", fmt.Sprintf("%.2f%%", cr.config.ErrorThreshold),
		)
		cr.Rollback("error_rate_exceeded")
		return
	}

	// Check latency threshold
	if canaryAvgLatency > cr.config.LatencyThreshold {
		cr.logger.Error("canary rollback triggered: latency exceeded threshold",
			"canary_avg_latency", fmt.Sprintf("%.3fs", canaryAvgLatency),
			"stable_avg_latency", fmt.Sprintf("%.3fs", stableAvgLatency),
			"threshold", fmt.Sprintf("%.3fs", cr.config.LatencyThreshold),
		)
		cr.Rollback("latency_exceeded")
		return
	}

	// Check relative error rate (canary should not be significantly worse than stable)
	if stableErrorRate > 0 && canaryErrorRate > stableErrorRate*2 {
		cr.logger.Error("canary rollback triggered: error rate significantly higher than stable",
			"canary_error_rate", fmt.Sprintf("%.2f%%", canaryErrorRate),
			"stable_error_rate", fmt.Sprintf("%.2f%%", stableErrorRate),
		)
		cr.Rollback("relative_error_rate_high")
		return
	}
}

// Rollback disables canary deployment and routes all traffic to stable.
func (cr *CanaryRouter) Rollback(reason string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.active {
		return // Already rolled back
	}

	cr.active = false
	cr.logger.Warn("canary deployment rolled back",
		"reason", reason,
		"canary_version", cr.config.Version,
	)

	// Update Prometheus gauge
	if cr.promMetrics != nil {
		cr.promMetrics.SetCanaryActive(false)
	}
}

// GetMetrics returns current canary metrics snapshot.
func (cr *CanaryRouter) GetMetrics() MetricsSnapshot {
	cr.metrics.mu.RLock()
	defer cr.metrics.mu.RUnlock()

	canaryAvgLatency := 0.0
	if cr.metrics.canaryLatencyCount > 0 {
		canaryAvgLatency = cr.metrics.canaryLatencySum / float64(cr.metrics.canaryLatencyCount)
	}

	stableAvgLatency := 0.0
	if cr.metrics.stableLatencyCount > 0 {
		stableAvgLatency = cr.metrics.stableLatencySum / float64(cr.metrics.stableLatencyCount)
	}

	canaryErrorRate := 0.0
	if cr.metrics.canaryRequests > 0 {
		canaryErrorRate = float64(cr.metrics.canaryErrors) / float64(cr.metrics.canaryRequests) * 100
	}

	stableErrorRate := 0.0
	if cr.metrics.stableRequests > 0 {
		stableErrorRate = float64(cr.metrics.stableErrors) / float64(cr.metrics.stableRequests) * 100
	}

	return MetricsSnapshot{
		CanaryRequests:     cr.metrics.canaryRequests,
		CanaryErrors:       cr.metrics.canaryErrors,
		CanaryErrorRate:    canaryErrorRate,
		CanaryAvgLatency:   canaryAvgLatency,
		StableRequests:     cr.metrics.stableRequests,
		StableErrors:       cr.metrics.stableErrors,
		StableErrorRate:    stableErrorRate,
		StableAvgLatency:   stableAvgLatency,
		WindowStart:        cr.metrics.windowStart,
		WindowDuration:     time.Since(cr.metrics.windowStart),
		CanaryActive:       cr.active,
		CanaryVersion:      cr.config.Version,
	}
}

// ResetMetrics resets the metrics window.
func (cr *CanaryRouter) ResetMetrics() {
	cr.metrics.mu.Lock()
	defer cr.metrics.mu.Unlock()

	cr.metrics.canaryRequests = 0
	cr.metrics.canaryErrors = 0
	cr.metrics.canaryLatencySum = 0
	cr.metrics.canaryLatencyCount = 0
	cr.metrics.stableRequests = 0
	cr.metrics.stableErrors = 0
	cr.metrics.stableLatencySum = 0
	cr.metrics.stableLatencyCount = 0
	cr.metrics.windowStart = time.Now()
}

// MetricsSnapshot represents a point-in-time snapshot of canary metrics.
type MetricsSnapshot struct {
	CanaryRequests   int64         `json:"canary_requests"`
	CanaryErrors     int64         `json:"canary_errors"`
	CanaryErrorRate  float64       `json:"canary_error_rate"`
	CanaryAvgLatency float64       `json:"canary_avg_latency"`
	StableRequests   int64         `json:"stable_requests"`
	StableErrors     int64         `json:"stable_errors"`
	StableErrorRate  float64       `json:"stable_error_rate"`
	StableAvgLatency float64       `json:"stable_avg_latency"`
	WindowStart      time.Time     `json:"window_start"`
	WindowDuration   time.Duration `json:"window_duration"`
	CanaryActive     bool          `json:"canary_active"`
	CanaryVersion    string        `json:"canary_version"`
}

// canaryResponseWriter wraps http.ResponseWriter to capture status code.
type canaryResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *canaryResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
