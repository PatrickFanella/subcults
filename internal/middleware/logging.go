// Package middleware provides HTTP middleware components for the API server.
package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// userDIDKey is the context key for user DID.
type userDIDKey struct{}

// errorCodeKey is the context key for error code.
type errorCodeKey struct{}

// rateLimitKeyKey is the context key for rate limit key (user ID or IP).
type rateLimitKeyKey struct{}

// SetUserDID stores the user DID in the context.
// This should be called by authentication middleware after validating the token.
func SetUserDID(ctx context.Context, did string) context.Context {
	return context.WithValue(ctx, userDIDKey{}, did)
}

// GetUserDID retrieves the user DID from context. Returns empty string if not present.
func GetUserDID(ctx context.Context) string {
	if did, ok := ctx.Value(userDIDKey{}).(string); ok {
		return did
	}
	return ""
}

// SetErrorCode stores an error code in the context.
// This should be called by handlers when returning error responses.
func SetErrorCode(ctx context.Context, code string) context.Context {
	return context.WithValue(ctx, errorCodeKey{}, code)
}

// GetErrorCode retrieves the error code from context. Returns empty string if not present.
func GetErrorCode(ctx context.Context) string {
	if code, ok := ctx.Value(errorCodeKey{}).(string); ok {
		return code
	}
	return ""
}

// SetRateLimitKey stores the rate limit key in the context.
// This should be called by rate limiting middleware to track which key was rate limited.
func SetRateLimitKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, rateLimitKeyKey{}, key)
}

// GetRateLimitKey retrieves the rate limit key from context. Returns empty string if not present.
func GetRateLimitKey(ctx context.Context) string {
	if key, ok := ctx.Value(rateLimitKeyKey{}).(string); ok {
		return key
	}
	return ""
}

// responseWriter wraps http.ResponseWriter to capture status code, response size, and context.
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	size        int
	wroteHeader bool
	ctx         context.Context // stores the latest context from handlers
}

// WriteHeader captures the status code before writing it.
// Only the first call sets the status code; subsequent calls are ignored
// to match http.ResponseWriter behavior where only the first status is sent.
func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size and writes the data.
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// SetContext allows handlers to update the context that will be used for logging.
// This is called by error handling code to propagate error codes to the logging middleware.
func (rw *responseWriter) SetContext(ctx context.Context) {
	rw.ctx = ctx
}

// Context returns the latest context, falling back to the original if never set.
func (rw *responseWriter) Context() context.Context {
	if rw.ctx != nil {
		return rw.ctx
	}
	return context.Background()
}

// ContextSetter is an interface for response writers that support context updates.
// This allows error handling code to propagate context changes to the logging middleware.
type ContextSetter interface {
	SetContext(ctx context.Context)
}

// UpdateResponseContext updates the context in the response writer if it supports it.
// This should be called by handlers after setting error codes in the context.
func UpdateResponseContext(w http.ResponseWriter, ctx context.Context) {
	if cs, ok := w.(ContextSetter); ok {
		cs.SetContext(ctx)
	}
}

// newResponseWriter creates a new responseWriter with default 200 status.
func newResponseWriter(w http.ResponseWriter, initialCtx context.Context) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		ctx:            initialCtx,
	}
}

// NewLogger creates an slog.Logger based on the environment.
// In production (env == "production"), it returns a JSON handler.
// Otherwise, it returns a text handler for development.
func NewLogger(env string) *slog.Logger {
	return newLoggerWithWriter(env, os.Stdout)
}

// newLoggerWithWriter creates an slog.Logger with a custom writer.
// This is primarily used for testing to capture log output.
func newLoggerWithWriter(env string, w io.Writer) *slog.Logger {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	return slog.New(handler)
}

// Logging is a middleware that logs HTTP requests with structured fields.
// It captures: method, path, status, latency (ms), request ID, user DID (if present),
// response size, and error_code (for error responses).
//
// Note: If a handler panics, the log entry will not be written. To ensure logging
// even on panics, place a recovery middleware outside of the logging middleware.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status, size, and context
			rw := newResponseWriter(w, r.Context())

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate latency in milliseconds
			latency := time.Since(start).Milliseconds()

			// Get the final context (may have been updated by handlers)
			finalCtx := rw.Context()

			// Build log attributes
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Int64("latency_ms", latency),
				slog.Int("size", rw.size),
			}

			// Add request ID if present
			if requestID := GetRequestID(finalCtx); requestID != "" {
				attrs = append(attrs, slog.String("request_id", requestID))
			}

			// Add user DID if present
			if userDID := GetUserDID(finalCtx); userDID != "" {
				attrs = append(attrs, slog.String("user_did", userDID))
			}

			// Add error code for error responses (4xx and 5xx)
			if rw.statusCode >= 400 {
				if errorCode := GetErrorCode(finalCtx); errorCode != "" {
					attrs = append(attrs, slog.String("error_code", errorCode))

					// Add rate limit key if this is a rate limit violation
					if errorCode == "rate_limit_exceeded" {
						if rateLimitKey := GetRateLimitKey(finalCtx); rateLimitKey != "" {
							attrs = append(attrs, slog.String("rate_limit_key", rateLimitKey))
						}
					}
				}
			}

			// Log at appropriate level based on status code using LogAttrs
			if rw.statusCode >= 500 {
				logger.LogAttrs(finalCtx, slog.LevelError, "request completed", attrs...)
			} else if rw.statusCode >= 400 {
				logger.LogAttrs(finalCtx, slog.LevelWarn, "request completed", attrs...)
			} else {
				logger.LogAttrs(finalCtx, slog.LevelInfo, "request completed", attrs...)
			}
		})
	}
}
