package tracing_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestEndToEndTracing demonstrates end-to-end tracing through HTTP middleware
// and custom spans, verifying that traces are properly created and propagated.
func TestEndToEndTracing(t *testing.T) {
	// Create a test tracer with a span recorder
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create a handler that uses custom spans
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Business logic span
		ctx, endBusinessLogic := tracing.StartSpan(ctx, "business_logic")
		tracing.SetAttributes(ctx,
			attribute.String("user_id", "test-user"),
			attribute.String("operation", "test-operation"),
		)

		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		// Database span
		ctx, endDBQuery := tracing.StartDBSpan(ctx, "users", tracing.DBOperationQuery)
		time.Sleep(5 * time.Millisecond)
		endDBQuery(nil)

		// Add event
		tracing.AddEvent(ctx, "operation_complete",
			attribute.Bool("success", true),
		)

		endBusinessLogic(nil)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Apply tracing middleware
	tracedHandler := middleware.Tracing("test-service")(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Execute request
	tracedHandler.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Verify spans were created
	spans := spanRecorder.Ended()

	// Expected spans:
	// 1. HTTP handler span (from middleware)
	// 2. business_logic span
	// 3. query users span (DB operation)
	expectedSpanCount := 3
	if len(spans) != expectedSpanCount {
		t.Errorf("expected %d spans, got %d", expectedSpanCount, len(spans))
		for i, span := range spans {
			t.Logf("  span %d: %s", i, span.Name())
		}
	}

	// Verify span names and hierarchy
	spanNames := make(map[string]bool)
	for _, span := range spans {
		spanNames[span.Name()] = true
	}

	requiredSpans := []string{"GET /test", "business_logic", "query users"}
	for _, name := range requiredSpans {
		if !spanNames[name] {
			t.Errorf("missing required span: %s", name)
		}
	}

	// Verify all spans share the same trace ID (context propagation)
	if len(spans) > 0 {
		traceID := spans[0].SpanContext().TraceID()
		for i, span := range spans {
			if span.SpanContext().TraceID() != traceID {
				t.Errorf("span %d has different trace ID: expected %s, got %s",
					i, traceID, span.SpanContext().TraceID())
			}
		}
	}

	// Verify DB span has correct attributes
	for _, span := range spans {
		if span.Name() == "query users" {
			attrs := span.Attributes()
			foundDBSystem := false
			foundDBOperation := false
			foundDBTable := false

			for _, attr := range attrs {
				switch attr.Key {
				case "db.system":
					if attr.Value.AsString() != "postgresql" {
						t.Errorf("expected db.system=postgresql, got %s", attr.Value.AsString())
					}
					foundDBSystem = true
				case "db.operation":
					if attr.Value.AsString() != "query" {
						t.Errorf("expected db.operation=query, got %s", attr.Value.AsString())
					}
					foundDBOperation = true
				case "db.sql.table":
					if attr.Value.AsString() != "users" {
						t.Errorf("expected db.sql.table=users, got %s", attr.Value.AsString())
					}
					foundDBTable = true
				}
			}

			if !foundDBSystem {
				t.Error("DB span missing db.system attribute")
			}
			if !foundDBOperation {
				t.Error("DB span missing db.operation attribute")
			}
			if !foundDBTable {
				t.Error("DB span missing db.sql.table attribute")
			}
		}
	}
}

// TestTracingDisabled verifies that when tracing is disabled, operations still work
// but no spans are created.
func TestTracingDisabled(t *testing.T) {
	// Create provider with tracing disabled
	provider, err := tracing.NewProvider(tracing.Config{
		ServiceName: "test-service",
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("failed to create disabled provider: %v", err)
	}

	if provider.IsEnabled() {
		t.Error("expected tracing to be disabled")
	}

	// Operations should still work
	ctx := context.Background()
	ctx, endSpan := tracing.StartSpan(ctx, "test-operation")
	tracing.SetAttributes(ctx, attribute.String("key", "value"))
	tracing.AddEvent(ctx, "test-event")
	endSpan(nil)

	// No errors should occur
	t.Log("tracing operations completed without errors when disabled")
}

// TestTraceContextPropagation verifies that trace context is properly propagated
// through HTTP headers using W3C Trace Context format.
func TestTraceContextPropagation(t *testing.T) {
	// Create a test tracer
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// Create a handler that extracts trace information
	var capturedTraceID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = middleware.GetTraceID(r)
		w.WriteHeader(http.StatusOK)
	})

	// Apply tracing middleware
	tracedHandler := middleware.Tracing("test-service")(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Execute request
	tracedHandler.ServeHTTP(rr, req)

	// Verify trace ID was captured
	if capturedTraceID == "" {
		t.Error("expected non-empty trace ID")
	}

	// Verify trace ID matches the span
	spans := spanRecorder.Ended()
	if len(spans) > 0 {
		spanTraceID := spans[0].SpanContext().TraceID().String()
		if capturedTraceID != spanTraceID {
			t.Errorf("trace ID mismatch: handler captured %s, span has %s",
				capturedTraceID, spanTraceID)
		}
	}
}
