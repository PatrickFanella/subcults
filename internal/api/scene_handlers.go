// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/color"
	"github.com/onnwee/subcults/internal/membership"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/stream"
	"github.com/onnwee/subcults/internal/tracing"
	"github.com/onnwee/subcults/internal/validate"
	"go.opentelemetry.io/otel/attribute"
)

// CreateSceneRequest represents the request body for creating a scene.
type CreateSceneRequest struct {
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	OwnerDID      string         `json:"owner_did"`
	AllowPrecise  bool           `json:"allow_precise"`
	PrecisePoint  *scene.Point   `json:"precise_point,omitempty"`
	CoarseGeohash string         `json:"coarse_geohash"`
	Tags          []string       `json:"tags,omitempty"`
	Visibility    string         `json:"visibility,omitempty"`
	Palette       *scene.Palette `json:"palette,omitempty"`
}

// UpdateSceneRequest represents the request body for updating a scene.
// Only includes mutable fields (owner is immutable).
type UpdateSceneRequest struct {
	Name         *string        `json:"name,omitempty"`
	Description  *string        `json:"description,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Visibility   *string        `json:"visibility,omitempty"`
	Palette      *scene.Palette `json:"palette,omitempty"`
	AllowPrecise *bool          `json:"allow_precise,omitempty"`
	PrecisePoint *scene.Point   `json:"precise_point,omitempty"`
}

// UpdateScenePaletteRequest represents the request body for updating scene palette.
type UpdateScenePaletteRequest struct {
	Palette scene.Palette `json:"palette"`
}

// SceneHandlers holds dependencies for scene HTTP handlers.
type SceneHandlers struct {
	repo           scene.SceneRepository
	membershipRepo membership.MembershipRepository
	streamRepo     stream.SessionRepository
}

// NewSceneHandlers creates a new SceneHandlers instance.
func NewSceneHandlers(repo scene.SceneRepository, membershipRepo membership.MembershipRepository, streamRepo stream.SessionRepository) *SceneHandlers {
	return &SceneHandlers{
		repo:           repo,
		membershipRepo: membershipRepo,
		streamRepo:     streamRepo,
	}
}

// validateVisibility validates the visibility mode.
func validateVisibility(visibility string) string {
	if visibility == "" {
		return "" // Empty is OK, will default to "public"
	}
	if visibility != "public" && visibility != "private" && visibility != "unlisted" {
		return "visibility must be 'public', 'private', or 'unlisted'"
	}
	return ""
}

// CreateScene handles POST /scenes - creates a new scene.
func (h *SceneHandlers) CreateScene(w http.ResponseWriter, r *http.Request) {
	var req CreateSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate and sanitize name
	validatedName, err := validate.SceneName(req.Name)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidSceneName)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidSceneName, fmt.Sprintf("Invalid scene name: %v", err))
		return
	}
	req.Name = validatedName

	// Validate and sanitize description
	validatedDesc, err := validate.Description(req.Description)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid description: %v", err))
		return
	}
	req.Description = validatedDesc

	// Validate owner_did
	if strings.TrimSpace(req.OwnerDID) == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "owner_did is required")
		return
	}

	// Validate coarse_geohash
	if strings.TrimSpace(req.CoarseGeohash) == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, "coarse_geohash is required")
		return
	}

	// Validate visibility
	if errMsg := validateVisibility(req.Visibility); errMsg != "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, errMsg)
		return
	}

	// Set default visibility
	if req.Visibility == "" {
		req.Visibility = "public"
	}

	// Check for duplicate name
	ctx, endCheckSpan := tracing.StartSpan(r.Context(), "check_duplicate_scene_name")
	tracing.SetAttributes(ctx,
		attribute.String("owner_did", req.OwnerDID),
		attribute.String("scene_name", req.Name))

	exists, err := h.repo.ExistsByOwnerAndName(req.OwnerDID, req.Name, "")
	if err != nil {
		slog.ErrorContext(ctx, "failed to check duplicate scene name", "error", err, "owner_did", req.OwnerDID, "name", req.Name)
		endCheckSpan(err)
		ctx := middleware.SetErrorCode(ctx, ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to check for duplicate scene name")
		return
	}
	endCheckSpan(nil)

	if exists {
		ctx := middleware.SetErrorCode(ctx, ErrCodeDuplicateSceneName)
		WriteError(w, ctx, http.StatusConflict, ErrCodeDuplicateSceneName, "Scene with this name already exists for this owner")
		return
	}

	// Sanitize tags to prevent HTML injection
	sanitizedTags := make([]string, len(req.Tags))
	for i, tag := range req.Tags {
		sanitizedTags[i] = validate.SanitizeHTML(tag)
	}

	// Create scene
	now := time.Now()
	newScene := &scene.Scene{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Description:   req.Description,
		OwnerDID:      req.OwnerDID,
		AllowPrecise:  req.AllowPrecise,
		PrecisePoint:  req.PrecisePoint,
		CoarseGeohash: req.CoarseGeohash,
		Tags:          sanitizedTags,
		Visibility:    req.Visibility,
		Palette:       req.Palette,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	// Insert into repository (will automatically enforce location consent).
	// If AllowPrecise is false, PrecisePoint will be cleared before storage.
	ctx, endInsertSpan := tracing.StartSpan(ctx, "insert_scene")
	tracing.SetAttributes(ctx,
		attribute.String("scene_id", newScene.ID),
		attribute.String("visibility", newScene.Visibility),
		attribute.Bool("allow_precise", newScene.AllowPrecise))

	if err := h.repo.Insert(newScene); err != nil {
		slog.ErrorContext(ctx, "failed to insert scene", "error", err, "scene_id", newScene.ID)
		endInsertSpan(err)
		ctx := middleware.SetErrorCode(ctx, ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to create scene")
		return
	}
	endInsertSpan(nil)

	// Retrieve the stored scene to get privacy-enforced version
	ctx, endGetSpan := tracing.StartSpan(ctx, "get_created_scene")
	stored, err := h.repo.GetByID(newScene.ID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to retrieve created scene", "error", err, "scene_id", newScene.ID)
		endGetSpan(err)
		ctx := middleware.SetErrorCode(ctx, ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve created scene")
		return
	}
	endGetSpan(nil)

	// Add success event
	tracing.AddEvent(ctx, "scene_created",
		attribute.String("scene_id", stored.ID))

	// Return created scene
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(stored); err != nil {
		// Log error but response already started
		return
	}
}

// GetScene handles GET /scenes/{id} - retrieves a scene with visibility enforcement.
func (h *SceneHandlers) GetScene(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID is required")
		return
	}
	sceneID := pathParts[0]

	// Get the scene
	foundScene, err := h.repo.GetByID(sceneID)
	if err != nil {
		// Handle deleted scenes with specific error code
		if err == scene.ErrSceneDeleted {
			slog.DebugContext(r.Context(), "scene deleted", "scene_id", sceneID)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeSceneDeleted)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeSceneDeleted, "Scene not found")
			return
		}
		// Use uniform error message to prevent timing attacks and user enumeration
		// Same error for non-existent and forbidden resources
		if err == scene.ErrSceneNotFound {
			slog.DebugContext(r.Context(), "scene not found", "scene_id", sceneID)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Get requester DID (empty if not authenticated)
	requesterDID := middleware.GetUserDID(r.Context())

	// Check visibility permissions
	canAccess, err := h.canAccessScene(r.Context(), foundScene, requesterDID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check scene access", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to check access permissions")
		return
	}

	if !canAccess {
		// Use uniform error message - same as "not found" to prevent enumeration
		// Log at debug level only to avoid leaking information
		slog.DebugContext(r.Context(), "scene access denied",
			"scene_id", sceneID,
			"visibility", foundScene.Visibility,
			"requester_did", requesterDID,
			"is_owner", foundScene.IsOwner(requesterDID))
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
		WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
		return
	}

	// Log successful access at debug level
	slog.DebugContext(r.Context(), "scene access granted",
		"scene_id", sceneID,
		"visibility", foundScene.Visibility,
		"requester_did", requesterDID)

	// Return scene (privacy already enforced by repository)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(foundScene); err != nil {
		return
	}
}

// canAccessScene checks if a user can access a scene based on visibility rules.
// Returns true if access is allowed, false otherwise.
func (h *SceneHandlers) canAccessScene(ctx context.Context, s *scene.Scene, requesterDID string) (bool, error) {
	// Owner always has access
	if s.IsOwner(requesterDID) {
		return true, nil
	}

	// Check visibility rules
	switch s.Visibility {
	case scene.VisibilityPublic:
		// Public scenes are accessible to everyone
		return true, nil

	case scene.VisibilityMembersOnly:
		// Members-only scenes require active membership
		if requesterDID == "" {
			return false, nil
		}

		// Check if requester is an active member
		m, err := h.membershipRepo.GetBySceneAndUser(s.ID, requesterDID)
		if err != nil {
			// Not a member or error retrieving membership
			if err == membership.ErrMembershipNotFound {
				return false, nil
			}
			return false, err
		}

		// Only active members can access
		return m.Status == "active", nil

	case scene.VisibilityHidden:
		// Hidden scenes only accessible to owner (already checked above)
		return false, nil

	default:
		// Unknown visibility mode - deny access for safety
		slog.WarnContext(ctx, "unknown visibility mode", "visibility", s.Visibility, "scene_id", s.ID)
		return false, nil
	}
}

// UpdateScene handles PATCH /scenes/{id} - updates an existing scene.
func (h *SceneHandlers) UpdateScene(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID from URL path
	// For now, we'll use a simple path parsing; in production this would use chi or similar
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID is required")
		return
	}
	sceneID := pathParts[0]

	// Verify authenticated user
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Parse request body
	var req UpdateSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Get existing scene
	existingScene, err := h.repo.GetByID(sceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Verify ownership
	if !existingScene.IsOwner(userDID) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "You do not have permission to update this scene")
		return
	}

	// Validate and apply updates
	if req.Name != nil {
		// Validate and sanitize new name
		newName, err := validate.SceneName(*req.Name)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidSceneName)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidSceneName, fmt.Sprintf("Invalid scene name: %v", err))
			return
		}

		// Check for duplicate name (excluding current scene)
		exists, err := h.repo.ExistsByOwnerAndName(existingScene.OwnerDID, newName, sceneID)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to check duplicate scene name", "error", err, "owner_did", existingScene.OwnerDID, "name", newName, "scene_id", sceneID)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
			WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to check for duplicate scene name")
			return
		}
		if exists {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeDuplicateSceneName)
			WriteError(w, ctx, http.StatusConflict, ErrCodeDuplicateSceneName, "Scene with this name already exists for this owner")
			return
		}
		existingScene.Name = newName
	}

	if req.Description != nil {
		validatedDesc, err := validate.Description(*req.Description)
		if err != nil {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, fmt.Sprintf("Invalid description: %v", err))
			return
		}
		existingScene.Description = validatedDesc
	}

	if req.Tags != nil {
		sanitizedTags := make([]string, len(req.Tags))
		for i, tag := range req.Tags {
			sanitizedTags[i] = validate.SanitizeHTML(tag)
		}
		existingScene.Tags = sanitizedTags
	}

	if req.Visibility != nil {
		if errMsg := validateVisibility(*req.Visibility); errMsg != "" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, errMsg)
			return
		}
		existingScene.Visibility = *req.Visibility
	}

	if req.Palette != nil {
		existingScene.Palette = req.Palette
	}

	if req.AllowPrecise != nil {
		existingScene.AllowPrecise = *req.AllowPrecise
	}

	if req.PrecisePoint != nil {
		existingScene.PrecisePoint = req.PrecisePoint
	}

	// Note: Repository Update will automatically enforce location consent.
	// If AllowPrecise is false, PrecisePoint will be cleared regardless of request value.
	// This is defense in depth - handler accepts both fields, repository enforces privacy.

	// Update timestamp
	now := time.Now()
	existingScene.UpdatedAt = &now

	// Update in repository (will enforce location consent)
	if err := h.repo.Update(existingScene); err != nil {
		slog.ErrorContext(r.Context(), "failed to update scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to update scene")
		return
	}

	// Retrieve updated scene
	updated, err := h.repo.GetByID(sceneID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve updated scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve updated scene")
		return
	}

	// Return updated scene
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		return
	}
}

// DeleteScene handles DELETE /scenes/{id} - soft-deletes a scene.
func (h *SceneHandlers) DeleteScene(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID is required")
		return
	}
	sceneID := pathParts[0]

	// Verify authenticated user
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Get existing scene to verify ownership
	existingScene, err := h.repo.GetByID(sceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		if err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeSceneDeleted)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeSceneDeleted, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Verify ownership
	if !existingScene.IsOwner(userDID) {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "You do not have permission to delete this scene")
		return
	}

	// Soft delete the scene
	if err := h.repo.Delete(sceneID); err != nil {
		if err == scene.ErrSceneNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		if err == scene.ErrSceneDeleted {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeSceneDeleted)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeSceneDeleted, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to delete scene")
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// UpdateScenePalette handles PATCH /scenes/{id}/palette - updates scene color palette.
func (h *SceneHandlers) UpdateScenePalette(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID is required")
		return
	}
	sceneID := pathParts[0]

	// Parse request body
	var req UpdateScenePaletteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid JSON in request body")
		return
	}

	// Get existing scene first to check ownership
	existingScene, err := h.repo.GetByID(sceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Authorization: Only the owner can update the palette
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}
	if existingScene.OwnerDID != userDID {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "Forbidden: you do not own this scene")
		return
	}

	// Define color fields in deterministic order for consistent validation
	type colorField struct {
		name  string
		value *string // Pointer to the palette field
	}
	colorFields := []colorField{
		{"primary", &req.Palette.Primary},
		{"secondary", &req.Palette.Secondary},
		{"accent", &req.Palette.Accent},
		{"background", &req.Palette.Background},
		{"text", &req.Palette.Text},
	}

	// Validate and sanitize all color fields
	for _, field := range colorFields {
		// Check if field is provided
		if strings.TrimSpace(*field.value) == "" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidPalette)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidPalette, field.name+" color is required")
			return
		}

		// Sanitize to prevent script injection (also validates hex format)
		sanitized := color.SanitizeColor(*field.value)
		if sanitized == "" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidPalette)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidPalette, field.name+" color: invalid hex color format, expected #RRGGBB")
			return
		}

		// Update the palette with sanitized value
		*field.value = sanitized
	}

	// Validate contrast ratio between text and background (WCAG AA minimum 4.5:1)
	ratio, err := color.ValidateContrast(req.Palette.Text, req.Palette.Background)
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInvalidPalette)
		if ratio > 0 {
			msg := fmt.Sprintf("Insufficient contrast between text and background colors (got %s:1, need 4.5:1 minimum for WCAG AA)",
				formatRatio(ratio))
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidPalette, msg)
		} else {
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeInvalidPalette, err.Error())
		}
		return
	}

	// Update palette
	existingScene.Palette = &req.Palette

	// Update timestamp
	now := time.Now()
	existingScene.UpdatedAt = &now

	// Update in repository
	if err := h.repo.Update(existingScene); err != nil {
		slog.ErrorContext(r.Context(), "failed to update scene palette", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to update scene palette")
		return
	}

	// Return updated scene
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(existingScene); err != nil {
		return
	}
}

// formatRatio formats a contrast ratio to 1 decimal place, removing trailing zeros.
// Examples: 4.5 -> "4.5", 4.0 -> "4", 21.0 -> "21"
func formatRatio(ratio float64) string {
	formatted := fmt.Sprintf("%.1f", ratio)
	// Remove trailing zeros after decimal point
	formatted = strings.TrimRight(formatted, "0")
	// Remove decimal point if no fractional part remains
	formatted = strings.TrimRight(formatted, ".")
	return formatted
}

// OwnedSceneSummary represents a summary of a scene owned by the user.
// Used for the dashboard endpoint to provide key metrics without heavy fields.
type OwnedSceneSummary struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description,omitempty"`
	CoarseGeohash   string     `json:"coarse_geohash"`
	Tags            []string   `json:"tags,omitempty"`
	Visibility      string     `json:"visibility"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	MembersCount    int        `json:"members_count"`
	HasActiveStream bool       `json:"has_active_stream"`
}

// ListOwnedScenes handles GET /scenes/owned - lists all scenes owned by the authenticated user.
func (h *SceneHandlers) ListOwnedScenes(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user DID
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Get all scenes owned by user
	scenes, err := h.repo.ListByOwner(userDID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list owned scenes", "error", err, "user_did", userDID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scenes")
		return
	}

	// Early return if no scenes
	if len(scenes) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode([]OwnedSceneSummary{}); err != nil {
			slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
		}
		return
	}

	// Collect all scene IDs for batch queries
	sceneIDs := make([]string, len(scenes))
	for i, sc := range scenes {
		sceneIDs[i] = sc.ID
	}

	// Batch query for membership counts (avoids N+1 query problem)
	membershipCounts, err := h.membershipRepo.CountByScenes(sceneIDs, "active")
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to count memberships", "error", err, "user_did", userDID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve membership counts")
		return
	}

	// Batch query for active streams (avoids N+1 query problem)
	activeStreams, err := h.streamRepo.HasActiveStreamsForScenes(sceneIDs)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check active streams", "error", err, "user_did", userDID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to check active streams")
		return
	}

	// Build summary for each scene
	summaries := make([]OwnedSceneSummary, 0, len(scenes))
	for _, sc := range scenes {
		summary := OwnedSceneSummary{
			ID:              sc.ID,
			Name:            sc.Name,
			Description:     sc.Description,
			CoarseGeohash:   sc.CoarseGeohash,
			Tags:            sc.Tags,
			Visibility:      sc.Visibility,
			CreatedAt:       sc.CreatedAt,
			UpdatedAt:       sc.UpdatedAt,
			MembersCount:    membershipCounts[sc.ID], // Defaults to 0 if not in map
			HasActiveStream: activeStreams[sc.ID],    // Defaults to false if not in map
		}
		summaries = append(summaries, summary)
	}

	// Return summaries
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(summaries); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
		return
	}
}
