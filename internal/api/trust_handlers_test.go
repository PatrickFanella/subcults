package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/trust"
)

// TestGetTrustScore_Success tests successful trust score retrieval.
func TestGetTrustScore_Success(t *testing.T) {
	// Setup repositories
	sceneRepo := scene.NewInMemorySceneRepository()
	dataSource := trust.NewInMemoryDataSource()
	scoreStore := trust.NewInMemoryScoreStore()
	dirtyTracker := trust.NewDirtyTracker()

	handlers := NewTrustHandlers(sceneRepo, dataSource, scoreStore, dirtyTracker)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     timePtr(time.Now()),
		UpdatedAt:     timePtr(time.Now()),
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Add some memberships
	dataSource.AddMembership(trust.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:user1",
		Role:        "owner",
		TrustWeight: 0.8,
	})
	dataSource.AddMembership(trust.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:user2",
		Role:        "member",
		TrustWeight: 0.6,
	})

	// Add an alliance
	dataSource.AddAlliance(trust.Alliance{
		FromSceneID: "scene-123",
		ToSceneID:   "scene-456",
		Weight:      0.7,
	})

	// Store a trust score (computed from memberships/alliances using the model formula)
	scoreStore.SaveScore(trust.SceneTrustScore{
		SceneID:    "scene-123",
		Score:      0.77, // 0.7 (alliance) * ((0.8*2.0 + 0.6*1.0)/2) = 0.7 * 1.1 = 0.77
		ComputedAt: time.Now(),
	})

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/trust/scene-123", nil)
	w := httptest.NewRecorder()

	handlers.GetTrustScore(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response TrustScoreResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.SceneID != "scene-123" {
		t.Errorf("expected scene_id 'scene-123', got %s", response.SceneID)
	}

	if response.TrustScore < 0 || response.TrustScore > 1 {
		t.Errorf("trust_score should be between 0 and 1, got %f", response.TrustScore)
	}

	if response.Breakdown == nil {
		t.Error("expected breakdown to be present")
	} else {
		if response.Breakdown.AverageAllianceWeight != 0.7 {
			t.Errorf("expected average_alliance_weight 0.7, got %f", response.Breakdown.AverageAllianceWeight)
		}

		// Average membership trust weight should be (0.8 + 0.6) / 2 = 0.7
		if response.Breakdown.AverageMembershipTrustWeight != 0.7 {
			t.Errorf("expected average_membership_trust_weight 0.7, got %f", response.Breakdown.AverageMembershipTrustWeight)
		}

		// Average role multiplier should be (1.0 + 0.5) / 2 = 0.75
		if response.Breakdown.RoleMultiplierAggregate != 0.75 {
			t.Errorf("expected role_multiplier_aggregate 0.75, got %f", response.Breakdown.RoleMultiplierAggregate)
		}
	}

	if response.Stale {
		t.Errorf("expected stale to be false, got %t", response.Stale)
	}

	if response.LastUpdated == "" {
		t.Error("expected last_updated to be present")
	}
}

// TestGetTrustScore_SceneNotFound tests 404 response for missing scene.
func TestGetTrustScore_SceneNotFound(t *testing.T) {
	// Setup repositories
	sceneRepo := scene.NewInMemorySceneRepository()
	dataSource := trust.NewInMemoryDataSource()
	scoreStore := trust.NewInMemoryScoreStore()
	dirtyTracker := trust.NewDirtyTracker()

	handlers := NewTrustHandlers(sceneRepo, dataSource, scoreStore, dirtyTracker)

	// Make request for non-existent scene
	req := httptest.NewRequest(http.MethodGet, "/trust/nonexistent", nil)
	w := httptest.NewRecorder()

	handlers.GetTrustScore(w, req)

	// Assert 404 response
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeSceneNotFound {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeSceneNotFound, errResp.Error.Code)
	}
}

// TestGetTrustScore_StaleFlag tests that stale flag is set when scene is dirty.
func TestGetTrustScore_StaleFlag(t *testing.T) {
	// Setup repositories
	sceneRepo := scene.NewInMemorySceneRepository()
	dataSource := trust.NewInMemoryDataSource()
	scoreStore := trust.NewInMemoryScoreStore()
	dirtyTracker := trust.NewDirtyTracker()

	handlers := NewTrustHandlers(sceneRepo, dataSource, scoreStore, dirtyTracker)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-stale",
		Name:          "Stale Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     timePtr(time.Now()),
		UpdatedAt:     timePtr(time.Now()),
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Add membership
	dataSource.AddMembership(trust.Membership{
		SceneID:     "scene-stale",
		UserDID:     "did:plc:user1",
		Role:        "member",
		TrustWeight: 0.5,
	})

	// Store a trust score
	scoreStore.SaveScore(trust.SceneTrustScore{
		SceneID:    "scene-stale",
		Score:      0.5,
		ComputedAt: time.Now().Add(-1 * time.Hour), // Old score
	})

	// Mark scene as dirty (needs recomputation)
	dirtyTracker.MarkDirty("scene-stale")

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/trust/scene-stale", nil)
	w := httptest.NewRecorder()

	handlers.GetTrustScore(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response TrustScoreResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Stale {
		t.Errorf("expected stale to be true, got %t", response.Stale)
	}
}

// TestGetTrustScore_NoStoredScore tests response when no score is stored.
func TestGetTrustScore_NoStoredScore(t *testing.T) {
	// Setup repositories
	sceneRepo := scene.NewInMemorySceneRepository()
	dataSource := trust.NewInMemoryDataSource()
	scoreStore := trust.NewInMemoryScoreStore()
	dirtyTracker := trust.NewDirtyTracker()

	handlers := NewTrustHandlers(sceneRepo, dataSource, scoreStore, dirtyTracker)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-noscore",
		Name:          "No Score Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     timePtr(time.Now()),
		UpdatedAt:     timePtr(time.Now()),
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Add membership
	dataSource.AddMembership(trust.Membership{
		SceneID:     "scene-noscore",
		UserDID:     "did:plc:user1",
		Role:        "member",
		TrustWeight: 0.5,
	})

	// No stored score - score should be computed on the fly

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/trust/scene-noscore", nil)
	w := httptest.NewRecorder()

	handlers.GetTrustScore(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response TrustScoreResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should compute score on the fly:
	// 1.0 (no alliances factor) * avg(TrustWeight * RoleMultiplier)
	// = 1.0 * avg(0.5 * 0.5) = 1.0 * 0.25 = 0.25
	if response.TrustScore != 0.25 {
		t.Errorf("expected trust_score 0.25, got %f", response.TrustScore)
	}

	// Last updated should be empty when no stored score
	if response.LastUpdated != "" {
		t.Errorf("expected last_updated to be empty, got %s", response.LastUpdated)
	}
}

// TestGetTrustScore_NoMemberships tests response when scene has no memberships.
func TestGetTrustScore_NoMemberships(t *testing.T) {
	// Setup repositories
	sceneRepo := scene.NewInMemorySceneRepository()
	dataSource := trust.NewInMemoryDataSource()
	scoreStore := trust.NewInMemoryScoreStore()
	dirtyTracker := trust.NewDirtyTracker()

	handlers := NewTrustHandlers(sceneRepo, dataSource, scoreStore, dirtyTracker)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-empty",
		Name:          "Empty Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     timePtr(time.Now()),
		UpdatedAt:     timePtr(time.Now()),
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// No memberships or alliances

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/trust/scene-empty", nil)
	w := httptest.NewRecorder()

	handlers.GetTrustScore(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response TrustScoreResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Score should be 0.0 when no memberships
	if response.TrustScore != 0.0 {
		t.Errorf("expected trust_score 0.0, got %f", response.TrustScore)
	}

	// Breakdown should be nil when no memberships to avoid misleading defaults
	if response.Breakdown != nil {
		t.Error("expected breakdown to be nil when no memberships")
	}
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
