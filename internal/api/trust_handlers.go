// Package api provides HTTP handlers for the Subcults API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/trust"
)

// TrustScoreBreakdown represents the detailed breakdown of trust score computation.
type TrustScoreBreakdown struct {
	AverageAllianceWeight        float64 `json:"average_alliance_weight"`
	AverageMembershipTrustWeight float64 `json:"average_membership_trust_weight"`
	RoleMultiplierAggregate      float64 `json:"role_multiplier_aggregate"`
}

// TrustScoreResponse represents the response for trust score endpoint.
type TrustScoreResponse struct {
	SceneID     string               `json:"scene_id"`
	TrustScore  float64              `json:"trust_score"`
	Breakdown   *TrustScoreBreakdown `json:"breakdown,omitempty"`
	Stale       bool                 `json:"stale"`
	LastUpdated string               `json:"last_updated,omitempty"`
}

// TrustHandlers holds dependencies for trust HTTP handlers.
type TrustHandlers struct {
	sceneRepo    scene.SceneRepository
	dataSource   trust.DataSource
	scoreStore   trust.ScoreStore
	dirtyTracker *trust.DirtyTracker
}

// NewTrustHandlers creates a new TrustHandlers instance.
func NewTrustHandlers(
	sceneRepo scene.SceneRepository,
	dataSource trust.DataSource,
	scoreStore trust.ScoreStore,
	dirtyTracker *trust.DirtyTracker,
) *TrustHandlers {
	return &TrustHandlers{
		sceneRepo:    sceneRepo,
		dataSource:   dataSource,
		scoreStore:   scoreStore,
		dirtyTracker: dirtyTracker,
	}
}

// GetTrustScore handles GET /trust/{sceneId} - retrieves trust score and breakdown.
func (h *TrustHandlers) GetTrustScore(w http.ResponseWriter, r *http.Request) {
	// Extract scene ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/trust/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Scene ID is required")
		return
	}
	sceneID := pathParts[0]

	// Verify scene exists
	_, err := h.sceneRepo.GetByID(sceneID)
	if err != nil {
		if err == scene.ErrSceneNotFound || err == scene.ErrSceneDeleted {
			slog.DebugContext(r.Context(), "scene not found for trust score", "scene_id", sceneID)
			ctx := middleware.SetErrorCode(r.Context(), ErrCodeSceneNotFound)
			WriteError(w, ctx, http.StatusNotFound, ErrCodeSceneNotFound, "Scene not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to retrieve scene for trust score", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve scene")
		return
	}

	// Get stored trust score
	storedScore, err := h.scoreStore.GetScore(sceneID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve trust score", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve trust score")
		return
	}

	// Check if score needs recomputation
	isStale := h.dirtyTracker.IsDirty(sceneID)

	// Get memberships and alliances for breakdown
	memberships, err := h.dataSource.GetMembershipsByScene(sceneID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve memberships for trust breakdown", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve trust breakdown")
		return
	}

	alliances, err := h.dataSource.GetAlliancesByScene(sceneID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to retrieve alliances for trust breakdown", "error", err, "scene_id", sceneID)
		ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "Failed to retrieve trust breakdown")
		return
	}

	// Compute breakdown
	breakdown := computeTrustBreakdown(memberships, alliances)

	// Prepare response
	response := TrustScoreResponse{
		SceneID:    sceneID,
		TrustScore: 0.0,
		Breakdown:  breakdown,
		Stale:      isStale,
	}

	// Use stored score if available, otherwise use freshly computed score
	if storedScore != nil {
		response.TrustScore = storedScore.Score
		response.LastUpdated = storedScore.ComputedAt.Format("2006-01-02T15:04:05Z07:00")
	} else {
		// No stored score, compute on the fly
		response.TrustScore = trust.ComputeTrustScore(memberships, alliances)
	}

	// Return trust score data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode trust score response", "error", err)
		return
	}
}

// computeTrustBreakdown calculates the detailed breakdown of trust score components.
// These values are informational summaries and do not represent the exact internal computation steps.
func computeTrustBreakdown(memberships []trust.Membership, alliances []trust.Alliance) *TrustScoreBreakdown {
	if len(memberships) == 0 {
		// No memberships means the trust score is 0.0; omit breakdown to avoid
		// misleading defaults that don't reflect the computation formula.
		return nil
	}

	// Calculate average alliance weight
	allianceAvg := 1.0
	if len(alliances) > 0 {
		var allianceSum float64
		for _, a := range alliances {
			allianceSum += a.Weight
		}
		allianceAvg = allianceSum / float64(len(alliances))
	}

	// Calculate informational averages for breakdown
	// Note: The actual trust score formula applies role multipliers to each membership's
	// trust weight BEFORE averaging: avg(trust_weight * role_multiplier)
	var membershipSum float64
	var roleMultSum float64
	for _, m := range memberships {
		membershipSum += m.TrustWeight

		multiplier, ok := trust.RoleMultiplier[m.Role]
		if !ok {
			multiplier = trust.DefaultRoleMultiplier
		}
		roleMultSum += multiplier
	}
	membershipAvg := membershipSum / float64(len(memberships))
	roleMultAvg := roleMultSum / float64(len(memberships))

	return &TrustScoreBreakdown{
		AverageAllianceWeight:        allianceAvg,
		AverageMembershipTrustWeight: membershipAvg,
		RoleMultiplierAggregate:      roleMultAvg,
	}
}
