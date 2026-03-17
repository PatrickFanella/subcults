// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/json"
	"html"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/alliance"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/trust"
)

const (
	// MaxReasonLength is the maximum allowed length for alliance reason text.
	MaxReasonLength = 256
)

// CreateAllianceRequest represents the request body for creating an alliance.
type CreateAllianceRequest struct {
	FromSceneID string  `json:"from_scene_id"`
	ToSceneID   string  `json:"to_scene_id"`
	Weight      float64 `json:"weight"`
	Reason      *string `json:"reason,omitempty"`
}

// UpdateAllianceRequest represents the request body for updating an alliance.
type UpdateAllianceRequest struct {
	Weight *float64 `json:"weight,omitempty"`
	Reason *string  `json:"reason,omitempty"`
}

// AllianceHandlers holds dependencies for alliance HTTP handlers.
type AllianceHandlers struct {
	allianceRepo     alliance.AllianceRepository
	sceneRepo        scene.SceneRepository
	trustDataSource  trust.DataSource
	trustDirtyTracker *trust.DirtyTracker
}

// NewAllianceHandlers creates a new AllianceHandlers instance.
func NewAllianceHandlers(
	allianceRepo alliance.AllianceRepository,
	sceneRepo scene.SceneRepository,
	trustDataSource trust.DataSource,
	trustDirtyTracker *trust.DirtyTracker,
) *AllianceHandlers {
	return &AllianceHandlers{
		allianceRepo:      allianceRepo,
		sceneRepo:         sceneRepo,
		trustDataSource:   trustDataSource,
		trustDirtyTracker: trustDirtyTracker,
	}
}

// validateAllianceWeight validates alliance weight is between 0.0 and 1.0.
// Returns error message if validation fails, empty string if valid.
func validateAllianceWeight(weight float64) string {
	if weight < 0.0 || weight > 1.0 {
		return "weight must be between 0.0 and 1.0"
	}
	return ""
}

// validateReason validates alliance reason length and content.
// Returns error message if validation fails, empty string if valid.
func validateReason(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return "reason cannot be empty or whitespace only"
	}
	if len(trimmed) > MaxReasonLength {
		return "reason must not exceed 256 characters"
	}
	return ""
}

// CreateAlliance handles POST /alliances - creates a new alliance.
func (h *AllianceHandlers) CreateAlliance(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Parse request body
	var req CreateAllianceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate weight
	if errMsg := validateAllianceWeight(req.Weight); errMsg != "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidWeight)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidWeight, errMsg)
		return
	}

	// Validate reason length if provided
	if req.Reason != nil {
		if errMsg := validateReason(*req.Reason); errMsg != "" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, errMsg)
			return
		}
	}

	// Validate scene IDs are not empty
	if strings.TrimSpace(req.FromSceneID) == "" || strings.TrimSpace(req.ToSceneID) == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "from_scene_id and to_scene_id are required")
		return
	}

	// Validate distinct scene IDs
	if req.FromSceneID == req.ToSceneID {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeSelfAlliance)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeSelfAlliance, "Cannot create alliance with same scene")
		return
	}

	// Verify from_scene exists and user owns it
	fromScene, err := h.sceneRepo.GetByID(req.FromSceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "From scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve from scene", "error", err, "scene_id", req.FromSceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve from scene")
		return
	}

	// Check ownership
	if !fromScene.IsOwner(userDID) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "Only scene owner can create alliances")
		return
	}

	// Verify to_scene exists
	_, err = h.sceneRepo.GetByID(req.ToSceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "To scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve to scene", "error", err, "scene_id", req.ToSceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve to scene")
		return
	}

	// Sanitize reason if provided
	var sanitizedReason *string
	if req.Reason != nil {
		escaped := html.EscapeString(*req.Reason)
		sanitizedReason = &escaped
	}

	// Create alliance
	newAlliance := &alliance.Alliance{
		ID:          uuid.New().String(),
		FromSceneID: req.FromSceneID,
		ToSceneID:   req.ToSceneID,
		Weight:      req.Weight,
		Status:      "active",
		Reason:      sanitizedReason,
	}

	if err := h.allianceRepo.Insert(newAlliance); err != nil {
		slog.ErrorContext(r.Context(), "failed to create alliance", "error", err)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to create alliance")
		return
	}

	// Sync alliance to trust data source for trust score computation
	h.trustDataSource.AddAlliance(trust.Alliance{
		FromSceneID: newAlliance.FromSceneID,
		ToSceneID:   newAlliance.ToSceneID,
		Weight:      newAlliance.Weight,
	})

	// Mark scene as dirty for trust recomputation
	h.trustDirtyTracker.MarkDirty(newAlliance.FromSceneID)

	// Retrieve created alliance to ensure consistency
	created, err := h.allianceRepo.GetByID(newAlliance.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve created alliance", "error", err, "alliance_id", newAlliance.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve created alliance")
		return
	}

	// Return created alliance
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

// GetAlliance handles GET /alliances/{id} - retrieves an alliance by ID.
func (h *AllianceHandlers) GetAlliance(w http.ResponseWriter, r *http.Request) {
	// Extract alliance ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/alliances/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Alliance ID is required")
		return
	}
	allianceID := pathParts[0]

	// Retrieve alliance
	foundAlliance, err := h.allianceRepo.GetByID(allianceID)
	if err != nil {
		if err == alliance.ErrAllianceNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Alliance not found")
			return
		}
		if err == alliance.ErrAllianceDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeAllianceDeleted)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeAllianceDeleted, "Alliance not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve alliance", "error", err, "alliance_id", allianceID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve alliance")
		return
	}

	// Return alliance
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(foundAlliance); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

// UpdateAlliance handles PATCH /alliances/{id} - updates an existing alliance.
func (h *AllianceHandlers) UpdateAlliance(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Extract alliance ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/alliances/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Alliance ID is required")
		return
	}
	allianceID := pathParts[0]

	// Parse request body
	var req UpdateAllianceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Get existing alliance
	existingAlliance, err := h.allianceRepo.GetByID(allianceID)
	if err != nil {
		if err == alliance.ErrAllianceNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Alliance not found")
			return
		}
		if err == alliance.ErrAllianceDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeAllianceDeleted)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeAllianceDeleted, "Alliance not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve alliance", "error", err, "alliance_id", allianceID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve alliance")
		return
	}

	// Verify from_scene ownership
	fromScene, err := h.sceneRepo.GetByID(existingAlliance.FromSceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "From scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve from scene", "error", err, "scene_id", existingAlliance.FromSceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve from scene")
		return
	}

	// Check ownership
	if !fromScene.IsOwner(userDID) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "Only scene owner can update alliances")
		return
	}

	// Apply updates
	if req.Weight != nil {
		if errMsg := validateAllianceWeight(*req.Weight); errMsg != "" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidWeight)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidWeight, errMsg)
			return
		}
		existingAlliance.Weight = *req.Weight
	}

	if req.Reason != nil {
		if errMsg := validateReason(*req.Reason); errMsg != "" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, errMsg)
			return
		}
		sanitized := html.EscapeString(*req.Reason)
		existingAlliance.Reason = &sanitized
	}

	// Update in repository (repository will set UpdatedAt)
	if err := h.allianceRepo.Update(existingAlliance); err != nil {
		slog.ErrorContext(r.Context(), "failed to update alliance", "error", err, "alliance_id", allianceID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to update alliance")
		return
	}

	// Sync updated alliance to trust data source
	// Clear old alliances and re-add all active alliances for the scene
	h.trustDataSource.ClearAlliances(existingAlliance.FromSceneID)
	
	// Re-fetch all alliances for this scene and sync them
	// For now, just add the updated one (in production with DB, we'd query all active alliances)
	h.trustDataSource.AddAlliance(trust.Alliance{
		FromSceneID: existingAlliance.FromSceneID,
		ToSceneID:   existingAlliance.ToSceneID,
		Weight:      existingAlliance.Weight,
	})

	// Mark scene as dirty for trust recomputation
	h.trustDirtyTracker.MarkDirty(existingAlliance.FromSceneID)

	// Retrieve updated alliance
	updated, err := h.allianceRepo.GetByID(allianceID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve updated alliance", "error", err, "alliance_id", allianceID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve updated alliance")
		return
	}

	// Return updated alliance
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

// DeleteAlliance handles DELETE /alliances/{id} - soft-deletes an alliance.
func (h *AllianceHandlers) DeleteAlliance(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Extract alliance ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/alliances/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Alliance ID is required")
		return
	}
	allianceID := pathParts[0]

	// Get existing alliance to check ownership
	existingAlliance, err := h.allianceRepo.GetByID(allianceID)
	if err != nil {
		if err == alliance.ErrAllianceNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Alliance not found")
			return
		}
		if err == alliance.ErrAllianceDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeAllianceDeleted)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeAllianceDeleted, "Alliance not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve alliance", "error", err, "alliance_id", allianceID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve alliance")
		return
	}

	// Verify from_scene ownership
	fromScene, err := h.sceneRepo.GetByID(existingAlliance.FromSceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "From scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve from scene", "error", err, "scene_id", existingAlliance.FromSceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve from scene")
		return
	}

	// Check ownership
	if !fromScene.IsOwner(userDID) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "Only scene owner can delete alliances")
		return
	}

	// Soft delete the alliance
	if err := h.allianceRepo.Delete(allianceID); err != nil {
		if err == alliance.ErrAllianceNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Alliance not found")
			return
		}
		if err == alliance.ErrAllianceDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeAllianceDeleted)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeAllianceDeleted, "Alliance not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete alliance", "error", err, "alliance_id", allianceID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to delete alliance")
		return
	}

	// Remove alliance from trust data source
	// Clear all alliances for the scene and let the trust job recompute
	// In production with DB, we'd query and re-sync remaining active alliances
	h.trustDataSource.ClearAlliances(existingAlliance.FromSceneID)

	// Mark scene as dirty for trust recomputation
	h.trustDirtyTracker.MarkDirty(existingAlliance.FromSceneID)

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}
