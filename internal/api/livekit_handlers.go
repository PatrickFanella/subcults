// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/livekit"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/stream"
)

// LiveKitTokenRequest represents the request body for generating a LiveKit token.
type LiveKitTokenRequest struct {
	RoomID  string  `json:"room_id"`            // Required: Room identifier
	SceneID *string `json:"scene_id,omitempty"` // Optional: Associated scene ID
	EventID *string `json:"event_id,omitempty"` // Optional: Associated event ID
}

// LiveKitTokenResponse represents the response for a LiveKit token request.
type LiveKitTokenResponse struct {
	Token     string `json:"token"`      // The JWT access token
	ExpiresAt string `json:"expires_at"` // Token expiration time in RFC3339 format
}

// LiveKitHandlers holds dependencies for LiveKit HTTP handlers.
type LiveKitHandlers struct {
	tokenService *livekit.TokenService
	auditRepo    audit.Repository
}

// NewLiveKitHandlers creates a new LiveKitHandlers instance.
func NewLiveKitHandlers(tokenService *livekit.TokenService, auditRepo audit.Repository) *LiveKitHandlers {
	return &LiveKitHandlers{
		tokenService: tokenService,
		auditRepo:    auditRepo,
	}
}

// Room ID validation: alphanumeric, hyphens, underscores, colons (max 128 chars)
// This prevents injection attacks and restricts to safe characters.
var roomIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_:-]{1,128}$`)

// validateRoomID checks if the room ID matches the allowed pattern.
func validateRoomID(roomID string) bool {
	return roomIDPattern.MatchString(roomID)
}

// IssueToken handles POST /livekit/token requests.
// Generates a short-lived LiveKit access token for authenticated users.
// Requires valid JWT authentication and associates the user's DID with the LiveKit session.
func (h *LiveKitHandlers) IssueToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user DID from context (set by auth middleware)
	userDID := middleware.GetUserDID(ctx)
	if userDID == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Parse request body
	var req LiveKitTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body")
		return
	}

	// Validate room ID
	req.RoomID = strings.TrimSpace(req.RoomID)
	if req.RoomID == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "room_id is required")
		return
	}

	if !validateRoomID(req.RoomID) {
		ctx = middleware.SetErrorCode(ctx, ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "room_id contains invalid characters or exceeds maximum length")
		return
	}

	// TODO: Future enhancement - verify membership if room is restricted
	// For now, any authenticated user can join any room

	// Generate participant identity using shared utility function
	participantID := stream.GenerateParticipantID(userDID)

	// Prepare metadata
	metadata := make(map[string]interface{})
	metadata["did"] = userDID
	if req.SceneID != nil && *req.SceneID != "" {
		metadata["sceneId"] = *req.SceneID
	}
	if req.EventID != nil && *req.EventID != "" {
		metadata["eventId"] = *req.EventID
	}

	// Generate LiveKit token
	tokenReq := &livekit.TokenRequest{
		RoomName: req.RoomID,
		Identity: participantID,
		Expiry:   0, // Use default expiry (5 minutes)
		Metadata: metadata,
	}

	tokenResp, err := h.tokenService.GenerateToken(tokenReq)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate LiveKit token",
			"error", err,
			"room_id", req.RoomID,
			"user_did", userDID,
		)
		ctx = middleware.SetErrorCode(ctx, ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to generate token")
		return
	}

	// Log token issuance for audit (never log the token itself)
	// Use "livekit_room" as entity type and room_id as entity ID
	auditEntry := audit.LogEntry{
		UserDID:    userDID,
		EntityType: "livekit_room",
		EntityID:   req.RoomID,
		Action:     "token_issued",
		RequestID:  middleware.GetRequestID(ctx),
	}

	if _, err := h.auditRepo.LogAccess(auditEntry); err != nil {
		// Log error but don't fail the request
		slog.ErrorContext(ctx, "failed to log token issuance audit entry",
			"error", err,
			"room_id", req.RoomID,
			"user_did", userDID,
		)
	}

	// Return token response
	response := LiveKitTokenResponse{
		Token:     tokenResp.Token,
		ExpiresAt: tokenResp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"), // RFC3339
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(ctx, "failed to encode token response", "error", err)
	}
}
