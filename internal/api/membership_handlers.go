// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/membership"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
)

// MembershipHandlers holds dependencies for membership HTTP handlers.
type MembershipHandlers struct {
	membershipRepo membership.MembershipRepository
	sceneRepo      scene.SceneRepository
	auditRepo      audit.Repository
}

// NewMembershipHandlers creates a new MembershipHandlers instance.
func NewMembershipHandlers(
	membershipRepo membership.MembershipRepository,
	sceneRepo scene.SceneRepository,
	auditRepo audit.Repository,
) *MembershipHandlers {
	return &MembershipHandlers{
		membershipRepo: membershipRepo,
		sceneRepo:      sceneRepo,
		auditRepo:      auditRepo,
	}
}

// RequestMembership handles POST /scenes/{id}/membership/request
// Creates a pending membership request for the authenticated user.
func (h *MembershipHandlers) RequestMembership(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID is required")
		return
	}
	sceneID := pathParts[0]

	// Get authenticated user DID from context
	userDID := middleware.GetUserDID(r.Context())
	if userDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Verify scene exists
	existingScene, err := h.sceneRepo.GetByID(sceneID)
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

	// Check if user is already the scene owner
	if existingScene.OwnerDID == userDID {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeConflict)
		WriteError(w, ctx, http.StatusConflict, ErrCodeConflict, "Scene owner cannot request membership")
		return
	}

	// Check for existing membership
	existingMembership, err := h.membershipRepo.GetBySceneAndUser(sceneID, userDID)
	if err == nil {
		// Membership exists
		if existingMembership.Status == "pending" {
			// Duplicate pending request - return 409 Conflict
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeConflict)
			WriteError(w, ctx, http.StatusConflict, ErrCodeConflict, "Pending membership request already exists")
			return
		}
		if existingMembership.Status == "active" {
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeConflict)
			WriteError(w, ctx, http.StatusConflict, ErrCodeConflict, "User is already an active member")
			return
		}
		// If status is "rejected", allow creating a new request by updating the existing one
	} else if err != membership.ErrMembershipNotFound {
		// Unexpected error
		slog.ErrorContext(r.Context(), "failed to check existing membership", "error", err, "scene_id", sceneID, "user_did", userDID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to check existing membership")
		return
	}

	// Create or update membership request
	newMembership := &membership.Membership{
		SceneID:     sceneID,
		UserDID:     userDID,
		Role:        "member", // Default role for requests
		Status:      "pending",
		TrustWeight: 0.5, // Default trust weight
	}

	// If there's a rejected membership, update it by setting the ID
	if existingMembership != nil && existingMembership.Status == "rejected" {
		newMembership.ID = existingMembership.ID
	}

	result, err := h.membershipRepo.Upsert(newMembership)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create membership request", "error", err, "scene_id", sceneID, "user_did", userDID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to create membership request")
		return
	}

	// Audit log the membership request
	if h.auditRepo != nil {
		if err := audit.LogAccessFromRequest(r, h.auditRepo, "membership", result.ID, "membership_request", audit.OutcomeSuccess); err != nil {
			slog.WarnContext(r.Context(), "failed to log membership request audit", "error", err, "membership_id", result.ID)
			// Continue - audit failure should not block the operation
		}
	}

	// Retrieve the created/updated membership to get complete data with timestamps
	createdMembership, err := h.membershipRepo.GetByID(result.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve created membership", "error", err, "membership_id", result.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve created membership")
		return
	}

	// Return created membership
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(createdMembership); err != nil {
		return
	}
}

// ApproveMembership handles POST /scenes/{id}/membership/{userId}/approve
// Approves a pending membership request (scene owner only).
func (h *MembershipHandlers) ApproveMembership(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID and user DID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) < 4 || pathParts[0] == "" || pathParts[2] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID and User DID are required")
		return
	}
	sceneID := pathParts[0]

	// URL decode the DID
	targetUserDID, err := url.PathUnescape(pathParts[2])
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid user DID in URL")
		return
	}

	// Get authenticated user DID from context
	ownerDID := middleware.GetUserDID(r.Context())
	if ownerDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Verify scene exists and user is owner
	existingScene, err := h.sceneRepo.GetByID(sceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound {
			// Use uniform error message to prevent enumeration
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Check authorization - only owner can approve
	if existingScene.OwnerDID != ownerDID {
		// Use uniform error message to prevent enumeration
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "Only scene owner can approve memberships")
		return
	}

	// Get the membership to approve
	existingMembership, err := h.membershipRepo.GetBySceneAndUser(sceneID, targetUserDID)
	if err != nil {
		if err == membership.ErrMembershipNotFound {
			// Use uniform error message to prevent enumeration
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Membership request not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve membership", "error", err, "scene_id", sceneID, "user_did", targetUserDID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve membership")
		return
	}

	// Verify membership is in pending status
	if existingMembership.Status != "pending" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeConflict)
		WriteError(w, ctx, http.StatusConflict, ErrCodeConflict, "Only pending membership requests can be approved")
		return
	}

	// Update status to active with current timestamp as since
	now := time.Now()
	if err := h.membershipRepo.UpdateStatus(existingMembership.ID, "active", &now); err != nil {
		slog.ErrorContext(r.Context(), "failed to approve membership", "error", err, "membership_id", existingMembership.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to approve membership")
		return
	}

	// Audit log the approval
	if h.auditRepo != nil {
		if err := audit.LogAccessFromRequest(r, h.auditRepo, "membership", existingMembership.ID, "membership_approve", audit.OutcomeSuccess); err != nil {
			slog.WarnContext(r.Context(), "failed to log membership approval audit", "error", err, "membership_id", existingMembership.ID)
			// Continue - audit failure should not block the operation
		}
	}

	// Get updated membership for response
	updatedMembership, err := h.membershipRepo.GetByID(existingMembership.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve approved membership", "error", err, "membership_id", existingMembership.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve approved membership")
		return
	}

	// Return approved membership
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedMembership); err != nil {
		return
	}
}

// RejectMembership handles POST /scenes/{id}/membership/{userId}/reject
// Rejects a pending membership request (scene owner only).
func (h *MembershipHandlers) RejectMembership(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID and user DID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scenes/"), "/")
	if len(pathParts) < 4 || pathParts[0] == "" || pathParts[2] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID and User DID are required")
		return
	}
	sceneID := pathParts[0]

	// URL decode the DID
	targetUserDID, err := url.PathUnescape(pathParts[2])
	if err != nil {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid user DID in URL")
		return
	}

	// Get authenticated user DID from context
	ownerDID := middleware.GetUserDID(r.Context())
	if ownerDID == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
		WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "Authentication required")
		return
	}

	// Verify scene exists and user is owner
	existingScene, err := h.sceneRepo.GetByID(sceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound {
			// Use uniform error message to prevent enumeration
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Check authorization - only owner can reject
	if existingScene.OwnerDID != ownerDID {
		// Use uniform error message to prevent enumeration
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeForbidden)
		WriteError(w, ctx, http.StatusForbidden, ErrCodeForbidden, "Only scene owner can reject memberships")
		return
	}

	// Get the membership to reject
	existingMembership, err := h.membershipRepo.GetBySceneAndUser(sceneID, targetUserDID)
	if err != nil {
		if err == membership.ErrMembershipNotFound {
			// Use uniform error message to prevent enumeration
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeNotFound, "Membership request not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve membership", "error", err, "scene_id", sceneID, "user_did", targetUserDID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve membership")
		return
	}

	// Verify membership is in pending status
	if existingMembership.Status != "pending" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeConflict)
		WriteError(w, ctx, http.StatusConflict, ErrCodeConflict, "Only pending membership requests can be rejected")
		return
	}

	// Update status to rejected (without changing since timestamp)
	if err := h.membershipRepo.UpdateStatus(existingMembership.ID, "rejected", nil); err != nil {
		slog.ErrorContext(r.Context(), "failed to reject membership", "error", err, "membership_id", existingMembership.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to reject membership")
		return
	}

	// Audit log the rejection
	if h.auditRepo != nil {
		if err := audit.LogAccessFromRequest(r, h.auditRepo, "membership", existingMembership.ID, "membership_reject", audit.OutcomeSuccess); err != nil {
			slog.WarnContext(r.Context(), "failed to log membership rejection audit", "error", err, "membership_id", existingMembership.ID)
			// Continue - audit failure should not block the operation
		}
	}

	// Get updated membership for response
	updatedMembership, err := h.membershipRepo.GetByID(existingMembership.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve rejected membership", "error", err, "membership_id", existingMembership.ID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve rejected membership")
		return
	}

	// Return rejected membership
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedMembership); err != nil {
		return
	}
}
