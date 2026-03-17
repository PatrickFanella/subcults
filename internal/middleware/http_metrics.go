// Package middleware provides HTTP middleware components for the API server.
package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// pathNormalizer is a compiled regex for normalizing dynamic path segments.
var pathNormalizer = regexp.MustCompile(`/[^/]+`)

// normalizePath converts paths with dynamic segments to route patterns to prevent
// cardinality explosion in metrics. This maps paths like /events/123 to /events/{id}.
func normalizePath(path string) string {
	// Exact matches for static routes (no normalization needed)
	staticRoutes := map[string]bool{
		"/":                  true,
		"/events":            true,
		"/scenes":            true,
		"/posts":             true,
		"/streams":           true,
		"/search/events":     true,
		"/search/scenes":     true,
		"/search/posts":      true,
		"/search/global":     true,
		"/livekit/token":     true,
		"/uploads/sign":      true,
		"/payments/onboard":  true,
		"/payments/checkout": true,
		"/payments/status":   true,
		"/internal/stripe":   true,
		"/health":            true,
		"/ready":             true,
		"/metrics":           true,
	}

	if staticRoutes[path] {
		return path
	}

	// Pattern-based normalization for dynamic routes
	// Handle specific known patterns first for accuracy

	// /events/{id}/... patterns
	if strings.HasPrefix(path, "/events/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			// /events/{id}/cancel, /events/{id}/rsvp, /events/{id}/feed
			if len(parts) == 4 && (parts[3] == "cancel" || parts[3] == "rsvp" || parts[3] == "feed") {
				return "/events/{id}/" + parts[3]
			}
			// /events/{id}
			if len(parts) == 3 && parts[2] != "" {
				return "/events/{id}"
			}
		}
	}

	// /scenes/{id}/... patterns
	if strings.HasPrefix(path, "/scenes/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			// /scenes/{id}/feed
			if len(parts) == 4 && parts[3] == "feed" {
				return "/scenes/{id}/feed"
			}
			// /scenes/{id}
			if len(parts) == 3 && parts[2] != "" {
				return "/scenes/{id}"
			}
		}
	}

	// /streams/{id}/... patterns
	if strings.HasPrefix(path, "/streams/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			// /streams/{id}/end, /streams/{id}/join, /streams/{id}/leave, etc.
			if len(parts) == 4 {
				switch parts[3] {
				case "end", "join", "leave", "analytics", "lock", "featured_participant":
					return "/streams/{id}/" + parts[3]
				case "participants":
					return "/streams/{id}/participants"
				}
			}
			// /streams/{id}/participants/{participant_id}/mute or kick
			if len(parts) == 6 && parts[3] == "participants" && (parts[5] == "mute" || parts[5] == "kick") {
				return "/streams/{id}/participants/{participant_id}/" + parts[5]
			}
			// /streams/{id}
			if len(parts) == 3 && parts[2] != "" {
				return "/streams/{id}"
			}
		}
	}

	// /posts/{id}
	if strings.HasPrefix(path, "/posts/") {
		parts := strings.Split(path, "/")
		if len(parts) == 3 && parts[2] != "" {
			return "/posts/{id}"
		}
	}

	// /trust/{did}
	if strings.HasPrefix(path, "/trust/") {
		parts := strings.Split(path, "/")
		if len(parts) == 3 && parts[2] != "" {
			return "/trust/{id}"
		}
	}

	// Fallback: return as-is for unknown patterns
	// This ensures we don't accidentally break metrics for new routes
	return path
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code and response size.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	size        int64
	wroteHeader bool
}

// WriteHeader captures the status code before writing it.
func (mrw *metricsResponseWriter) WriteHeader(code int) {
	if mrw.wroteHeader {
		return
	}
	mrw.statusCode = code
	mrw.wroteHeader = true
	mrw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size and writes the data.
func (mrw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := mrw.ResponseWriter.Write(b)
	mrw.size += int64(n)
	return n, err
}

// newMetricsResponseWriter creates a new metricsResponseWriter with default 200 status.
func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// HTTPMetrics is a middleware that records HTTP request metrics.
// It captures duration, request/response sizes, and request counts.
// Health check endpoints (/health, /ready) are excluded from metrics to avoid cardinality issues.
func HTTPMetrics(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exclude health check endpoints from metrics
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Wrap response writer to capture status and size
			mrw := newMetricsResponseWriter(w)

			// Get request size from Content-Length header
			requestSize := int64(0)
			if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
				if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
					requestSize = size
				}
			}

			// Call the next handler
			next.ServeHTTP(mrw, r)

			// Calculate duration in seconds
			duration := time.Since(start).Seconds()

			// Normalize path to prevent cardinality explosion
			normalizedPath := normalizePath(r.URL.Path)

			// Record metrics
			metrics.ObserveHTTPRequest(
				r.Method,
				normalizedPath,
				strconv.Itoa(mrw.statusCode),
				duration,
				requestSize,
				mrw.size,
			)
		})
	}
}
