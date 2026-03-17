// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
)

// Moderation-specific error codes.
const (
	ErrCodeInvalidModerationStatus = "invalid_moderation_status"
	ErrCodeModerationExists        = "moderation_already_exists"
)

// ModerationHandlers holds dependencies for moderation-related HTTP handlers.
type ModerationHandlers struct {
	sceneRepo scene.SceneRepository
	adminDIDs []string // List of authorized admin DIDs
}

// NewModerationHandlers creates a new ModerationHandlers instance.
func NewModerationHandlers(
	sceneRepo scene.SceneRepository,
	adminDIDs []string,
) *ModerationHandlers {
	return &ModerationHandlers{
		sceneRepo: sceneRepo,
		adminDIDs: adminDIDs,
	}
}

// isAdminDID checks if the given DID is an authorized admin.
func (h *ModerationHandlers) isAdminDID(did string) bool {
	for _, adminDID := range h.adminDIDs {
		if adminDID == did {
			return true
		}
	}
	return false
}

// MuteSceneRequest represents the request body for muting a scene.
type MuteSceneRequest struct {
	Reason string `json:"reason"`
}

// MuteSceneResponse represents the response for a successful mute operation.
type MuteSceneResponse struct {
	SceneID           string `json:"scene_id"`
	ModerationStatus  string `json:"moderation_status"`
	ModerationReason  string `json:"moderation_reason"`
	ModerationTimestamp string `json:"moderation_timestamp"`
}

// MuteScene hides a scene from public view due to policy violation.
// POST /internal/moderation/scenes/{sceneID}/mute
func (h *ModerationHandlers) MuteScene(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user DID from context
	userDID := middleware.GetUserDID(ctx)
	if userDID == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeUnauthorized)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeUnauthorized, "authentication required")
		return
	}

	// Verify user is an admin
	if !h.isAdminDID(userDID) {
		ctx = middleware.SetErrorCode(ctx, ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "admin privileges required")
		return
	}

	// Extract scene ID from URL
	sceneID := r.PathValue("sceneID")
	if sceneID == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "scene_id is required")
		return
	}

	// Parse request body
	var req MuteSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
		return
	}

	// Validate reason is provided
	if req.Reason == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "reason is required")
		return
	}

	// Get scene from repository
	existingScene, err := h.sceneRepo.GetByID(sceneID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get scene for muting", "scene_id", sceneID, "error", err)
		ctx = middleware.SetErrorCode(ctx, ErrCodeNotFound)
		WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "scene not found")
		return
	}

	// Normalize empty moderation status (e.g., from in-memory repo or legacy rows)
	if existingScene.ModerationStatus == "" {
		existingScene.ModerationStatus = "visible"
	}

	// Check if already muted
	if existingScene.ModerationStatus != "visible" && existingScene.ModerationStatus != "flagged" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeModerationExists)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeModerationExists, "scene is already moderated")
		return
	}

	// Update scene with moderation status
	now := time.Now()
	existingScene.ModerationStatus = "hidden"
	existingScene.ModerationReason = &req.Reason
	existingScene.ModeratedBy = &userDID
	existingScene.ModerationTimestamp = &now

	if err := h.sceneRepo.Update(existingScene); err != nil {
		slog.ErrorContext(ctx, "failed to update scene moderation status", "scene_id", sceneID, "error", err)
		ctx = middleware.SetErrorCode(ctx, ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "failed to mute scene")
		return
	}

	// Log moderation action for audit trail
	slog.InfoContext(ctx, "scene muted",
		"scene_id", sceneID,
		"admin_did", userDID,
		"reason", req.Reason)

	// Build response
	response := MuteSceneResponse{
		SceneID:             sceneID,
		ModerationStatus:    existingScene.ModerationStatus,
		ModerationReason:    req.Reason,
		ModerationTimestamp: now.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// UnmuteSceneResponse represents the response for a successful unmute operation.
type UnmuteSceneResponse struct {
	SceneID          string `json:"scene_id"`
	ModerationStatus string `json:"moderation_status"`
}

// UnmuteScene restores a muted scene to visible status.
// POST /internal/moderation/scenes/{sceneID}/unmute
func (h *ModerationHandlers) UnmuteScene(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user DID from context
	userDID := middleware.GetUserDID(ctx)
	if userDID == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeUnauthorized)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeUnauthorized, "authentication required")
		return
	}

	// Verify user is an admin
	if !h.isAdminDID(userDID) {
		ctx = middleware.SetErrorCode(ctx, ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "admin privileges required")
		return
	}

	// Extract scene ID from URL
	sceneID := r.PathValue("sceneID")
	if sceneID == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "scene_id is required")
		return
	}

	// Get scene from repository
	existingScene, err := h.sceneRepo.GetByID(sceneID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get scene for unmuting", "scene_id", sceneID, "error", err)
		ctx = middleware.SetErrorCode(ctx, ErrCodeNotFound)
		WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "scene not found")
		return
	}

	// Clear moderation status and audit fields
	existingScene.ModerationStatus = "visible"
	existingScene.ModerationReason = nil
	existingScene.ModeratedBy = nil
	existingScene.ModerationTimestamp = nil

	if err := h.sceneRepo.Update(existingScene); err != nil {
		slog.ErrorContext(ctx, "failed to update scene moderation status", "scene_id", sceneID, "error", err)
		ctx = middleware.SetErrorCode(ctx, ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "failed to unmute scene")
		return
	}

	// Log moderation action for audit trail
	slog.InfoContext(ctx, "scene unmuted",
		"scene_id", sceneID,
		"admin_did", userDID)

	// Build response
	response := UnmuteSceneResponse{
		SceneID:          sceneID,
		ModerationStatus: existingScene.ModerationStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
