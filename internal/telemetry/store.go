package telemetry

import (
	"context"
	"errors"
)

// ErrDuplicateError is returned when an error log already exists for the
// given session + message hash combination.
var ErrDuplicateError = errors.New("duplicate error log")

// Store defines the persistence interface for telemetry events and client error logs.
type Store interface {
	// InsertEvents persists a batch of telemetry events.
	InsertEvents(ctx context.Context, events []TelemetryEvent) error

	// InsertClientError persists a client error log. Returns ErrDuplicateError
	// if the error_hash + session_id already exists.
	InsertClientError(ctx context.Context, errLog ClientErrorLog) (string, error)

	// InsertReplayEvents persists session replay events linked to an error log.
	InsertReplayEvents(ctx context.Context, errorLogID string, events []ReplayEvent) error
}
