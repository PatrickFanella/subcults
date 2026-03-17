// Package telemetry provides the domain models, store interfaces, and metrics
// for frontend telemetry event collection and client-side error logging.
package telemetry

import "time"

// TelemetryEvent represents a single analytics event from the frontend.
type TelemetryEvent struct {
	ID        string                 `json:"id,omitempty"`
	SessionID string                 `json:"sessionId"`
	UserDID   string                 `json:"userId,omitempty"`
	Name      string                 `json:"name"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp int64                  `json:"ts"`
	CreatedAt time.Time              `json:"createdAt,omitempty"`
}

// ClientErrorLog represents a client-side error report.
type ClientErrorLog struct {
	ID             string        `json:"id,omitempty"`
	SessionID      string        `json:"sessionId"`
	UserDID        string        `json:"userId,omitempty"`
	ErrorType      string        `json:"type"`
	ErrorMessage   string        `json:"message"`
	ErrorStack     string        `json:"stack,omitempty"`
	ComponentStack string        `json:"componentStack,omitempty"`
	URL            string        `json:"url,omitempty"`
	UserAgent      string        `json:"userAgent,omitempty"`
	ErrorHash      string        `json:"errorHash,omitempty"`
	OccurredAt     int64         `json:"timestamp"`
	CreatedAt      time.Time     `json:"createdAt,omitempty"`
	ReplayEvents   []ReplayEvent `json:"replayEvents,omitempty"`
}

// ReplayEvent represents a session replay snapshot (click, scroll, navigation, etc.)
// attached to an error log.
type ReplayEvent struct {
	ID             string                 `json:"id,omitempty"`
	ErrorLogID     string                 `json:"errorLogId,omitempty"`
	EventType      string                 `json:"type"`
	EventData      map[string]interface{} `json:"data,omitempty"`
	EventTimestamp int64                  `json:"timestamp"`
}
