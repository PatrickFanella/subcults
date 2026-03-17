// Package audit provides audit logging functionality for tracking access to
// sensitive endpoints and operations for compliance and incident response.
package audit

import (
	"time"
)

// AuditLog represents a single audit event in the system.
type AuditLog struct {
	ID         string
	UserDID    string
	EntityType string
	EntityID   string
	Action     string
	Outcome    string // "success" or "failure"
	CreatedAt  time.Time

	// Optional metadata
	RequestID string
	IPAddress string
	UserAgent string

	// Tamper detection
	PreviousHash string // SHA-256 hash of previous log entry for tamper detection
}

// LogEntry represents the input for creating an audit log entry.
type LogEntry struct {
	UserDID    string
	EntityType string
	EntityID   string
	Action     string
	Outcome    string // "success" or "failure"

	// Optional metadata
	RequestID string
	IPAddress string
	UserAgent string
}
