// Package main provides an example of using OpenTelemetry tracing in the Subcults API.
// This example demonstrates how to instrument HTTP handlers and database operations.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/tracing"

	"go.opentelemetry.io/otel/attribute"
)

func main() {
	// Initialize logger
	logger := middleware.NewLogger("development")
	slog.SetDefault(logger)

	// Initialize OpenTelemetry tracing
	tracerProvider, err := tracing.NewProvider(tracing.Config{
		ServiceName:  "tracing-example",
		Enabled:      true,
		Environment:  "development",
		ExporterType: "otlp-http",
		OTLPEndpoint: "localhost:4318",
		SamplingRate: 1.0, // 100% sampling for demo
		InsecureMode: true,
	})
	if err != nil {
		logger.Error("failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	logger.Info("tracing initialized for example")

	// Create HTTP handlers
	mux := http.NewServeMux()

	// Example 1: Simple traced handler
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("handling hello request")

		// Get trace ID for logging correlation
		traceID := middleware.GetTraceID(r)
		logger.Info("processing request", "trace_id", traceID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Example 2: Handler with custom spans
	mux.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Create a custom span for business logic
		ctx, endBusinessLogic := tracing.StartSpan(ctx, "process_business_logic")

		// Add custom attributes
		tracing.SetAttributes(ctx,
			attribute.String("user.id", "example-user-123"),
			attribute.String("operation.type", "data_processing"),
		)

		// Simulate some work
		time.Sleep(100 * time.Millisecond)

		// Add an event
		tracing.AddEvent(ctx, "validation_complete",
			attribute.Bool("valid", true),
		)

		// Simulate database query
		ctx, endDBQuery := tracing.StartDBSpan(ctx, "users", tracing.DBOperationQuery)
		time.Sleep(50 * time.Millisecond)
		endDBQuery(nil) // No error

		endBusinessLogic(nil) // No error

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Processing complete"))
	})

	// Example 3: Handler with error tracing
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctx, endSpan := tracing.StartSpan(ctx, "failing_operation")

		// Simulate an error
		err := performFailingOperation(ctx)

		// Record error in span
		endSpan(err)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error occurred"))
	})

	// Apply middleware chain
	// Order: Tracing (outer) -> RequestID -> Logging (inner)
	handler := middleware.Tracing("tracing-example")(
		middleware.RequestID(
			middleware.Logging(logger)(mux),
		),
	)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8081",
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting example server on :8081")
		logger.Info("try these endpoints:")
		logger.Info("  curl http://localhost:8081/hello")
		logger.Info("  curl http://localhost:8081/process")
		logger.Info("  curl http://localhost:8081/error")
		logger.Info("view traces at http://localhost:16686")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown tracer provider first to flush pending spans
	if err := tracerProvider.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown tracer provider", "error", err)
	}

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

// performFailingOperation simulates a failing operation for demonstration.
func performFailingOperation(ctx context.Context) error {
	ctx, endSpan := tracing.StartSpan(ctx, "database_transaction")
	defer func() {
		// Error will be handled by defer
	}()

	// Simulate some work before failure
	time.Sleep(50 * time.Millisecond)

	// Return an error
	err := &DatabaseError{
		Table:   "users",
		Query:   "SELECT * FROM users WHERE id = $1",
		Message: "connection timeout",
	}

	endSpan(err)
	return err
}

// DatabaseError is a custom error type for demonstration.
type DatabaseError struct {
	Table   string
	Query   string
	Message string
}

func (e *DatabaseError) Error() string {
	return "database error: " + e.Message
}
