// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/stream"
	"github.com/onnwee/subcults/internal/validate"
)

// CreateEventRequest represents the request body for creating an event.
type CreateEventRequest struct {
	SceneID       string       `json:"scene_id"`
	Title         string       `json:"title"`
	Description   string       `json:"description,omitempty"`
	AllowPrecise  bool         `json:"allow_precise"`
	PrecisePoint  *scene.Point `json:"precise_point,omitempty"`
	CoarseGeohash string       `json:"coarse_geohash"`
	Tags          []string     `json:"tags,omitempty"`
	StartsAt      time.Time    `json:"starts_at"`
	EndsAt        *time.Time   `json:"ends_at,omitempty"`
}

// UpdateEventRequest represents the request body for updating an event.
// Only includes mutable fields (scene_id is immutable).
type UpdateEventRequest struct {
	Title         *string      `json:"title,omitempty"`
	Description   *string      `json:"description,omitempty"`
	Tags          []string     `json:"tags,omitempty"`
	AllowPrecise  *bool        `json:"allow_precise,omitempty"`
	PrecisePoint  *scene.Point `json:"precise_point,omitempty"`
	CoarseGeohash *string      `json:"coarse_geohash,omitempty"`
	StartsAt      *time.Time   `json:"starts_at,omitempty"`
	EndsAt        *time.Time   `json:"ends_at,omitempty"`
}

// CancelEventRequest represents the request body for cancelling an event.
type CancelEventRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// EventHandlers holds dependencies for event HTTP handlers.
type EventHandlers struct {
	eventRepo       scene.EventRepository
	sceneRepo       scene.SceneRepository
	auditRepo       audit.Repository
	rsvpRepo        scene.RSVPRepository
	streamRepo      stream.SessionRepository
	trustScoreStore TrustScoreStore // Optional, can be nil
}

// TrustScoreStore defines the interface for retrieving trust scores.
// This avoids importing the trust package directly.
type TrustScoreStore interface {
	GetScore(sceneID string) (score *TrustScore, err error)
}

// TrustScore represents a trust score value.
type TrustScore struct {
	SceneID string
	Score   float64
}

// NewEventHandlers creates a new EventHandlers instance.
// trustScoreStore is optional and can be nil if trust ranking is not used.
func NewEventHandlers(eventRepo scene.EventRepository, sceneRepo scene.SceneRepository, auditRepo audit.Repository, rsvpRepo scene.RSVPRepository, streamRepo stream.SessionRepository, trustScoreStore TrustScoreStore) *EventHandlers {
	return &EventHandlers{
		eventRepo:       eventRepo,
		sceneRepo:       sceneRepo,
		auditRepo:       auditRepo,
		rsvpRepo:        rsvpRepo,
		streamRepo:      streamRepo,
		trustScoreStore: trustScoreStore,
	}
}

// EventWithRSVPCounts represents an event with aggregated RSVP counts and active stream info.
type EventWithRSVPCounts struct {
	*scene.Event
	RSVPCounts   *scene.RSVPCounts        `json:"rsvp_counts"`
	Scene        *SceneSearchResult       `json:"scene,omitempty"`
	ActiveStream *stream.ActiveStreamInfo `json:"active_stream,omitempty"`
}

// sceneBatchFetcher is an optional repository capability for batch scene lookups.
// SearchEvents uses this when available to reduce per-scene fetches in responses.
type sceneBatchFetcher interface {
	GetByIDs(ids []string) ([]*scene.Scene, error)
}

// toSceneSearchResult converts an internal scene model to a public search-safe
// scene payload with jittered coordinates for privacy.
func toSceneSearchResult(parentScene *scene.Scene) *SceneSearchResult {
	if parentScene == nil {
		return nil
	}
	result := &SceneSearchResult{
		ID:            parentScene.ID,
		Name:          parentScene.Name,
		Description:   parentScene.Description,
		CoarseGeohash: parentScene.CoarseGeohash,
		Tags:          parentScene.Tags,
		Visibility:    parentScene.Visibility,
	}
	if parentScene.PrecisePoint != nil {
		result.JitteredPoint = applyJitter(parentScene.PrecisePoint)
	}
	return result
}

// validateTimeWindow validates that start time is before end time.
// Returns error message if validation fails, empty string if valid.
func validateTimeWindow(startsAt time.Time, endsAt *time.Time) string {
	if endsAt != nil && !startsAt.Before(*endsAt) {
		return "start time must be before end time"
	}
	return ""
}

// isSceneOwner checks if the given userDID owns the scene.
func (h *EventHandlers) isSceneOwner(ctx context.Context, sceneID, userDID string) (bool, error) {
	foundScene, err := h.sceneRepo.GetByID(sceneID)
	if err != nil {
		return false, err
	}
	return foundScene.IsOwner(userDID), nil
}

// CreateEvent handles POST /events - creates a new event.
func (h *EventHandlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate and sanitize title
	validatedTitle, err := validate.EventTitle(req.Title)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid event title: %v", err))
		return
	}
	req.Title = validatedTitle

	// Validate scene_id
	if strings.TrimSpace(req.SceneID) == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "scene_id is required")
		return
	}

	// Validate coarse_geohash
	if strings.TrimSpace(req.CoarseGeohash) == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "coarse_geohash is required")
		return
	}

	// Validate time window
	if errMsg := validateTimeWindow(req.StartsAt, req.EndsAt); errMsg != "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidTimeRange)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidTimeRange, errMsg)
		return
	}

	// Get user DID from context (set by auth middleware)
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Check if user is scene owner (authorization)
	isOwner, err := h.isSceneOwner(r.Context(), req.SceneID, userDID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to check scene ownership", "error", err, "scene_id", req.SceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to verify scene ownership")
		return
	}
	if !isOwner {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "You do not have permission to create events for this scene")
		return
	}

	// Validate and sanitize description
	validatedDesc, err := validate.Description(req.Description)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid description: %v", err))
		return
	}
	req.Description = validatedDesc

	// Sanitize tags to prevent HTML injection
	sanitizedTags := make([]string, len(req.Tags))
	for i, tag := range req.Tags {
		sanitizedTags[i] = validate.SanitizeHTML(tag)
	}

	// Create event
	now := time.Now()
	newEvent := &scene.Event{
		ID:            uuid.New().String(),
		SceneID:       req.SceneID,
		Title:         req.Title,
		Description:   req.Description,
		AllowPrecise:  req.AllowPrecise,
		PrecisePoint:  req.PrecisePoint,
		CoarseGeohash: req.CoarseGeohash,
		Tags:          sanitizedTags,
		Status:        "scheduled", // Default status
		StartsAt:      req.StartsAt,
		EndsAt:        req.EndsAt,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// Insert into repository (will automatically enforce location consent).
	// If AllowPrecise is false, PrecisePoint will be cleared before storage.
	if err := h.eventRepo.Insert(newEvent); err != nil {
		slog.ErrorContext(r.Context(), "failed to insert event", "error", err, "event_id", newEvent.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to create event")
		return
	}

	// Retrieve the stored event to get privacy-enforced version
	stored, err := h.eventRepo.GetByID(newEvent.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve created event", "error", err, "event_id", newEvent.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve created event")
		return
	}

	// Return created event
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(stored); err != nil {
		// Log error but response already started
		slog.ErrorContext(r.Context(), "failed to encode event response", "error", err)
	}
}

// UpdateEvent handles PATCH /events/{id} - updates an existing event.
func (h *EventHandlers) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	// Extract event ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/events/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Event ID is required")
		return
	}
	eventID := pathParts[0]

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Get existing event
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		if err == scene.ErrEventNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Event not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get event", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve event")
		return
	}

	// Get user DID from context (set by auth middleware)
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Check if user is scene owner (authorization)
	isOwner, err := h.isSceneOwner(r.Context(), existingEvent.SceneID, userDID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to check scene ownership", "error", err, "scene_id", existingEvent.SceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to verify scene ownership")
		return
	}
	if !isOwner {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "You do not have permission to update this event")
		return
	}

	// Apply updates to existing event
	updatedEvent := *existingEvent

	if req.Title != nil {
		// Validate and sanitize title
		validatedTitle, err := validate.EventTitle(*req.Title)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid event title: %v", err))
			return
		}
		updatedEvent.Title = validatedTitle
	}

	if req.Description != nil {
		validatedDesc, err := validate.Description(*req.Description)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid description: %v", err))
			return
		}
		updatedEvent.Description = validatedDesc
	}

	if req.Tags != nil {
		sanitizedTags := make([]string, len(req.Tags))
		for i, tag := range req.Tags {
			sanitizedTags[i] = validate.SanitizeHTML(tag)
		}
		updatedEvent.Tags = sanitizedTags
	}

	if req.AllowPrecise != nil {
		updatedEvent.AllowPrecise = *req.AllowPrecise
	}

	if req.PrecisePoint != nil {
		updatedEvent.PrecisePoint = req.PrecisePoint
	}

	if req.CoarseGeohash != nil {
		if strings.TrimSpace(*req.CoarseGeohash) == "" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "coarse_geohash cannot be empty")
			return
		}
		updatedEvent.CoarseGeohash = *req.CoarseGeohash
	}

	// Handle time updates with validation
	startsAt := updatedEvent.StartsAt
	endsAt := updatedEvent.EndsAt

	if req.StartsAt != nil {
		// Only allow updates if event is still in the future
		if existingEvent.StartsAt.Before(time.Now()) {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Cannot update start time for past events")
			return
		}
		startsAt = *req.StartsAt
	}

	if req.EndsAt != nil {
		endsAt = req.EndsAt
	}

	// Validate time window after applying updates
	if errMsg := validateTimeWindow(startsAt, endsAt); errMsg != "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidTimeRange)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidTimeRange, errMsg)
		return
	}

	updatedEvent.StartsAt = startsAt
	updatedEvent.EndsAt = endsAt

	// Update timestamp
	now := time.Now()
	updatedEvent.UpdatedAt = &now

	// Update in repository (will automatically enforce location consent)
	if err := h.eventRepo.Update(&updatedEvent); err != nil {
		slog.ErrorContext(r.Context(), "failed to update event", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to update event")
		return
	}

	// Retrieve the stored event to get privacy-enforced version
	stored, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve updated event", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve updated event")
		return
	}

	// Return updated event
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(stored); err != nil {
		// Log error but response already started
		slog.ErrorContext(r.Context(), "failed to encode event response", "error", err)
	}
}

// GetEvent handles GET /events/{id} - retrieves an event.
func (h *EventHandlers) GetEvent(w http.ResponseWriter, r *http.Request) {
	// Extract event ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/events/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Event ID is required")
		return
	}
	eventID := pathParts[0]

	// Get the event
	foundEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		if err == scene.ErrEventNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Event not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get event", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve event")
		return
	}

	// Privacy enforcement is handled by the repository
	// The repository automatically enforces location consent via EnforceLocationConsent()

	// Get RSVP counts for the event
	rsvpCounts, err := h.rsvpRepo.GetCountsByEvent(eventID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get RSVP counts", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve RSVP counts")
		return
	}

	// Get active stream for the event
	activeStream, err := h.streamRepo.GetActiveStreamForEvent(eventID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get active stream", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve active stream")
		return
	}

	// Create response with event, RSVP counts, and active stream
	response := EventWithRSVPCounts{
		Event:        foundEvent,
		RSVPCounts:   rsvpCounts,
		ActiveStream: activeStream,
	}

	// Return event with RSVP counts
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error but response already started
		slog.ErrorContext(r.Context(), "failed to encode event response", "error", err)
	}
}

// CancelEvent handles POST /events/{id}/cancel - cancels an event.
func (h *EventHandlers) CancelEvent(w http.ResponseWriter, r *http.Request) {
	// Extract event ID from URL path
	// Note: The routing layer already validates this is a /events/{id}/cancel request
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/events/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Event ID is required")
		return
	}
	eventID := pathParts[0]

	// Parse request body (optional reason)
	var req CancelEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		// Allow empty body (io.EOF) but reject malformed JSON
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Sanitize reason if provided to prevent HTML injection
	if req.Reason != nil {
		sanitized := validate.SanitizeHTML(*req.Reason)
		req.Reason = &sanitized
	}

	// Get existing event
	existingEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		if err == scene.ErrEventNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Event not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get event", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve event")
		return
	}

	// Get user DID from context (set by auth middleware)
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Check if user is scene owner (authorization)
	isOwner, err := h.isSceneOwner(r.Context(), existingEvent.SceneID, userDID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to check scene ownership", "error", err, "scene_id", existingEvent.SceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to verify scene ownership")
		return
	}
	if !isOwner {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "You do not have permission to cancel this event")
		return
	}

	// Track whether event was already cancelled for audit log decision
	alreadyCancelled := existingEvent.Status == "cancelled" && existingEvent.CancelledAt != nil

	// Cancel the event (idempotent)
	if err := h.eventRepo.Cancel(eventID, req.Reason); err != nil {
		slog.ErrorContext(r.Context(), "failed to cancel event", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to cancel event")
		return
	}

	// Emit audit log only if this was the first cancellation (not idempotent case)
	if !alreadyCancelled {
		if err := audit.LogAccessFromRequest(r, h.auditRepo, "event", eventID, "event_cancel", audit.OutcomeSuccess); err != nil {
			slog.ErrorContext(r.Context(), "failed to log event cancellation", "error", err, "event_id", eventID)
			// Don't fail the request, but log the error
		}
	}

	// Retrieve the updated event
	cancelledEvent, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve cancelled event", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve cancelled event")
		return
	}

	// Return cancelled event
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(cancelledEvent); err != nil {
		// Log error but response already started
		slog.ErrorContext(r.Context(), "failed to encode event response", "error", err)
	}
}

// SearchEventsResponse represents the response for event search with active stream info.
type SearchEventsResponse struct {
	Events     []*EventWithRSVPCounts `json:"events"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

// SearchEvents handles GET /search/events - searches events by bbox and time range.
// Supports optional text search (q parameter) and trust-weighted ranking.
func (h *EventHandlers) SearchEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	// Parse bbox (format: minLng,minLat,maxLng,maxLat)
	bboxStr := query.Get("bbox")
	if bboxStr == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "bbox parameter is required")
		return
	}

	bboxParts := strings.Split(bboxStr, ",")
	if len(bboxParts) != 4 {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "bbox must be in format: minLng,minLat,maxLng,maxLat")
		return
	}

	// Parse and validate bbox coordinates
	var minLng, minLat, maxLng, maxLat float64
	var err error

	if minLng, err = parseFloat(bboxParts[0], "minLng"); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	if minLat, err = parseFloat(bboxParts[1], "minLat"); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	if maxLng, err = parseFloat(bboxParts[2], "maxLng"); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	if maxLat, err = parseFloat(bboxParts[3], "maxLat"); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}

	// Validate bbox ranges
	if minLng < -180 || minLng > 180 || maxLng < -180 || maxLng > 180 {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "longitude must be between -180 and 180")
		return
	}
	if minLat < -90 || minLat > 90 || maxLat < -90 || maxLat > 90 {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "latitude must be between -90 and 90")
		return
	}
	if minLng >= maxLng {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "minLng must be less than maxLng")
		return
	}
	if minLat >= maxLat {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "minLat must be less than maxLat")
		return
	}

	// Parse time range (support legacy from/to and start_date/end_date aliases)
	fromStr := strings.TrimSpace(query.Get("from"))
	if fromStr == "" {
		fromStr = strings.TrimSpace(query.Get("start_date"))
	}
	toStr := strings.TrimSpace(query.Get("to"))
	if toStr == "" {
		toStr = strings.TrimSpace(query.Get("end_date"))
	}

	if fromStr == "" || toStr == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "both 'from' and 'to' parameters are required")
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "invalid 'from' timestamp, must be RFC3339 format")
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "invalid 'to' timestamp, must be RFC3339 format")
		return
	}

	// Validate time range
	if !from.Before(to) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidTimeRange)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidTimeRange, "'from' must be before 'to'")
		return
	}

	// Validate max window length (30 days)
	maxWindow := 30 * 24 * time.Hour
	if to.Sub(from) > maxWindow {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "time window cannot exceed 30 days")
		return
	}

	// Parse optional text search query
	searchQuery := query.Get("q")
	statusFilter := strings.ToLower(strings.TrimSpace(query.Get("status")))
	if statusFilter != "" && statusFilter != "upcoming" && statusFilter != "live" && statusFilter != "past" && statusFilter != "cancelled" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "status must be one of: upcoming, live, past, cancelled")
		return
	}
	sceneIDFilter := strings.TrimSpace(query.Get("scene_id"))
	organizerFilter := strings.TrimSpace(query.Get("organizer"))
	organizerSceneIDs := make([]string, 0)
	if organizerFilter != "" {
		organizerScenes, err := h.sceneRepo.ListByOwner(organizerFilter)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to list organizer scenes", "error", err, "organizer", organizerFilter)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
			WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search events")
			return
		}
		for _, organizerScene := range organizerScenes {
			organizerSceneIDs = append(organizerSceneIDs, organizerScene.ID)
		}
		if len(organizerSceneIDs) == 0 {
			response := SearchEventsResponse{
				Events: make([]*EventWithRSVPCounts, 0),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				slog.ErrorContext(r.Context(), "failed to encode search response", "error", err)
			}
			return
		}

		if sceneIDFilter != "" {
			ownsScene := false
			for _, organizerSceneID := range organizerSceneIDs {
				if organizerSceneID == sceneIDFilter {
					ownsScene = true
					break
				}
			}
			if !ownsScene {
				response := SearchEventsResponse{
					Events: make([]*EventWithRSVPCounts, 0),
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(response); err != nil {
					slog.ErrorContext(r.Context(), "failed to encode search response", "error", err)
				}
				return
			}
			organizerSceneIDs = []string{sceneIDFilter}
		}
	}

	// Parse pagination parameters
	limitStr := query.Get("limit")
	limit := 50 // default limit
	if limitStr != "" {
		parsedLimit, err := parseIntInRange(limitStr, "limit", 1, 100)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, err.Error())
			return
		}
		limit = parsedLimit
	}

	cursor := query.Get("cursor")

	// Build trust scores map if trust ranking is enabled and store is available
	var trustScores map[string]float64
	if h.trustScoreStore != nil {
		// For now, we'll fetch trust scores on-demand when we get the events
		// In a production implementation, we might want to batch-fetch or cache these
		trustScores = make(map[string]float64)
	}

	// Search events with new SearchEvents method
	events, nextCursor, err := h.eventRepo.SearchEvents(scene.EventSearchOptions{
		MinLng:      minLng,
		MinLat:      minLat,
		MaxLng:      maxLng,
		MaxLat:      maxLat,
		From:        from,
		To:          to,
		Query:       searchQuery,
		Status:      statusFilter,
		SceneID:     sceneIDFilter,
		SceneIDs:    organizerSceneIDs,
		Limit:       limit,
		Cursor:      cursor,
		TrustScores: trustScores,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to search events", "error", err)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search events")
		return
	}

	// Collect unique scene IDs for trust score fetching
	if h.trustScoreStore != nil && len(events) > 0 {
		sceneIDs := make(map[string]bool)
		for _, event := range events {
			sceneIDs[event.SceneID] = true
		}

		// Fetch trust scores for all scenes
		for sceneID := range sceneIDs {
			score, err := h.trustScoreStore.GetScore(sceneID)
			if err != nil {
				// Log error but don't fail the request
				slog.WarnContext(r.Context(), "failed to get trust score", "scene_id", sceneID, "error", err)
				continue
			}
			if score != nil {
				trustScores[sceneID] = score.Score
			}
		}

		// If we got trust scores, re-run the search with them
		if len(trustScores) > 0 {
			events, nextCursor, err = h.eventRepo.SearchEvents(scene.EventSearchOptions{
				MinLng:      minLng,
				MinLat:      minLat,
				MaxLng:      maxLng,
				MaxLat:      maxLat,
				From:        from,
				To:          to,
				Query:       searchQuery,
				Status:      statusFilter,
				SceneID:     sceneIDFilter,
				SceneIDs:    organizerSceneIDs,
				Limit:       limit,
				Cursor:      cursor,
				TrustScores: trustScores,
			})
			if err != nil {
				slog.ErrorContext(r.Context(), "failed to search events with trust scores", "error", err)
				ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
				WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to search events")
				return
			}
		}
	}

	// Batch fetch active streams to avoid N+1 queries
	eventIDs := make([]string, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}

	activeStreamsMap, err := h.streamRepo.GetActiveStreamsForEvents(eventIDs)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get active streams", "error", err)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve active streams")
		return
	}

	// Batch fetch RSVP counts to avoid N+1 queries
	rsvpCountsMap, err := h.rsvpRepo.GetCountsForEvents(eventIDs)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get RSVP counts", "error", err)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve RSVP counts")
		return
	}

	sceneMap := make(map[string]*SceneSearchResult)
	if len(events) > 0 {
		sceneIDs := make(map[string]struct{}, len(events))
		orderedSceneIDs := make([]string, 0)
		for _, event := range events {
			if _, seen := sceneIDs[event.SceneID]; seen {
				continue
			}
			sceneIDs[event.SceneID] = struct{}{}
			orderedSceneIDs = append(orderedSceneIDs, event.SceneID)
		}

		if batchRepo, ok := h.sceneRepo.(sceneBatchFetcher); ok {
			parentScenes, err := batchRepo.GetByIDs(orderedSceneIDs)
			if err != nil {
				slog.WarnContext(r.Context(), "failed to batch fetch scenes for event search response; falling back to individual fetches", "error", err)
			} else {
				for _, parentScene := range parentScenes {
					sceneMap[parentScene.ID] = toSceneSearchResult(parentScene)
				}
			}
		}

		for _, sceneID := range orderedSceneIDs {
			if _, ok := sceneMap[sceneID]; ok {
				continue
			}
			parentScene, err := h.sceneRepo.GetByID(sceneID)
			if err != nil {
				slog.WarnContext(r.Context(), "failed to fetch scene for event search response", "scene_id", sceneID, "error", err)
				continue
			}
			sceneMap[sceneID] = toSceneSearchResult(parentScene)
		}
	}

	// Build response with events, RSVP counts, and active streams
	eventsWithData := make([]*EventWithRSVPCounts, len(events))
	for i, event := range events {
		eventsWithData[i] = &EventWithRSVPCounts{
			Event:        event,
			RSVPCounts:   rsvpCountsMap[event.ID],
			Scene:        sceneMap[event.SceneID],
			ActiveStream: activeStreamsMap[event.ID], // nil if no active stream
		}
	}

	// Return response
	response := SearchEventsResponse{
		Events:     eventsWithData,
		NextCursor: nextCursor,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode search response", "error", err)
	}
}

// parseFloat parses a float64 from a string with contextual error message.
func parseFloat(s, fieldName string) (float64, error) {
	s = strings.TrimSpace(s)
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid number", fieldName)
	}
	return val, nil
}

// parseIntInRange parses an integer from a string with range validation.
func parseIntInRange(s, fieldName string, min, max int) (int, error) {
	s = strings.TrimSpace(s)
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", fieldName)
	}
	if val < min || val > max {
		return 0, fmt.Errorf("%s must be between %d and %d", fieldName, min, max)
	}
	return val, nil
}
