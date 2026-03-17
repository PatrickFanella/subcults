// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
)

// RSVPRequest represents the request body for creating/updating an RSVP.
type RSVPRequest struct {
	Status string `json:"status"` // "going" or "maybe"
}

// RSVPResponse represents the response body for RSVP operations.
// Note: UserID is intentionally omitted to protect user privacy.
type RSVPResponse struct {
	EventID   string     `json:"event_id"`
	Status    string     `json:"status"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// RSVPHandlers holds dependencies for RSVP HTTP handlers.
type RSVPHandlers struct {
	rsvpRepo  scene.RSVPRepository
	eventRepo scene.EventRepository
}

// NewRSVPHandlers creates a new RSVPHandlers instance.
func NewRSVPHandlers(rsvpRepo scene.RSVPRepository, eventRepo scene.EventRepository) *RSVPHandlers {
	return &RSVPHandlers{
		rsvpRepo:  rsvpRepo,
		eventRepo: eventRepo,
	}
}

// CreateOrUpdateRSVP handles POST /events/{id}/rsvp - creates or updates an RSVP.
func (h *RSVPHandlers) CreateOrUpdateRSVP(w http.ResponseWriter, r *http.Request) {
	// Extract event ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/events/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Event ID is required")
		return
	}
	eventID := pathParts[0]

	// Parse request body
	var req RSVPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate status
	status := strings.TrimSpace(req.Status)
	if status != "going" && status != "maybe" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "status must be 'going' or 'maybe'")
		return
	}

	// Get user DID from context (set by auth middleware)
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Verify event exists and is upcoming
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

	// Validate event is strictly upcoming (starts_at > now)
	// Business rule: RSVPs are only allowed for events that haven't started yet
	now := time.Now()
	if !existingEvent.StartsAt.After(now) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Cannot RSVP to past or ongoing events")
		return
	}

	// Create or update RSVP
	rsvp := &scene.RSVP{
		EventID: eventID,
		UserID:  userDID,
		Status:  status,
	}

	if err := h.rsvpRepo.Upsert(rsvp); err != nil {
		slog.ErrorContext(r.Context(), "failed to upsert RSVP", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to save RSVP")
		return
	}

	// Retrieve the stored RSVP to get timestamps
	stored, err := h.rsvpRepo.GetByEventAndUser(eventID, userDID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve RSVP", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve RSVP")
		return
	}

	// Create response without exposing user_id (privacy requirement)
	response := RSVPResponse{
		EventID:   stored.EventID,
		Status:    stored.Status,
		CreatedAt: stored.CreatedAt,
		UpdatedAt: stored.UpdatedAt,
	}

	// Return created/updated RSVP
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error but response already started
		slog.ErrorContext(r.Context(), "failed to encode RSVP response", "error", err)
	}
}

// DeleteRSVP handles DELETE /events/{id}/rsvp - removes an RSVP.
func (h *RSVPHandlers) DeleteRSVP(w http.ResponseWriter, r *http.Request) {
	// Extract event ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/events/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Event ID is required")
		return
	}
	eventID := pathParts[0]

	// Get user DID from context (set by auth middleware)
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Verify event exists and is upcoming
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

	// Validate event is strictly upcoming (starts_at > now)
	// Business rule: RSVP modifications are only allowed for events that haven't started yet
	now := time.Now()
	if !existingEvent.StartsAt.After(now) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "Cannot modify RSVP for past or ongoing events")
		return
	}

	// Delete RSVP
	if err := h.rsvpRepo.Delete(eventID, userDID); err != nil {
		if err == scene.ErrRSVPNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "RSVP not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete RSVP", "error", err, "event_id", eventID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to delete RSVP")
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}
