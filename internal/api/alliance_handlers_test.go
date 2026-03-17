package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onnwee/subcults/internal/alliance"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/trust"
)

// newTestAllianceHandlers creates handlers with in-memory repositories for testing.
func newTestAllianceHandlers() *AllianceHandlers {
	allianceRepo := alliance.NewInMemoryAllianceRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	trustDataSource := trust.NewInMemoryDataSource()
	trustDirtyTracker := trust.NewDirtyTracker()
	return NewAllianceHandlers(allianceRepo, sceneRepo, trustDataSource, trustDirtyTracker)
}

// createTestScene creates a scene for testing.
func createTestScene(t *testing.T, repo scene.SceneRepository, id, ownerDID string) *scene.Scene {
	t.Helper()
	testScene := &scene.Scene{
		ID:            id,
		Name:          "Test Scene",
		Description:   "Test Description",
		OwnerDID:      ownerDID,
		AllowPrecise:  false,
		CoarseGeohash: "u4pruydqqvj",
		Visibility:    scene.VisibilityPublic,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to create test scene: %v", err)
	}
	return testScene
}

// TestCreateAlliance_Success tests successful alliance creation.
func TestCreateAlliance_Success(t *testing.T) {
	handlers := newTestAllianceHandlers()

	// Create test scenes
	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	reqBody := CreateAllianceRequest{
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.8,
		Reason:      stringPtr("Testing collaboration"),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/alliances", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), ownerDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateAlliance(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var result alliance.Alliance
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.FromSceneID != "scene-from" {
		t.Errorf("expected from_scene_id 'scene-from', got '%s'", result.FromSceneID)
	}
	if result.ToSceneID != "scene-to" {
		t.Errorf("expected to_scene_id 'scene-to', got '%s'", result.ToSceneID)
	}
	if result.Weight != 0.8 {
		t.Errorf("expected weight 0.8, got %f", result.Weight)
	}
	if result.Reason == nil || *result.Reason != "Testing collaboration" {
		t.Errorf("expected reason 'Testing collaboration', got %v", result.Reason)
	}
	if result.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", result.Status)
	}
}

// TestCreateAlliance_InvalidWeight tests alliance creation with invalid weight.
func TestCreateAlliance_InvalidWeight(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
	}{
		{"negative weight", -0.5},
		{"weight above 1.0", 1.5},
		{"weight far below 0", -10.0},
		{"weight far above 1", 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := newTestAllianceHandlers()

			ownerDID := "did:plc:owner123"
			createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
			createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

			reqBody := CreateAllianceRequest{
				FromSceneID: "scene-from",
				ToSceneID:   "scene-to",
				Weight:      tt.weight,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/alliances", bytes.NewReader(body))
			ctx := middleware.SetUserDID(req.Context(), ownerDID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handlers.CreateAlliance(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != ErrCodeInvalidWeight {
				t.Errorf("expected error code '%s', got '%s'", ErrCodeInvalidWeight, errResp.Error.Code)
			}
		})
	}
}

// TestCreateAlliance_SelfAlliance tests creating alliance with same from/to scene.
func TestCreateAlliance_SelfAlliance(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-same", ownerDID)

	reqBody := CreateAllianceRequest{
		FromSceneID: "scene-same",
		ToSceneID:   "scene-same",
		Weight:      0.8,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/alliances", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), ownerDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateAlliance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeSelfAlliance {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeSelfAlliance, errResp.Error.Code)
	}
}

// TestCreateAlliance_Unauthorized tests alliance creation by non-owner.
func TestCreateAlliance_Unauthorized(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	unauthorizedDID := "did:plc:unauthorized"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	reqBody := CreateAllianceRequest{
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.8,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/alliances", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), unauthorizedDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateAlliance(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeForbidden {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeForbidden, errResp.Error.Code)
	}
}

// TestCreateAlliance_SceneNotFound tests alliance creation with non-existent scene.
func TestCreateAlliance_SceneNotFound(t *testing.T) {
	tests := []struct {
		name        string
		fromSceneID string
		toSceneID   string
	}{
		{"from scene not found", "nonexistent", "scene-to"},
		{"to scene not found", "scene-from", "nonexistent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := newTestAllianceHandlers()

			ownerDID := "did:plc:owner123"
			createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
			createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

			reqBody := CreateAllianceRequest{
				FromSceneID: tt.fromSceneID,
				ToSceneID:   tt.toSceneID,
				Weight:      0.8,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/alliances", bytes.NewReader(body))
			ctx := middleware.SetUserDID(req.Context(), ownerDID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handlers.CreateAlliance(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("expected status 404, got %d", w.Code)
			}
		})
	}
}

// TestCreateAlliance_ReasonTooLong tests alliance creation with reason exceeding max length.
func TestCreateAlliance_ReasonTooLong(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	// Create reason longer than 256 characters
	longReason := strings.Repeat("a", MaxReasonLength+1)

	reqBody := CreateAllianceRequest{
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.8,
		Reason:      &longReason,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/alliances", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), ownerDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.CreateAlliance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeValidation, errResp.Error.Code)
	}
}

// TestGetAlliance_Success tests successful alliance retrieval.
func TestGetAlliance_Success(t *testing.T) {
	handlers := newTestAllianceHandlers()

	// Create test alliance
	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	testAlliance := &alliance.Alliance{
		ID:          "test-alliance-id",
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.7,
		Status:      "active",
	}
	if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
		t.Fatalf("failed to create test alliance: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/alliances/test-alliance-id", nil)
	w := httptest.NewRecorder()
	handlers.GetAlliance(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result alliance.Alliance
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID != "test-alliance-id" {
		t.Errorf("expected id 'test-alliance-id', got '%s'", result.ID)
	}
	if result.Weight != 0.7 {
		t.Errorf("expected weight 0.7, got %f", result.Weight)
	}
}

// TestUpdateAlliance_Success tests successful alliance update.
func TestUpdateAlliance_Success(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	testAlliance := &alliance.Alliance{
		ID:          "test-alliance-id",
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.5,
		Status:      "active",
	}
	if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
		t.Fatalf("failed to create test alliance: %v", err)
	}

	// Update weight and reason
	newWeight := 0.9
	newReason := "Updated collaboration"
	reqBody := UpdateAllianceRequest{
		Weight: &newWeight,
		Reason: &newReason,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPatch, "/alliances/test-alliance-id", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), ownerDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.UpdateAlliance(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result alliance.Alliance
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Weight != 0.9 {
		t.Errorf("expected weight 0.9, got %f", result.Weight)
	}
	if result.Reason == nil || *result.Reason != "Updated collaboration" {
		t.Errorf("expected reason 'Updated collaboration', got %v", result.Reason)
	}
}

// TestUpdateAlliance_Unauthorized tests alliance update by non-owner.
func TestUpdateAlliance_Unauthorized(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	unauthorizedDID := "did:plc:unauthorized"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	testAlliance := &alliance.Alliance{
		ID:          "test-alliance-id",
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.5,
		Status:      "active",
	}
	if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
		t.Fatalf("failed to create test alliance: %v", err)
	}

	newWeight := 0.9
	reqBody := UpdateAllianceRequest{
		Weight: &newWeight,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPatch, "/alliances/test-alliance-id", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), unauthorizedDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.UpdateAlliance(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

// TestDeleteAlliance_Success tests successful alliance soft deletion.
func TestDeleteAlliance_Success(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	testAlliance := &alliance.Alliance{
		ID:          "test-alliance-id",
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.5,
		Status:      "active",
	}
	if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
		t.Fatalf("failed to create test alliance: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/alliances/test-alliance-id", nil)
	ctx := middleware.SetUserDID(req.Context(), ownerDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.DeleteAlliance(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify alliance is soft-deleted (returns ErrAllianceDeleted on get)
	_, err := handlers.allianceRepo.GetByID("test-alliance-id")
	if err != alliance.ErrAllianceDeleted {
		t.Errorf("expected alliance to be soft-deleted and return ErrAllianceDeleted, got: %v", err)
	}
}

// TestDeleteAlliance_ExcludedFromGet tests that deleted alliance returns 404 on GET.
func TestDeleteAlliance_ExcludedFromGet(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	testAlliance := &alliance.Alliance{
		ID:          "test-alliance-id",
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.5,
		Status:      "active",
	}
	if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
		t.Fatalf("failed to create test alliance: %v", err)
	}

	// Delete the alliance
	deleteReq := httptest.NewRequest(http.MethodDelete, "/alliances/test-alliance-id", nil)
	ctx := middleware.SetUserDID(deleteReq.Context(), ownerDID)
	deleteReq = deleteReq.WithContext(ctx)
	deleteW := httptest.NewRecorder()
	handlers.DeleteAlliance(deleteW, deleteReq)

	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("delete failed: %d", deleteW.Code)
	}

	// Try to get the deleted alliance
	getReq := httptest.NewRequest(http.MethodGet, "/alliances/test-alliance-id", nil)
	getW := httptest.NewRecorder()
	handlers.GetAlliance(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for deleted alliance, got %d", getW.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(getW.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeAllianceDeleted {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeAllianceDeleted, errResp.Error.Code)
	}
}

// TestDeleteAlliance_AlreadyDeleted tests deleting an already deleted alliance.
func TestDeleteAlliance_AlreadyDeleted(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	testAlliance := &alliance.Alliance{
		ID:          "test-alliance-id",
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.5,
		Status:      "active",
	}
	if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
		t.Fatalf("failed to create test alliance: %v", err)
	}

	// First deletion
	req1 := httptest.NewRequest(http.MethodDelete, "/alliances/test-alliance-id", nil)
	ctx1 := middleware.SetUserDID(req1.Context(), ownerDID)
	req1 = req1.WithContext(ctx1)
	w1 := httptest.NewRecorder()
	handlers.DeleteAlliance(w1, req1)

	if w1.Code != http.StatusNoContent {
		t.Fatalf("first deletion should succeed with 204, got %d", w1.Code)
	}

	// Second deletion attempt
	req2 := httptest.NewRequest(http.MethodDelete, "/alliances/test-alliance-id", nil)
	ctx2 := middleware.SetUserDID(req2.Context(), ownerDID)
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	handlers.DeleteAlliance(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when deleting already deleted alliance, got %d", w2.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w2.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeAllianceDeleted {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeAllianceDeleted, errResp.Error.Code)
	}
}

// TestCreateAlliance_Unauthenticated tests alliance creation without authentication.
func TestCreateAlliance_Unauthenticated(t *testing.T) {
	handlers := newTestAllianceHandlers()

	reqBody := CreateAllianceRequest{
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.8,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/alliances", bytes.NewReader(body))
	// No user DID set in context

	w := httptest.NewRecorder()
	handlers.CreateAlliance(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeAuthFailed {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeAuthFailed, errResp.Error.Code)
	}
}

// TestUpdateAlliance_InvalidWeight tests alliance update with invalid weight.
func TestUpdateAlliance_InvalidWeight(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
	}{
		{"negative weight", -0.5},
		{"weight above 1.0", 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := newTestAllianceHandlers()

			ownerDID := "did:plc:owner123"
			createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
			createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

			testAlliance := &alliance.Alliance{
				ID:          "test-alliance-id",
				FromSceneID: "scene-from",
				ToSceneID:   "scene-to",
				Weight:      0.5,
				Status:      "active",
			}
			if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
				t.Fatalf("failed to create test alliance: %v", err)
			}

			reqBody := UpdateAllianceRequest{
				Weight: &tt.weight,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPatch, "/alliances/test-alliance-id", bytes.NewReader(body))
			ctx := middleware.SetUserDID(req.Context(), ownerDID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handlers.UpdateAlliance(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != ErrCodeInvalidWeight {
				t.Errorf("expected error code '%s', got '%s'", ErrCodeInvalidWeight, errResp.Error.Code)
			}
		})
	}
}

// TestUpdateAlliance_ReasonTooLong tests alliance update with reason exceeding max length.
func TestUpdateAlliance_ReasonTooLong(t *testing.T) {
	handlers := newTestAllianceHandlers()

	ownerDID := "did:plc:owner123"
	createTestScene(t, handlers.sceneRepo, "scene-from", ownerDID)
	createTestScene(t, handlers.sceneRepo, "scene-to", "did:plc:other")

	testAlliance := &alliance.Alliance{
		ID:          "test-alliance-id",
		FromSceneID: "scene-from",
		ToSceneID:   "scene-to",
		Weight:      0.5,
		Status:      "active",
	}
	if err := handlers.allianceRepo.Insert(testAlliance); err != nil {
		t.Fatalf("failed to create test alliance: %v", err)
	}

	// Create reason longer than 256 characters
	longReason := strings.Repeat("a", MaxReasonLength+1)

	reqBody := UpdateAllianceRequest{
		Reason: &longReason,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPatch, "/alliances/test-alliance-id", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), ownerDID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.UpdateAlliance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeValidation {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeValidation, errResp.Error.Code)
	}
}

// TestGetAlliance_NotFound tests retrieving a non-existent alliance.
func TestGetAlliance_NotFound(t *testing.T) {
	handlers := newTestAllianceHandlers()

	req := httptest.NewRequest(http.MethodGet, "/alliances/nonexistent-id", nil)
	w := httptest.NewRecorder()
	handlers.GetAlliance(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeNotFound, errResp.Error.Code)
	}
}

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}
