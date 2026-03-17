// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
)

// ProfilingConfig configures the profiling middleware.
type ProfilingConfig struct {
	// Enabled controls whether profiling endpoints are exposed.
	// SECURITY: This should ONLY be true in development environments.
	// NEVER enable profiling in production as it exposes sensitive runtime information.
	Enabled bool

	// Environment is used for additional safety checks (should be "development" or "dev").
	Environment string
}

// Profiling returns middleware that exposes pprof profiling endpoints at /debug/pprof/*.
// This middleware should ONLY be enabled in development environments.
//
// Available endpoints:
//   - /debug/pprof/          - Index page with all available profiles
//   - /debug/pprof/profile   - CPU profile (30s by default, use ?seconds=X to customize)
//   - /debug/pprof/heap      - Memory heap profile
//   - /debug/pprof/goroutine - Goroutine profile
//   - /debug/pprof/block     - Block profile (contention on mutexes)
//   - /debug/pprof/mutex     - Mutex contention profile
//   - /debug/pprof/threadcreate - Thread creation profile
//   - /debug/pprof/allocs    - All memory allocations
//   - /debug/pprof/cmdline   - Command line invocation
//   - /debug/pprof/symbol    - Symbol lookup
//   - /debug/pprof/trace     - Execution trace
//
// SECURITY WARNING:
// Profiling endpoints expose sensitive information about your application's internals:
// - Memory contents (potentially including secrets)
// - Source code structure
// - Resource usage patterns
// - Performance characteristics
//
// These endpoints should NEVER be exposed in production environments.
// Use environment-based feature flags and strict access controls.
//
// Example usage:
//
//	// In development only
//	if cfg.ProfilingEnabled && cfg.Env == "development" {
//	    handler = middleware.Profiling(middleware.ProfilingConfig{
//	        Enabled: true,
//	        Environment: cfg.Env,
//	    })(handler)
//	}
func Profiling(config ProfilingConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// If profiling is disabled, just pass through
		if !config.Enabled {
			return next
		}

		// SECURITY: Additional safety check - never enable in production-like environments
		if config.Environment == "production" || config.Environment == "prod" {
			slog.Error("SECURITY VIOLATION: profiling cannot be enabled in production environment",
				"environment", config.Environment,
			)
			// Return handler without profiling
			return next
		}

		slog.Warn("profiling endpoints enabled - DEVELOPMENT ONLY",
			"environment", config.Environment,
			"warning", "NEVER enable profiling in production",
			"endpoints", "/debug/pprof/*",
		)

		// Create a new mux for profiling routes
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request is for profiling endpoint
			if len(r.URL.Path) >= 12 && r.URL.Path[:12] == "/debug/pprof" {
				// Route to appropriate pprof handler
				switch r.URL.Path {
				case "/debug/pprof/cmdline":
					pprof.Cmdline(w, r)
				case "/debug/pprof/profile":
					pprof.Profile(w, r)
				case "/debug/pprof/symbol":
					pprof.Symbol(w, r)
				case "/debug/pprof/trace":
					pprof.Trace(w, r)
				default:
					// For /debug/pprof/ and /debug/pprof/<profile-name>, use Index handler
					pprof.Index(w, r)
				}
				return
			}

			// Not a profiling request, pass to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// ProfilingStatus returns a handler that reports the profiling status.
// This can be used as a health check endpoint to verify profiling configuration.
func ProfilingStatus(config ProfilingConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := "disabled"
		if config.Enabled {
			status = "enabled"
		}

		response := fmt.Sprintf(`{
  "profiling_enabled": %t,
  "environment": %q,
  "status": %q,
  "endpoints": [
    "/debug/pprof/",
    "/debug/pprof/profile",
    "/debug/pprof/heap",
    "/debug/pprof/goroutine",
    "/debug/pprof/block",
    "/debug/pprof/mutex",
    "/debug/pprof/threadcreate",
    "/debug/pprof/allocs",
    "/debug/pprof/cmdline",
    "/debug/pprof/symbol",
    "/debug/pprof/trace"
  ],
  "security_warning": "Profiling should NEVER be enabled in production"
}`, config.Enabled, config.Environment, status)

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(response)); err != nil {
			slog.Error("failed to write profiling status response", "error", err)
		}
	}
}
