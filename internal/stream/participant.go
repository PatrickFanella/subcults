// Package stream provides participant models for tracking stream session participants.
package stream

import (
	"strings"
	"time"
)

// Participant represents a participant in a stream session.
// Tracks join/leave times and handles reconnection scenarios.
type Participant struct {
	ID                string     `json:"id"`
	StreamSessionID   string     `json:"stream_session_id"`
	ParticipantID     string     `json:"participant_id"` // LiveKit participant identity
	UserDID           string     `json:"user_did"`       // Decentralized Identifier
	JoinedAt          time.Time  `json:"joined_at"`
	LeftAt            *time.Time `json:"left_at,omitempty"`  // NULL while active
	ReconnectionCount int        `json:"reconnection_count"` // Times rejoined after leaving
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// IsActive returns true if the participant is currently active (has not left).
func (p *Participant) IsActive() bool {
	return p.LeftAt == nil
}

// ParticipantStateEvent represents a real-time event for WebSocket broadcasting.
type ParticipantStateEvent struct {
	Type            string    `json:"type"` // "participant_joined" or "participant_left"
	StreamSessionID string    `json:"stream_session_id"`
	ParticipantID   string    `json:"participant_id"`
	UserDID         string    `json:"user_did"`
	Timestamp       time.Time `json:"timestamp"`
	IsReconnection  bool      `json:"is_reconnection"` // True if participant is rejoining
	ActiveCount     int       `json:"active_count"`    // Current active participant count
}

// GenerateParticipantID creates a deterministic participant identity from a user DID.
// Format: user-{stable-identifier}
//
// Note: This generates a stable identity based on the user's DID, ensuring that
// the same user always gets the same participant ID. This is important for LiveKit's
// room management and participant tracking, enabling features like:
// - Reconnection to the same session after temporary disconnection
// - Consistent participant tracking across multiple join attempts
// - Proper cleanup of previous sessions when rejoining
//
// The identifier is extracted from the DID and truncated if needed to maintain reasonable length.
func GenerateParticipantID(did string) string {
	// DIDs have format: did:method:identifier (e.g., did:plc:abc123...)
	// We'll use the identifier part to create a stable ID

	// Split on colons and take the last part (the identifier)
	parts := strings.Split(did, ":")
	var identifier string

	if len(parts) >= 3 {
		// Use the identifier portion (last part)
		identifier = parts[len(parts)-1]
	} else {
		// Fallback: if DID format is unexpected, use the whole DID
		identifier = did
	}

	// Extract the identifier part and truncate to reasonable length (max 48 chars to keep total under 64)
	const maxLen = 48
	if len(identifier) > maxLen {
		identifier = identifier[:maxLen]
	}

	return "user-" + identifier
}
