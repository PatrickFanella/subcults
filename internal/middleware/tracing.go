// Package middleware provides HTTP middleware components for the API server.
package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Tracing creates HTTP middleware that instruments requests with OpenTelemetry spans.
// It uses W3C Trace Context propagation and integrates with the existing RequestID middleware.
//
// Middleware creates spans with the following attributes:
// - http.method: HTTP method (GET, POST, etc.)
// - http.url: Full request URL
// - http.status_code: HTTP response status code
// - http.route: Route pattern (if available)
//
// It automatically propagates trace context using W3C Trace Context headers:
// - traceparent: Contains trace-id, parent-id, trace-flags
// - tracestate: Vendor-specific trace information
//
// The middleware should be placed in the middleware chain after RequestID
// to ensure request IDs are available in trace context.
func Tracing(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Use otelhttp instrumentation which handles:
		// - Span creation for each request
		// - W3C trace propagation (traceparent/tracestate headers)
		// - Automatic metric collection
		// - Error tracking
		return otelhttp.NewHandler(next, serviceName,
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				// Use HTTP method + path as span name (e.g., "GET /events")
				return r.Method + " " + r.URL.Path
			}),
		)
	}
}

// GetTraceID extracts the trace ID from the request context.
// Returns empty string if no trace is active.
func GetTraceID(r *http.Request) string {
	spanCtx := trace.SpanContextFromContext(r.Context())
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from the request context.
// Returns empty string if no span is active.
func GetSpanID(r *http.Request) string {
	spanCtx := trace.SpanContextFromContext(r.Context())
	if spanCtx.IsValid() {
		return spanCtx.SpanID().String()
	}
	return ""
}
