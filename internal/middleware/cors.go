// Package middleware provides HTTP middleware components for the API server.
package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string // List of allowed origins (no wildcards)
	AllowedMethods   []string // List of allowed HTTP methods
	AllowedHeaders   []string // List of allowed headers
	AllowCredentials bool     // Whether to allow credentials
	MaxAge           int      // Preflight cache duration in seconds
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS).
// It enforces strict origin validation (no wildcards) and supports preflight requests.
//
// Configuration:
//   - AllowedOrigins: Explicit list of allowed origins. If empty, CORS is disabled.
//   - AllowedMethods: HTTP methods to allow. Defaults to GET, POST, PUT, PATCH, DELETE, OPTIONS.
//   - AllowedHeaders: Headers to allow. Defaults to Content-Type, Authorization, X-Request-ID.
//   - AllowCredentials: Whether to allow credentials (cookies, auth headers).
//   - MaxAge: How long browsers can cache preflight responses (in seconds).
//
// Security:
//   - No wildcard origins - only explicitly listed origins are allowed
//   - Validates origin against allowlist on every request
//   - Preflight OPTIONS requests are handled automatically
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	// Build origin map for fast lookup
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range cfg.AllowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowedOriginsMap[origin] = true
		}
	}

	// Convert slices to comma-separated strings for headers
	allowedMethodsStr := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeadersStr := strings.Join(cfg.AllowedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no origins configured, CORS is disabled - skip processing
			if len(allowedOriginsMap) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")

			// If no origin header, this is a same-origin request - allow it
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Validate origin against allowlist
			if !allowedOriginsMap[origin] {
				// Origin not allowed - set error code for logging and reject with 403 Forbidden
				ctx := SetErrorCode(r.Context(), "cors_origin_not_allowed")
				r = r.WithContext(ctx)
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}

			// Origin is allowed - set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", allowedMethodsStr)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeadersStr)
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// For actual requests, only origin and credentials headers are needed
			// (methods and headers are only required for preflight)
			next.ServeHTTP(w, r)
		})
	}
}
