// Package middleware provides metrics for HTTP middleware components.
package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics names as constants for consistency.
const (
	MetricRateLimitRequests      = "rate_limit_requests_total"
	MetricRateLimitBlocked       = "rate_limit_blocked_total"
	MetricRateLimitRedisErrors   = "rate_limit_redis_errors_total"
	MetricHTTPRequestDuration    = "http_request_duration_seconds"
	MetricHTTPRequestsTotal      = "http_requests_total"
	MetricHTTPRequestSizeBytes   = "http_request_size_bytes"
	MetricHTTPResponseSizeBytes  = "http_response_size_bytes"
	MetricCanaryRequestsTotal    = "canary_requests_total"
	MetricCanaryErrorsTotal      = "canary_errors_total"
	MetricCanaryLatencySeconds   = "canary_latency_seconds"
	MetricCanaryActive           = "canary_active"
)

// Metrics contains Prometheus metrics for middleware operations.
// All operations are thread-safe.
type Metrics struct {
	rateLimitRequests    *prometheus.CounterVec
	rateLimitBlocked     *prometheus.CounterVec
	rateLimitRedisErrors prometheus.Counter
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestSize      *prometheus.HistogramVec
	httpResponseSize     *prometheus.HistogramVec
	canaryRequestsTotal  *prometheus.CounterVec
	canaryErrorsTotal    *prometheus.CounterVec
	canaryLatency        *prometheus.HistogramVec
	canaryActive         prometheus.Gauge
}

// NewMetrics creates and returns a new Metrics instance with all collectors initialized.
// The metrics are not registered; call Register to register them with a registry.
func NewMetrics() *Metrics {
	return &Metrics{
		rateLimitRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricRateLimitRequests,
				Help: "Total number of rate limit checks by endpoint",
			},
			[]string{"endpoint", "key_type"},
		),
		rateLimitBlocked: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricRateLimitBlocked,
				Help: "Total number of rate limit violations (blocked requests) by endpoint",
			},
			[]string{"endpoint", "key_type"},
		),
		rateLimitRedisErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: MetricRateLimitRedisErrors,
				Help: "Total number of Redis errors during rate limiting (fail-open events)",
			},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    MetricHTTPRequestDuration,
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.01, 0.1, 0.5, 1.0, 2.0},
			},
			[]string{"method", "path", "status"},
		),
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricHTTPRequestsTotal,
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    MetricHTTPRequestSizeBytes,
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100 B to ~100 MB
			},
			[]string{"method", "path", "status"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    MetricHTTPResponseSizeBytes,
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100 B to ~100 MB
			},
			[]string{"method", "path", "status"},
		),
		canaryRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricCanaryRequestsTotal,
				Help: "Total number of requests by canary cohort",
			},
			[]string{"cohort", "version"},
		),
		canaryErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricCanaryErrorsTotal,
				Help: "Total number of errors by canary cohort",
			},
			[]string{"cohort", "version"},
		),
		canaryLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    MetricCanaryLatencySeconds,
				Help:    "Request latency by canary cohort",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"cohort", "version"},
		),
		canaryActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: MetricCanaryActive,
				Help: "Whether canary deployment is currently active (1 = active, 0 = inactive)",
			},
		),
	}
}

// Register registers all metrics with the given registry.
// Returns an error if registration fails.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.rateLimitRequests,
		m.rateLimitBlocked,
		m.rateLimitRedisErrors,
		m.httpRequestDuration,
		m.httpRequestsTotal,
		m.httpRequestSize,
		m.httpResponseSize,
		m.canaryRequestsTotal,
		m.canaryErrorsTotal,
		m.canaryLatency,
		m.canaryActive,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// IncRateLimitRequests increments the rate limit requests counter.
// endpoint: The API endpoint being accessed (e.g., "/search/events")
// keyType: The type of rate limit key (e.g., "user", "ip")
func (m *Metrics) IncRateLimitRequests(endpoint, keyType string) {
	m.rateLimitRequests.WithLabelValues(endpoint, keyType).Inc()
}

// IncRateLimitBlocked increments the rate limit blocked counter.
// endpoint: The API endpoint being accessed (e.g., "/search/events")
// keyType: The type of rate limit key (e.g., "user", "ip")
func (m *Metrics) IncRateLimitBlocked(endpoint, keyType string) {
	m.rateLimitBlocked.WithLabelValues(endpoint, keyType).Inc()
}

// IncRateLimitRedisErrors increments the Redis error counter.
// This tracks fail-open events when Redis is unavailable.
func (m *Metrics) IncRateLimitRedisErrors() {
	m.rateLimitRedisErrors.Inc()
}

// ObserveHTTPRequest records HTTP request metrics.
// method: HTTP method (e.g., "GET", "POST")
// path: Request path (e.g., "/events")
// status: HTTP status code (e.g., 200, 404)
// duration: Request duration in seconds
// requestSize: Request body size in bytes
// responseSize: Response body size in bytes
func (m *Metrics) ObserveHTTPRequest(method, path, status string, duration float64, requestSize, responseSize int64) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": status,
	}
	m.httpRequestDuration.With(labels).Observe(duration)
	m.httpRequestsTotal.With(labels).Inc()
	m.httpRequestSize.With(labels).Observe(float64(requestSize))
	m.httpResponseSize.With(labels).Observe(float64(responseSize))
}

// Collectors returns all Prometheus collectors for testing.
func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.rateLimitRequests,
		m.rateLimitBlocked,
		m.rateLimitRedisErrors,
		m.httpRequestDuration,
		m.httpRequestsTotal,
		m.httpRequestSize,
		m.httpResponseSize,
		m.canaryRequestsTotal,
		m.canaryErrorsTotal,
		m.canaryLatency,
		m.canaryActive,
	}
}

// ObserveCanaryRequest records metrics for a canary request.
// cohort: "canary" or "stable"
// version: version identifier (e.g., "v1.2.0-canary" or "stable")
// duration: request duration in seconds
// isError: whether the request resulted in an error (5xx status)
func (m *Metrics) ObserveCanaryRequest(cohort, version string, duration float64, isError bool) {
	m.canaryRequestsTotal.WithLabelValues(cohort, version).Inc()
	if isError {
		m.canaryErrorsTotal.WithLabelValues(cohort, version).Inc()
	}
	m.canaryLatency.WithLabelValues(cohort, version).Observe(duration)
}

// SetCanaryActive sets the canary active gauge (1 = active, 0 = inactive).
func (m *Metrics) SetCanaryActive(active bool) {
	if active {
		m.canaryActive.Set(1)
	} else {
		m.canaryActive.Set(0)
	}
}
