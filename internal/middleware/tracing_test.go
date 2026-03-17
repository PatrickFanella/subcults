package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracing_CreatesSpan(t *testing.T) {
	// Create a test tracer with a span recorder
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	handler := Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify span was created
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	expectedSpanName := "GET /test"
	if span.Name() != expectedSpanName {
		t.Errorf("expected span name %q, got %q", expectedSpanName, span.Name())
	}

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestTracing_PropagatesContext(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	var capturedTraceID string
	var capturedSpanID string

	handler := Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = GetTraceID(r)
		capturedSpanID = GetSpanID(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/events", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify trace and span IDs were captured
	if capturedTraceID == "" {
		t.Error("expected non-empty trace ID")
	}

	if capturedSpanID == "" {
		t.Error("expected non-empty span ID")
	}

	// Verify IDs match the created span
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.SpanContext().TraceID().String() != capturedTraceID {
		t.Errorf("trace ID mismatch: expected %s, got %s",
			span.SpanContext().TraceID().String(), capturedTraceID)
	}

	if span.SpanContext().SpanID().String() != capturedSpanID {
		t.Errorf("span ID mismatch: expected %s, got %s",
			span.SpanContext().SpanID().String(), capturedSpanID)
	}
}

func TestTracing_DifferentMethods(t *testing.T) {
	tests := []struct {
		method       string
		path         string
		expectedName string
	}{
		{http.MethodGet, "/events", "GET /events"},
		{http.MethodPost, "/events", "POST /events"},
		{http.MethodPatch, "/events/123", "PATCH /events/123"},
		{http.MethodDelete, "/posts/456", "DELETE /posts/456"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			// Create a new span recorder for each test
			spanRecorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			otel.SetTracerProvider(tp)
			defer tp.Shutdown(context.Background())

			handler := Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			spans := spanRecorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("expected 1 span, got %d", len(spans))
			}

			span := spans[0]
			if span.Name() != tt.expectedName {
				t.Errorf("expected span name %q, got %q", tt.expectedName, span.Name())
			}
		})
	}
}

func TestGetTraceID_NoActiveSpan(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	traceID := GetTraceID(req)
	if traceID != "" {
		t.Errorf("expected empty trace ID for request without span, got %q", traceID)
	}
}

func TestGetSpanID_NoActiveSpan(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	spanID := GetSpanID(req)
	if spanID != "" {
		t.Errorf("expected empty span ID for request without span, got %q", spanID)
	}
}
