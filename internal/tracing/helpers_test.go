package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestStartDBSpan(t *testing.T) {
	// Create a test tracer with a span recorder
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()

	tests := []struct {
		name      string
		table     string
		operation DBOperation
	}{
		{"query with table", "users", DBOperationQuery},
		{"insert with table", "events", DBOperationInsert},
		{"update with table", "scenes", DBOperationUpdate},
		{"delete with table", "posts", DBOperationDelete},
		{"exec with table", "migrations", DBOperationExec},
		{"query without table", "", DBOperationQuery},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new span recorder for each test
			spanRecorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			otel.SetTracerProvider(tp)
			defer tp.Shutdown(context.Background())

			_, endSpan := StartDBSpan(ctx, tt.table, tt.operation)
			endSpan(nil)

			spans := spanRecorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("expected 1 span, got %d", len(spans))
			}

			span := spans[0]

			// Verify span name
			expectedName := string(tt.operation)
			if tt.table != "" {
				expectedName = expectedName + " " + tt.table
			}
			if span.Name() != expectedName {
				t.Errorf("expected span name %q, got %q", expectedName, span.Name())
			}

			// Verify attributes
			attrs := span.Attributes()
			hasDBSystem := false
			hasDBOperation := false
			hasDBTable := false

			for _, attr := range attrs {
				switch attr.Key {
				case "db.system":
					hasDBSystem = true
					if attr.Value.AsString() != "postgresql" {
						t.Errorf("expected db.system=postgresql, got %s", attr.Value.AsString())
					}
				case "db.operation":
					hasDBOperation = true
					if attr.Value.AsString() != string(tt.operation) {
						t.Errorf("expected db.operation=%s, got %s", tt.operation, attr.Value.AsString())
					}
				case "db.sql.table":
					hasDBTable = true
					if attr.Value.AsString() != tt.table {
						t.Errorf("expected db.sql.table=%s, got %s", tt.table, attr.Value.AsString())
					}
				}
			}

			if !hasDBSystem {
				t.Error("missing db.system attribute")
			}
			if !hasDBOperation {
				t.Error("missing db.operation attribute")
			}
			if tt.table != "" && !hasDBTable {
				t.Error("missing db.sql.table attribute")
			}
			if tt.table == "" && hasDBTable {
				t.Error("unexpected db.sql.table attribute")
			}
		})
	}
}

func TestStartDBSpan_WithError(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	testErr := errors.New("database error")

	_, endSpan := StartDBSpan(ctx, "users", DBOperationQuery)
	endSpan(testErr)

	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]

	// Verify error was recorded
	// Status code 2 is Error in OpenTelemetry
	if span.Status().Code.String() != "Error" {
		t.Errorf("expected error status, got %s", span.Status().Code.String())
	}

	if span.Status().Description != testErr.Error() {
		t.Errorf("expected error description %q, got %q", testErr.Error(), span.Status().Description)
	}
}

func TestStartSpan(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()

	spanName := "compute_trust_score"
	_, endSpan := StartSpan(ctx, spanName)
	endSpan(nil)

	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name() != spanName {
		t.Errorf("expected span name %q, got %q", spanName, span.Name())
	}

	// Verify success status (Unset is the default for successful operations)
	if span.Status().Code.String() != "Unset" && span.Status().Code.String() != "Ok" {
		t.Errorf("expected Unset or Ok status, got %s", span.Status().Code.String())
	}
}

func TestStartSpan_WithError(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	ctx := context.Background()
	testErr := errors.New("computation error")

	_, endSpan := StartSpan(ctx, "compute_trust_score")
	endSpan(testErr)

	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]

	// Verify error was recorded
	if span.Status().Code.String() != "Error" {
		t.Errorf("expected error status, got %s", span.Status().Code.String())
	}
}

func TestAddEvent(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")

	eventName := "cache_hit"
	AddEvent(ctx, eventName,
		attribute.String("cache_key", "user:123"),
		attribute.Int("ttl", 3600),
	)

	span.End()

	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	events := spans[0].Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Name != eventName {
		t.Errorf("expected event name %q, got %q", eventName, events[0].Name)
	}

	// Verify event attributes
	attrs := events[0].Attributes
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
}

func TestSetAttributes(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")

	SetAttributes(ctx,
		attribute.String("user_id", "did:example:123"),
		attribute.String("endpoint", "/events"),
	)

	span.End()

	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	attrs := spans[0].Attributes()
	if len(attrs) < 2 {
		t.Fatalf("expected at least 2 attributes, got %d", len(attrs))
	}

	// Verify specific attributes
	hasUserID := false
	hasEndpoint := false
	for _, attr := range attrs {
		switch attr.Key {
		case "user_id":
			hasUserID = true
			if attr.Value.AsString() != "did:example:123" {
				t.Errorf("expected user_id=did:example:123, got %s", attr.Value.AsString())
			}
		case "endpoint":
			hasEndpoint = true
			if attr.Value.AsString() != "/events" {
				t.Errorf("expected endpoint=/events, got %s", attr.Value.AsString())
			}
		}
	}

	if !hasUserID {
		t.Error("missing user_id attribute")
	}
	if !hasEndpoint {
		t.Error("missing endpoint attribute")
	}
}
