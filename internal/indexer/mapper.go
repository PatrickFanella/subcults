// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/onnwee/subcults/internal/alliance"
	"github.com/onnwee/subcults/internal/geo"
	"github.com/onnwee/subcults/internal/post"
	"github.com/onnwee/subcults/internal/scene"
)

// Mapper errors
var (
	ErrMissingRequiredField = errors.New("missing required field")
	ErrInvalidFieldValue    = errors.New("invalid field value")
)

// ATProtoSceneRecord represents the AT Protocol scene record structure.
type ATProtoSceneRecord struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Location    *ATProtoLocation       `json:"location,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Visibility  *string                `json:"visibility,omitempty"`
	Palette     *ATProtoPalette        `json:"palette,omitempty"`
	ExtraFields map[string]interface{} `json:"-"` // Capture unmapped fields
}

// ATProtoLocation represents location data in AT Protocol records.
type ATProtoLocation struct {
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	AllowPrecise bool    `json:"allowPrecise"`
}

// ATProtoPalette represents color palette in AT Protocol records.
type ATProtoPalette struct {
	Primary    *string `json:"primary,omitempty"`
	Secondary  *string `json:"secondary,omitempty"`
	Accent     *string `json:"accent,omitempty"`
	Background *string `json:"background,omitempty"`
	Text       *string `json:"text,omitempty"`
}

// ATProtoEventRecord represents the AT Protocol event record structure.
type ATProtoEventRecord struct {
	Name        string           `json:"name"` // Will map to "title" in domain model
	SceneID     string           `json:"sceneId"`
	Description *string          `json:"description,omitempty"`
	Location    *ATProtoLocation `json:"location,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Status      *string          `json:"status,omitempty"`
	StartsAt    string           `json:"startsAt"` // ISO 8601 timestamp
	EndsAt      *string          `json:"endsAt,omitempty"`
}

// ATProtoPostRecord represents the AT Protocol post record structure.
type ATProtoPostRecord struct {
	Text        string              `json:"text"`
	SceneID     *string             `json:"sceneId,omitempty"`
	EventID     *string             `json:"eventId,omitempty"`
	Attachments []ATProtoAttachment `json:"attachments,omitempty"`
	Labels      []string            `json:"labels,omitempty"`
}

// ATProtoAttachment represents an attachment in AT Protocol post records.
type ATProtoAttachment struct {
	URL             *string  `json:"url,omitempty"`
	Key             *string  `json:"key,omitempty"`
	Type            *string  `json:"type,omitempty"`
	SizeBytes       *int64   `json:"sizeBytes,omitempty"`
	Width           *int     `json:"width,omitempty"`
	Height          *int     `json:"height,omitempty"`
	DurationSeconds *float64 `json:"durationSeconds,omitempty"`
}

// ATProtoAllianceRecord represents the AT Protocol alliance record structure.
type ATProtoAllianceRecord struct {
	FromSceneID string   `json:"fromSceneId"`
	ToSceneID   string   `json:"toSceneId"`
	Weight      *float64 `json:"weight,omitempty"`
	Status      *string  `json:"status,omitempty"`
	Reason      *string  `json:"reason,omitempty"`
	Since       *string  `json:"since,omitempty"` // ISO 8601 timestamp
}

// MapSceneRecord converts an AT Protocol scene record to a domain Scene model.
// Returns a Scene with record tracking fields populated from the FilterResult.
func MapSceneRecord(record *FilterResult) (*scene.Scene, error) {
	if record == nil || len(record.Record) == 0 {
		return nil, ErrMissingRequiredField
	}

	var atProtoScene ATProtoSceneRecord
	if err := json.Unmarshal(record.Record, &atProtoScene); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scene record: %w", err)
	}

	// Validate required fields
	if atProtoScene.Name == "" {
		return nil, fmt.Errorf("%w: name", ErrMissingRequiredField)
	}

	// Build domain model
	domainScene := &scene.Scene{
		Name:        atProtoScene.Name,
		Description: stringPtrValue(atProtoScene.Description),
		OwnerDID:    record.DID,
		RecordDID:   &record.DID,
		RecordRKey:  &record.RKey,
	}

	// Map tags
	if len(atProtoScene.Tags) > 0 {
		domainScene.Tags = atProtoScene.Tags
	}

	// Map visibility
	if atProtoScene.Visibility != nil {
		domainScene.Visibility = *atProtoScene.Visibility
	} else {
		domainScene.Visibility = scene.VisibilityPublic // Default
	}

	// Map location if present
	if atProtoScene.Location != nil {
		domainScene.AllowPrecise = atProtoScene.Location.AllowPrecise
		domainScene.PrecisePoint = &scene.Point{
			Lat: atProtoScene.Location.Lat,
			Lng: atProtoScene.Location.Lng,
		}

		// Generate coarse geohash for location-based search
		// Use precision 6 for coarse geohash (~1.2km x 0.6km)
		domainScene.CoarseGeohash = geo.Encode(
			atProtoScene.Location.Lat,
			atProtoScene.Location.Lng,
			6,
		)

		// Enforce location consent - clears PrecisePoint if not allowed
		domainScene.EnforceLocationConsent()
	} else {
		// No location provided - use default geohash
		domainScene.CoarseGeohash = "u4pruyd" // Default to Seattle area
		domainScene.AllowPrecise = false
	}

	// Map palette
	if atProtoScene.Palette != nil {
		domainScene.Palette = &scene.Palette{
			Primary:    stringPtrValue(atProtoScene.Palette.Primary),
			Secondary:  stringPtrValue(atProtoScene.Palette.Secondary),
			Accent:     stringPtrValue(atProtoScene.Palette.Accent),
			Background: stringPtrValue(atProtoScene.Palette.Background),
			Text:       stringPtrValue(atProtoScene.Palette.Text),
		}
	}

	return domainScene, nil
}

// MapEventRecord converts an AT Protocol event record to a domain Event model.
// Returns an Event with record tracking fields populated from the FilterResult.
// Note: This does NOT populate scene_id (UUID) - caller must look it up.
func MapEventRecord(record *FilterResult) (*scene.Event, error) {
	if record == nil || len(record.Record) == 0 {
		return nil, ErrMissingRequiredField
	}

	var atProtoEvent ATProtoEventRecord
	if err := json.Unmarshal(record.Record, &atProtoEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event record: %w", err)
	}

	// Validate required fields
	if atProtoEvent.Name == "" {
		return nil, fmt.Errorf("%w: name", ErrMissingRequiredField)
	}
	if atProtoEvent.SceneID == "" {
		return nil, fmt.Errorf("%w: sceneId", ErrMissingRequiredField)
	}
	if atProtoEvent.StartsAt == "" {
		return nil, fmt.Errorf("%w: startsAt", ErrMissingRequiredField)
	}

	// Parse timestamps
	startsAt, err := time.Parse(time.RFC3339, atProtoEvent.StartsAt)
	if err != nil {
		return nil, fmt.Errorf("invalid startsAt timestamp: %w", err)
	}

	var endsAt *time.Time
	if atProtoEvent.EndsAt != nil {
		parsed, err := time.Parse(time.RFC3339, *atProtoEvent.EndsAt)
		if err != nil {
			return nil, fmt.Errorf("invalid endsAt timestamp: %w", err)
		}
		endsAt = &parsed
	}

	// Build domain model
	domainEvent := &scene.Event{
		// SceneID will be populated by caller after scene lookup
		Title:       atProtoEvent.Name,
		Description: stringPtrValue(atProtoEvent.Description),
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		RecordDID:   &record.DID,
		RecordRKey:  &record.RKey,
	}

	// Map tags
	if len(atProtoEvent.Tags) > 0 {
		domainEvent.Tags = atProtoEvent.Tags
	}

	// Map status
	if atProtoEvent.Status != nil {
		domainEvent.Status = *atProtoEvent.Status
	} else {
		domainEvent.Status = "scheduled" // Default
	}

	// Map location if present
	if atProtoEvent.Location != nil {
		domainEvent.AllowPrecise = atProtoEvent.Location.AllowPrecise
		domainEvent.PrecisePoint = &scene.Point{
			Lat: atProtoEvent.Location.Lat,
			Lng: atProtoEvent.Location.Lng,
		}

		// Generate coarse geohash for location-based search
		domainEvent.CoarseGeohash = geo.Encode(
			atProtoEvent.Location.Lat,
			atProtoEvent.Location.Lng,
			6,
		)

		// Enforce location consent
		domainEvent.EnforceLocationConsent()
	} else {
		// No location provided - use default geohash
		domainEvent.CoarseGeohash = "u4pruyd" // Default to Seattle area
		domainEvent.AllowPrecise = false
	}

	return domainEvent, nil
}

// MapPostRecord converts an AT Protocol post record to a domain Post model.
// Returns a Post with record tracking fields populated from the FilterResult.
// Note: This does NOT populate scene_id/event_id (UUIDs) - caller must look them up.
func MapPostRecord(record *FilterResult) (*post.Post, error) {
	if record == nil || len(record.Record) == 0 {
		return nil, ErrMissingRequiredField
	}

	var atProtoPost ATProtoPostRecord
	if err := json.Unmarshal(record.Record, &atProtoPost); err != nil {
		return nil, fmt.Errorf("failed to unmarshal post record: %w", err)
	}

	// Validate required fields
	if atProtoPost.Text == "" {
		return nil, fmt.Errorf("%w: text", ErrMissingRequiredField)
	}

	// At least one of sceneId or eventId must be present
	if atProtoPost.SceneID == nil && atProtoPost.EventID == nil {
		return nil, fmt.Errorf("%w: sceneId or eventId required", ErrMissingRequiredField)
	}

	// Build domain model
	now := time.Now()
	domainPost := &post.Post{
		// SceneID and/or EventID will be populated by caller after lookup
		AuthorDID:  record.DID,
		Text:       atProtoPost.Text,
		RecordDID:  &record.DID,
		RecordRKey: &record.RKey,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Map attachments
	if len(atProtoPost.Attachments) > 0 {
		domainPost.Attachments = make([]post.Attachment, len(atProtoPost.Attachments))
		for i, att := range atProtoPost.Attachments {
			domainPost.Attachments[i] = post.Attachment{
				URL:             stringPtrValue(att.URL),
				Key:             stringPtrValue(att.Key),
				Type:            stringPtrValue(att.Type),
				SizeBytes:       int64PtrValue(att.SizeBytes),
				Width:           att.Width,
				Height:          att.Height,
				DurationSeconds: att.DurationSeconds,
			}
		}
	}

	// Map labels
	if len(atProtoPost.Labels) > 0 {
		domainPost.Labels = atProtoPost.Labels
	}

	return domainPost, nil
}

// MapAllianceRecord converts an AT Protocol alliance record to a domain Alliance model.
// Returns an Alliance with record tracking fields populated from the FilterResult.
// Note: This does NOT populate from_scene_id/to_scene_id (UUIDs) - caller must look them up.
func MapAllianceRecord(record *FilterResult) (*alliance.Alliance, error) {
	if record == nil || len(record.Record) == 0 {
		return nil, ErrMissingRequiredField
	}

	var atProtoAlliance ATProtoAllianceRecord
	if err := json.Unmarshal(record.Record, &atProtoAlliance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alliance record: %w", err)
	}

	// Validate required fields
	if atProtoAlliance.FromSceneID == "" {
		return nil, fmt.Errorf("%w: fromSceneId", ErrMissingRequiredField)
	}
	if atProtoAlliance.ToSceneID == "" {
		return nil, fmt.Errorf("%w: toSceneId", ErrMissingRequiredField)
	}

	// Parse since timestamp if provided
	now := time.Now()
	since := now
	if atProtoAlliance.Since != nil {
		parsed, err := time.Parse(time.RFC3339, *atProtoAlliance.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid since timestamp: %w", err)
		}
		since = parsed
	}

	// Build domain model
	domainAlliance := &alliance.Alliance{
		// FromSceneID and ToSceneID will be populated by caller after lookup
		Weight:     float64PtrValue(atProtoAlliance.Weight, 1.0),     // Default weight is 1.0
		Status:     stringPtrValue(atProtoAlliance.Status, "active"), // Default status
		Reason:     atProtoAlliance.Reason,
		RecordDID:  &record.DID,
		RecordRKey: &record.RKey,
		Since:      since,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return domainAlliance, nil
}

// Helper functions for pointer conversions

func stringPtrValue(ptr *string, defaultVal ...string) string {
	if ptr != nil {
		return *ptr
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func int64PtrValue(ptr *int64) int64 {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func float64PtrValue(ptr *float64, defaultVal float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}
