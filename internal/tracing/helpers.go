// Package tracing provides OpenTelemetry distributed tracing setup and utilities.
package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DBOperation represents the type of database operation being traced.
type DBOperation string

const (
	// DBOperationQuery represents a SELECT query.
	DBOperationQuery DBOperation = "query"
	// DBOperationInsert represents an INSERT operation.
	DBOperationInsert DBOperation = "insert"
	// DBOperationUpdate represents an UPDATE operation.
	DBOperationUpdate DBOperation = "update"
	// DBOperationDelete represents a DELETE operation.
	DBOperationDelete DBOperation = "delete"
	// DBOperationExec represents a generic EXEC operation.
	DBOperationExec DBOperation = "exec"
)

// StartDBSpan creates a new span for a database operation.
// Returns the new context and a function to end the span.
//
// Example usage:
//
//	ctx, endSpan := tracing.StartDBSpan(ctx, "scenes", tracing.DBOperationQuery)
//	defer endSpan(err)
//	// ... perform database operation ...
func StartDBSpan(ctx context.Context, table string, operation DBOperation) (context.Context, func(error)) {
	tracer := otel.Tracer("subcults/db")

	spanName := string(operation)
	if table != "" {
		spanName = spanName + " " + table
	}

	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", string(operation)),
		),
	)

	if table != "" {
		span.SetAttributes(attribute.String("db.sql.table", table))
	}

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// StartSpan creates a new span for a general operation.
// Returns the new context and a function to end the span.
//
// Example usage:
//
//	ctx, endSpan := tracing.StartSpan(ctx, "compute_trust_score")
//	defer endSpan(err)
//	// ... perform operation ...
func StartSpan(ctx context.Context, name string) (context.Context, func(error)) {
	tracer := otel.Tracer("subcults")

	ctx, span := tracer.Start(ctx, name)

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span.
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}
