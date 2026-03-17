package audit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/onnwee/subcults/internal/middleware"
)

// Outcome constants for audit logs.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

var (
	// ErrNilRepository is returned when a nil repository is passed to logging functions.
	ErrNilRepository = errors.New("audit repository cannot be nil")
	// ErrInvalidEntityType is returned when an invalid entity type is provided.
	ErrInvalidEntityType = errors.New("entity type cannot be empty")
	// ErrInvalidEntityID is returned when an invalid entity ID is provided.
	ErrInvalidEntityID = errors.New("entity ID cannot be empty")
	// ErrInvalidAction is returned when an invalid action is provided.
	ErrInvalidAction = errors.New("action cannot be empty")
	// ErrInvalidOutcome is returned when an invalid outcome is provided.
	ErrInvalidOutcome = errors.New("outcome must be 'success' or 'failure'")
)

// ValidEntityTypes defines the allowed entity types for audit logging.
var ValidEntityTypes = map[string]bool{
	"scene":       true,
	"event":       true,
	"user":        true,
	"admin_panel": true,
	"post":        true,
	"membership":  true,
	"payment":     true,
	"stream":      true,
	"admin":       true,
}

// ValidActions defines the allowed actions for audit logging.
var ValidActions = map[string]bool{
	// Location access
	"access_precise_location": true,
	"access_coarse_location":  true,

	// Admin operations
	"view_admin_panel":        true,
	"view_privacy_settings":   true,
	"modify_privacy_settings": true,
	"admin_login":             true,
	"admin_action":            true,

	// Scene operations
	"view_scene_details": true,
	"scene_create":       true,
	"scene_update":       true,
	"scene_delete":       true,

	// Event operations
	"view_event_details": true,
	"event_create":       true,
	"event_update":       true,
	"event_delete":       true,
	"event_cancel":       true,

	// Membership operations
	"export_member_data": true,
	"membership_request": true,
	"membership_approve": true,
	"membership_reject":  true,

	// User authentication
	"user_login":  true,
	"user_logout": true,

	// Payment operations
	"payment_create":  true,
	"payment_success": true,
	"payment_failure": true,

	// Stream/Organizer operations
	"stream_start":      true,
	"stream_end":        true,
	"participant_mute":  true,
	"participant_kick":  true,
	"participant_unmute": true,
}

// validateLogEntry validates the required fields of a log entry against whitelists.
func validateLogEntry(entityType, entityID, action, outcome string) error {
	if entityType == "" {
		return ErrInvalidEntityType
	}
	if entityID == "" {
		return ErrInvalidEntityID
	}
	if action == "" {
		return ErrInvalidAction
	}

	// Validate outcome if provided (empty is allowed for backward compatibility)
	if outcome != "" && outcome != OutcomeSuccess && outcome != OutcomeFailure {
		return ErrInvalidOutcome
	}

	// Validate against whitelists if the values are not in the allowed sets
	if !ValidEntityTypes[entityType] {
		return ErrInvalidEntityType
	}
	if !ValidActions[action] {
		return ErrInvalidAction
	}

	return nil
}

// extractIPAddress extracts the client IP address from an HTTP request.
// It checks X-Forwarded-For, X-Real-IP, and RemoteAddr in that order.
// The port is stripped from the IP address to ensure compatibility with database storage.
func extractIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Use the first IP in the chain, trimming whitespace per RFC 7239
		var firstIP string
		if idx := strings.Index(xff, ","); idx != -1 {
			firstIP = strings.TrimSpace(xff[:idx])
		} else {
			firstIP = strings.TrimSpace(xff)
		}
		// Only use if non-empty after trimming, and strip port if present
		if firstIP != "" {
			host, _, err := net.SplitHostPort(firstIP)
			if err != nil {
				// IP might not have a port
				return firstIP
			}
			return host
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		xri = strings.TrimSpace(xri)
		// Strip port if present
		host, _, err := net.SplitHostPort(xri)
		if err != nil {
			// IP might not have a port
			return xri
		}
		return host
	}

	// Fall back to RemoteAddr (strip port properly for both IPv4 and IPv6)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return host
}

// LogAccess is a helper function that records an access event to the audit log.
// It extracts user DID and request ID from the context if available.
// entityType: Type of entity accessed (e.g., "scene", "event", "admin_panel")
// entityID: ID of the entity accessed
// action: Action performed (e.g., "access_precise_location", "view_admin_panel")
// outcome: "success" or "failure" (defaults to "success" if empty)
//
// Error handling: This function uses a fail-closed approach - if audit logging fails,
// the error is returned to the caller. This ensures compliance requirements are met
// but may impact availability if the audit system is down.
func LogAccess(ctx context.Context, repo Repository, entityType, entityID, action, outcome string) error {
	if repo == nil {
		return ErrNilRepository
	}

	// Default outcome to success if not provided
	if outcome == "" {
		outcome = OutcomeSuccess
	}

	if err := validateLogEntry(entityType, entityID, action, outcome); err != nil {
		return err
	}

	entry := LogEntry{
		UserDID:    middleware.GetUserDID(ctx),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Outcome:    outcome,
		RequestID:  middleware.GetRequestID(ctx),
	}

	_, err := repo.LogAccess(entry)
	return err
}

// LogAccessFromRequest is a helper function that records an access event with HTTP request metadata.
// It extracts user DID, request ID, IP address, and user agent from the request/context.
//
// IP address extraction:
// - Checks X-Forwarded-For header first (uses first IP from comma-separated list)
// - Falls back to X-Real-IP header
// - Finally uses RemoteAddr (with port stripped)
//
// outcome: "success" or "failure" (defaults to "success" if empty)
//
// Error handling: This function uses a fail-closed approach - if audit logging fails,
// the error is returned to the caller. This ensures compliance requirements are met
// but may impact availability if the audit system is down.
func LogAccessFromRequest(r *http.Request, repo Repository, entityType, entityID, action, outcome string) error {
	if repo == nil {
		return ErrNilRepository
	}

	// Default outcome to success if not provided
	if outcome == "" {
		outcome = OutcomeSuccess
	}

	if err := validateLogEntry(entityType, entityID, action, outcome); err != nil {
		return err
	}

	entry := LogEntry{
		UserDID:    middleware.GetUserDID(r.Context()),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Outcome:    outcome,
		RequestID:  middleware.GetRequestID(r.Context()),
		IPAddress:  extractIPAddress(r),
		UserAgent:  r.UserAgent(),
	}

	_, err := repo.LogAccess(entry)
	return err
}
