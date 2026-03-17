// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"crypto/subtle"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler creates an HTTP handler for the Prometheus metrics endpoint.
// It uses the provided registry to gather metrics.
func MetricsHandler(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

// InternalAuthMiddleware restricts access to requests with a valid token.
// If token is empty, no authentication is required.
// The token is checked against the X-Internal-Token header.
// Uses constant-time comparison to prevent timing attacks.
func InternalAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no token is configured, allow all requests
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for internal token header using constant-time comparison
			headerToken := r.Header.Get("X-Internal-Token")
			if subtle.ConstantTimeCompare([]byte(headerToken), []byte(token)) != 1 {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
