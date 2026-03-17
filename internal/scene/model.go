// Package scene provides models and repository for managing scenes and events
// with location privacy controls.
package scene

import "time"

// Visibility modes for scenes
const (
	VisibilityPublic      = "public"   // Visible to all users and appears in search
	VisibilityMembersOnly = "private"  // Visible only to active members and owner (DB uses "private")
	VisibilityHidden      = "unlisted" // Visible only to owner, exempt from search (DB uses "unlisted")
)

// Point represents a geographic coordinate with latitude and longitude.
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Palette represents the color scheme for a scene's visual identity.
// All colors should be hex codes in format #RRGGBB.
type Palette struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Background string `json:"background"`
	Text       string `json:"text"`
}

// Scene represents a subcultural scene with optional precise location data.
// The precise_point field is only persisted when allow_precise consent is true.
type Scene struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	OwnerDID     string `json:"owner_did"` // Decentralized Identifier
	AllowPrecise bool   `json:"allow_precise"`
	PrecisePoint *Point `json:"precise_point,omitempty"`
	// CoarseGeohash is a required NOT NULL field for privacy-conscious discovery.
	// Must be set before persisting to database. Enables location-based search without
	// exposing precise coordinates; omitting this field will cause database errors.
	CoarseGeohash string   `json:"coarse_geohash"`
	Tags          []string `json:"tags,omitempty"` // Categorization tags
	// Visibility mode for the scene. Valid values are "public", "private", or "unlisted".
	// Enforced by database CHECK constraint.
	Visibility  string   `json:"visibility,omitempty"`
	Palette     *Palette `json:"palette,omitempty"`       // Color scheme
	OwnerUserID *string  `json:"owner_user_id,omitempty"` // FK to users table

	// Payments
	ConnectedAccountID     *string `json:"connected_account_id,omitempty"`      // Stripe Connect Express account ID
	ConnectedAccountStatus string  `json:"connected_account_status,omitempty"`   // pending, active, or restricted
	AccountOnboardedAt     *time.Time `json:"account_onboarded_at,omitempty"`    // When Stripe account was fully onboarded

	// Moderation (admin visibility controls)
	ModerationStatus  string     `json:"moderation_status,omitempty"`  // visible, hidden, flagged, or suspended
	ModerationReason  *string    `json:"moderation_reason,omitempty"`  // Reason for moderation action
	ModeratedBy       *string    `json:"moderated_by,omitempty"`       // User DID of moderator
	ModerationTimestamp *time.Time `json:"moderation_timestamp,omitempty"` // When moderation action was taken

	// Timestamps
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// AT Protocol record tracking
	RecordDID  *string `json:"record_did,omitempty"`
	RecordRKey *string `json:"record_rkey,omitempty"`
}

// Event represents an event within a scene with optional precise location data.
// The precise_point field is only persisted when allow_precise consent is true.
type Event struct {
	ID            string     `json:"id"`
	SceneID       string     `json:"scene_id"`
	Title         string     `json:"title"` // Event title (renamed from Name per migration)
	Description   string     `json:"description,omitempty"`
	AllowPrecise  bool       `json:"allow_precise"`
	PrecisePoint  *Point     `json:"precise_point,omitempty"`
	CoarseGeohash string     `json:"coarse_geohash"` // Required for location-based discovery
	Tags          []string   `json:"tags,omitempty"`
	Status        string     `json:"status,omitempty"` // scheduled, live, ended, cancelled
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        *time.Time `json:"ends_at,omitempty"`

	// Timestamps
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	// Cancellation details
	CancellationReason *string `json:"cancellation_reason,omitempty"`

	// AT Protocol record tracking
	RecordDID  *string `json:"record_did,omitempty"`
	RecordRKey *string `json:"record_rkey,omitempty"`

	// LiveKit streaming
	StreamSessionID *string `json:"stream_session_id,omitempty"`
}

// EnforceLocationConsent clears PrecisePoint if AllowPrecise is false.
// This ensures that precise location data is never stored without consent.
// Returns the scene for chaining.
func (s *Scene) EnforceLocationConsent() *Scene {
	if !s.AllowPrecise {
		s.PrecisePoint = nil
	}
	return s
}

// EnforceLocationConsent clears PrecisePoint if AllowPrecise is false.
// This ensures that precise location data is never stored without consent.
// Returns the event for chaining.
func (e *Event) EnforceLocationConsent() *Event {
	if !e.AllowPrecise {
		e.PrecisePoint = nil
	}
	return e
}

// IsOwner checks if the given DID is the owner of the scene.
func (s *Scene) IsOwner(userDID string) bool {
	return s.OwnerDID == userDID
}

// RSVP represents a user's attendance intent for an event.
type RSVP struct {
	EventID string `json:"event_id"`
	// UserID stores the user's DID (Decentralized Identifier), not a UUID or FK to a users table.
	// This allows guest RSVPs and aligns with the database schema (see migration 000012 comment).
	UserID    string     `json:"user_id"`
	Status    string     `json:"status"` // "going" or "maybe"
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// RSVPCounts represents aggregated RSVP counts by status.
type RSVPCounts struct {
	Going int `json:"going"`
	Maybe int `json:"maybe"`
}
