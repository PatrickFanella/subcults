package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/membership"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/onnwee/subcults/internal/stream"
	"github.com/onnwee/subcults/internal/validate"
)

// TestCreateScene_Success tests successful scene creation.
func TestCreateScene_Success(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	reqBody := CreateSceneRequest{
		Name:          "Test Scene",
		Description:   "A test scene",
		OwnerDID:      "did:plc:test123",
		AllowPrecise:  true,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060},
		CoarseGeohash: "dr5regw",
		Tags:          []string{"test", "example"},
		Visibility:    "public",
		Palette:       &scene.Palette{Primary: "#ff0000", Secondary: "#00ff00"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.CreateScene(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var createdScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&createdScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createdScene.Name != "Test Scene" {
		t.Errorf("expected name 'Test Scene', got %s", createdScene.Name)
	}
	if createdScene.OwnerDID != "did:plc:test123" {
		t.Errorf("expected owner_did 'did:plc:test123', got %s", createdScene.OwnerDID)
	}
	if createdScene.Visibility != "public" {
		t.Errorf("expected visibility 'public', got %s", createdScene.Visibility)
	}
	if createdScene.PrecisePoint == nil {
		t.Error("expected precise_point to be set")
	}
	if createdScene.CreatedAt == nil {
		t.Error("expected created_at to be set")
	}
}

// TestCreateScene_DefaultVisibility tests that visibility defaults to "public".
func TestCreateScene_DefaultVisibility(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	reqBody := CreateSceneRequest{
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.CreateScene(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var createdScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&createdScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createdScene.Visibility != "public" {
		t.Errorf("expected default visibility 'public', got %s", createdScene.Visibility)
	}
}

// TestCreateScene_PrivacyEnforcement tests that privacy is enforced on creation.
func TestCreateScene_PrivacyEnforcement(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	reqBody := CreateSceneRequest{
		Name:          "Private Scene",
		OwnerDID:      "did:plc:test123",
		AllowPrecise:  false,                                     // Privacy not consented
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060}, // Should be cleared
		CoarseGeohash: "dr5regw",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.CreateScene(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var createdScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&createdScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createdScene.PrecisePoint != nil {
		t.Error("expected precise_point to be nil when allow_precise=false")
	}
}

// TestCreateScene_DuplicateName tests duplicate name rejection.
func TestCreateScene_DuplicateName(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create first scene
	firstReq := CreateSceneRequest{
		Name:          "Duplicate Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
	}

	body, _ := json.Marshal(firstReq)
	req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handlers.CreateScene(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("first creation failed with status %d", w.Code)
	}

	// Try to create second scene with same name and owner
	body, _ = json.Marshal(firstReq)
	req = httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handlers.CreateScene(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeDuplicateSceneName {
		t.Errorf("expected error code %s, got %s", ErrCodeDuplicateSceneName, errResp.Error.Code)
	}
}

// TestCreateScene_DuplicateNameCaseInsensitive tests case-insensitive duplicate name rejection.
func TestCreateScene_DuplicateNameCaseInsensitive(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()

	tests := []struct {
		name      string
		firstName string
		dupeNames []string
	}{
		{
			name:      "uppercase duplicate",
			firstName: "Test Scene",
			dupeNames: []string{"TEST SCENE", "test scene", "TeSt ScEnE"},
		},
		{
			name:      "lowercase duplicate",
			firstName: "my scene",
			dupeNames: []string{"MY SCENE", "My Scene", "mY sCeNe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh repo for each test
			repo := scene.NewInMemorySceneRepository()
			handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

			// Create first scene
			firstReq := CreateSceneRequest{
				Name:          tt.firstName,
				OwnerDID:      "did:plc:test123",
				CoarseGeohash: "dr5regw",
			}

			body, _ := json.Marshal(firstReq)
			req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handlers.CreateScene(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("first creation failed with status %d", w.Code)
			}

			// Try to create scenes with case variations
			for _, dupeName := range tt.dupeNames {
				dupeReq := CreateSceneRequest{
					Name:          dupeName,
					OwnerDID:      "did:plc:test123",
					CoarseGeohash: "dr5regw",
				}

				body, _ := json.Marshal(dupeReq)
				req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
				w := httptest.NewRecorder()
				handlers.CreateScene(w, req)

				if w.Code != http.StatusConflict {
					t.Errorf("expected status 409 for '%s', got %d", dupeName, w.Code)
				}

				var errResp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}

				if errResp.Error.Code != ErrCodeDuplicateSceneName {
					t.Errorf("expected error code %s for '%s', got %s", ErrCodeDuplicateSceneName, dupeName, errResp.Error.Code)
				}
			}
		})
	}
}

// TestCreateScene_InvalidName tests name validation.
func TestCreateScene_InvalidName(t *testing.T) {
	tests := []struct {
		name        string
		sceneName   string
		wantStatus  int
		wantErrCode string
	}{
		{
			name:        "too short",
			sceneName:   "ab",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrCodeInvalidSceneName,
		},
		{
			name:        "too long",
			sceneName:   strings.Repeat("a", 65),
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrCodeInvalidSceneName,
		},
		{
			name:        "invalid characters",
			sceneName:   "Scene<script>alert('xss')</script>",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrCodeInvalidSceneName,
		},
		{
			name:        "special chars not allowed",
			sceneName:   "Scene@#$%",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: ErrCodeInvalidSceneName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := scene.NewInMemorySceneRepository()
			membershipRepo := membership.NewInMemoryMembershipRepository()
			streamRepo := stream.NewInMemorySessionRepository()
			handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

			reqBody := CreateSceneRequest{
				Name:          tt.sceneName,
				OwnerDID:      "did:plc:test123",
				CoarseGeohash: "dr5regw",
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handlers.CreateScene(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != tt.wantErrCode {
				t.Errorf("expected error code %s, got %s", tt.wantErrCode, errResp.Error.Code)
			}
		})
	}
}

// TestCreateScene_MissingRequiredFields tests validation of required fields.
func TestCreateScene_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		reqBody CreateSceneRequest
	}{
		{
			name: "missing owner_did",
			reqBody: CreateSceneRequest{
				Name:          "Test Scene",
				CoarseGeohash: "dr5regw",
			},
		},
		{
			name: "missing coarse_geohash",
			reqBody: CreateSceneRequest{
				Name:     "Test Scene",
				OwnerDID: "did:plc:test123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := scene.NewInMemorySceneRepository()
			membershipRepo := membership.NewInMemoryMembershipRepository()
			streamRepo := stream.NewInMemorySessionRepository()
			handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handlers.CreateScene(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != ErrCodeValidation {
				t.Errorf("expected error code %s, got %s", ErrCodeValidation, errResp.Error.Code)
			}
		})
	}
}

// TestUpdateScene_Success tests successful scene update.
func TestUpdateScene_Success(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene first
	now := time.Now()
	originalScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Original Name",
		Description:   "Original description",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(originalScene); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}

	// Update the scene
	newName := "Updated Name"
	newDesc := "Updated description"
	newVis := "unlisted"
	updateReq := UpdateSceneRequest{
		Name:        &newName,
		Description: &newDesc,
		Visibility:  &newVis,
		Tags:        []string{"updated", "tags"},
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/scenes/test-scene-id", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateScene(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var updatedScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&updatedScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if updatedScene.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", updatedScene.Name)
	}
	if updatedScene.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %s", updatedScene.Description)
	}
	if updatedScene.Visibility != "unlisted" {
		t.Errorf("expected visibility 'unlisted', got %s", updatedScene.Visibility)
	}
	if len(updatedScene.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(updatedScene.Tags))
	}
	if updatedScene.OwnerDID != "did:plc:test123" {
		t.Errorf("owner_did should remain unchanged, got %s", updatedScene.OwnerDID)
	}
}

// TestUpdateScene_NotFound tests updating a non-existent scene.
func TestUpdateScene_NotFound(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	newName := "Updated Name"
	updateReq := UpdateSceneRequest{
		Name: &newName,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/scenes/nonexistent-id", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateScene(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s, got %s", ErrCodeNotFound, errResp.Error.Code)
	}
}

// TestUpdateScene_DuplicateName tests updating to a duplicate name.
func TestUpdateScene_DuplicateName(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	now := time.Now()

	// Create first scene
	scene1 := &scene.Scene{
		ID:            "scene-1",
		Name:          "Scene One",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	repo.Insert(scene1)

	// Create second scene
	scene2 := &scene.Scene{
		ID:            "scene-2",
		Name:          "Scene Two",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	repo.Insert(scene2)

	// Try to update scene-2 to have the same name as scene-1
	newName := "Scene One"
	updateReq := UpdateSceneRequest{
		Name: &newName,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPatch, "/scenes/scene-2", bytes.NewReader(body))
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateScene(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeDuplicateSceneName {
		t.Errorf("expected error code %s, got %s", ErrCodeDuplicateSceneName, errResp.Error.Code)
	}
}

// TestDeleteScene_Success tests successful scene deletion.
func TestDeleteScene_Success(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	repo.Insert(testScene)

	req := httptest.NewRequest(http.MethodDelete, "/scenes/test-scene-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.DeleteScene(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify scene is soft-deleted (returns ErrSceneDeleted on get)
	_, err := repo.GetByID("test-scene-id")
	if err != scene.ErrSceneDeleted {
		t.Errorf("expected scene to be soft-deleted and return ErrSceneDeleted, got: %v", err)
	}
}

// TestDeleteScene_NotFound tests deleting a non-existent scene.
func TestDeleteScene_NotFound(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	req := httptest.NewRequest(http.MethodDelete, "/scenes/nonexistent-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.DeleteScene(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s, got %s", ErrCodeNotFound, errResp.Error.Code)
	}
}

// TestDeleteScene_AlreadyDeleted tests deleting an already deleted scene.
func TestDeleteScene_AlreadyDeleted(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	repo.Insert(testScene)

	// Delete once
	repo.Delete("test-scene-id")

	// Try to delete again
	req := httptest.NewRequest(http.MethodDelete, "/scenes/test-scene-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.DeleteScene(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	// Verify it returns scene_deleted error code
	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeSceneDeleted {
		t.Errorf("expected error code %s for already deleted scene, got %s", ErrCodeSceneDeleted, errResp.Error.Code)
	}
}

// TestValidateSceneName tests scene name validation via the validate package.
func TestValidateSceneName(t *testing.T) {
	tests := []struct {
		name      string
		sceneName string
		wantErr   bool
	}{
		{"valid name", "Test Scene", false},
		{"valid with numbers", "Scene 123", false},
		{"valid with dash", "Test-Scene", false},
		{"valid with underscore", "Test_Scene", false},
		{"valid with period", "Scene v1.0", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 65), true},
		{"invalid chars", "Scene<>", true},
		{"invalid chars @", "Scene@email", true},
		{"invalid apostrophe", "Mike's Scene", true},
		{"invalid ampersand", "Rock & Roll", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validate.SceneName(tt.sceneName)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("validate.SceneName(%q) error = %v, wantErr %v", tt.sceneName, err, tt.wantErr)
			}
		})
	}
}

// TestValidateVisibility tests visibility validation.
func TestValidateVisibility(t *testing.T) {
	tests := []struct {
		visibility string
		wantErr    bool
	}{
		{"public", false},
		{"private", false},
		{"unlisted", false},
		{"", false}, // Empty is OK
		{"invalid", true},
		{"PUBLIC", true}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.visibility, func(t *testing.T) {
			errMsg := validateVisibility(tt.visibility)
			hasErr := errMsg != ""
			if hasErr != tt.wantErr {
				t.Errorf("validateVisibility(%q) error = %v, wantErr %v", tt.visibility, errMsg, tt.wantErr)
			}
		})
	}
}

// TestUpdateScenePalette_Success tests successful palette update.
func TestUpdateScenePalette_Success(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene first
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Update palette
	reqBody := UpdateScenePaletteRequest{
		Palette: scene.Palette{
			Primary:    "#ff0000",
			Secondary:  "#00ff00",
			Accent:     "#0000ff",
			Background: "#ffffff",
			Text:       "#000000",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/scenes/test-scene-id/palette", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Add authentication context
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateScenePalette(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updatedScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&updatedScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if updatedScene.Palette == nil {
		t.Fatal("expected palette to be set")
	}
	if updatedScene.Palette.Primary != "#ff0000" {
		t.Errorf("expected primary color #ff0000, got %s", updatedScene.Palette.Primary)
	}
	if updatedScene.Palette.Text != "#000000" {
		t.Errorf("expected text color #000000, got %s", updatedScene.Palette.Text)
	}
}

// TestUpdateScenePalette_InvalidHexColor tests rejection of invalid hex colors.
func TestUpdateScenePalette_InvalidHexColor(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene first
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	tests := []struct {
		name    string
		palette scene.Palette
		wantErr string
	}{
		{
			name: "invalid primary color",
			palette: scene.Palette{
				Primary:    "not-a-color",
				Secondary:  "#00ff00",
				Accent:     "#0000ff",
				Background: "#ffffff",
				Text:       "#000000",
			},
			wantErr: "primary color",
		},
		{
			name: "missing hash in secondary",
			palette: scene.Palette{
				Primary:    "#ff0000",
				Secondary:  "00ff00",
				Accent:     "#0000ff",
				Background: "#ffffff",
				Text:       "#000000",
			},
			wantErr: "secondary color",
		},
		{
			name: "too short accent color",
			palette: scene.Palette{
				Primary:    "#ff0000",
				Secondary:  "#00ff00",
				Accent:     "#00f",
				Background: "#ffffff",
				Text:       "#000000",
			},
			wantErr: "accent color",
		},
		{
			name: "empty background color",
			palette: scene.Palette{
				Primary:    "#ff0000",
				Secondary:  "#00ff00",
				Accent:     "#0000ff",
				Background: "",
				Text:       "#000000",
			},
			wantErr: "background color is required",
		},
		{
			name: "empty text color",
			palette: scene.Palette{
				Primary:    "#ff0000",
				Secondary:  "#00ff00",
				Accent:     "#0000ff",
				Background: "#ffffff",
				Text:       "",
			},
			wantErr: "text color is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := UpdateScenePaletteRequest{Palette: tt.palette}
			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPatch, "/scenes/test-scene-id/palette", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			// Add authentication context
			ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
			req = req.WithContext(ctx)

			handlers.UpdateScenePalette(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != ErrCodeInvalidPalette {
				t.Errorf("expected error code %s, got %s", ErrCodeInvalidPalette, errResp.Error.Code)
			}

			if !strings.Contains(errResp.Error.Message, tt.wantErr) {
				t.Errorf("expected error message to contain %q, got %q", tt.wantErr, errResp.Error.Message)
			}
		})
	}
}

// TestUpdateScenePalette_InsufficientContrast tests rejection of palettes with poor contrast.
func TestUpdateScenePalette_InsufficientContrast(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene first
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	tests := []struct {
		name string
		text string
		bg   string
	}{
		{
			name: "light gray on white",
			text: "#cccccc",
			bg:   "#ffffff",
		},
		{
			name: "yellow on white",
			text: "#ffff00",
			bg:   "#ffffff",
		},
		{
			name: "light blue on white",
			text: "#aaddff",
			bg:   "#ffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := UpdateScenePaletteRequest{
				Palette: scene.Palette{
					Primary:    "#ff0000",
					Secondary:  "#00ff00",
					Accent:     "#0000ff",
					Background: tt.bg,
					Text:       tt.text,
				},
			}

			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPatch, "/scenes/test-scene-id/palette", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			// Add authentication context
			ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handlers.UpdateScenePalette(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != ErrCodeInvalidPalette {
				t.Errorf("expected error code %s, got %s", ErrCodeInvalidPalette, errResp.Error.Code)
			}

			if !strings.Contains(errResp.Error.Message, "contrast") {
				t.Errorf("expected error message to contain 'contrast', got %q", errResp.Error.Message)
			}
		})
	}
}

// TestUpdateScenePalette_ScriptTagSanitization tests XSS prevention.
func TestUpdateScenePalette_ScriptTagSanitization(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene first
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	reqBody := UpdateScenePaletteRequest{
		Palette: scene.Palette{
			Primary:    "<script>alert(1)</script>",
			Secondary:  "#00ff00",
			Accent:     "#0000ff",
			Background: "#ffffff",
			Text:       "#000000",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/scenes/test-scene-id/palette", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Add authentication context
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateScenePalette(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeInvalidPalette {
		t.Errorf("expected error code %s, got %s", ErrCodeInvalidPalette, errResp.Error.Code)
	}
}

// TestUpdateScenePalette_SceneNotFound tests handling of non-existent scene.
func TestUpdateScenePalette_SceneNotFound(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	reqBody := UpdateScenePaletteRequest{
		Palette: scene.Palette{
			Primary:    "#ff0000",
			Secondary:  "#00ff00",
			Accent:     "#0000ff",
			Background: "#ffffff",
			Text:       "#000000",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/scenes/nonexistent-id/palette", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Add authentication context
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateScenePalette(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s, got %s", ErrCodeNotFound, errResp.Error.Code)
	}
}

// TestUpdateScenePalette_Unauthorized tests rejection when no auth token is provided.
func TestUpdateScenePalette_Unauthorized(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene first
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	reqBody := UpdateScenePaletteRequest{
		Palette: scene.Palette{
			Primary:    "#ff0000",
			Secondary:  "#00ff00",
			Accent:     "#0000ff",
			Background: "#ffffff",
			Text:       "#000000",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/scenes/test-scene-id/palette", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No authentication context provided
	w := httptest.NewRecorder()

	handlers.UpdateScenePalette(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeAuthFailed {
		t.Errorf("expected error code %s, got %s", ErrCodeAuthFailed, errResp.Error.Code)
	}
}

// TestUpdateScenePalette_Forbidden tests rejection when user is not the owner.
func TestUpdateScenePalette_Forbidden(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene owned by a different user
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	reqBody := UpdateScenePaletteRequest{
		Palette: scene.Palette{
			Primary:    "#ff0000",
			Secondary:  "#00ff00",
			Accent:     "#0000ff",
			Background: "#ffffff",
			Text:       "#000000",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/scenes/test-scene-id/palette", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Authenticate as a different user
	ctx := middleware.SetUserDID(req.Context(), "did:plc:different-user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateScenePalette(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeForbidden {
		t.Errorf("expected error code %s, got %s", ErrCodeForbidden, errResp.Error.Code)
	}
}

// TestGetScene_PublicScene tests that public scenes are accessible to everyone.
func TestGetScene_PublicScene(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a public scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "public-scene-id",
		Name:          "Public Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Test access by unauthenticated user
	req := httptest.NewRequest(http.MethodGet, "/scenes/public-scene-id", nil)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var retrievedScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&retrievedScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if retrievedScene.ID != "public-scene-id" {
		t.Errorf("expected scene ID 'public-scene-id', got %s", retrievedScene.ID)
	}

	// Test access by different authenticated user
	req2 := httptest.NewRequest(http.MethodGet, "/scenes/public-scene-id", nil)
	ctx := middleware.SetUserDID(req2.Context(), "did:plc:other-user")
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()

	handlers.GetScene(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200 for authenticated user, got %d", w2.Code)
	}
}

// TestGetScene_MembersOnlyScene_NonMember tests that non-members cannot access members-only scenes.
func TestGetScene_MembersOnlyScene_NonMember(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a members-only scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "members-scene-id",
		Name:          "Members Only Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityMembersOnly,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Test access by non-member (unauthenticated)
	req := httptest.NewRequest(http.MethodGet, "/scenes/members-scene-id", nil)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	// Should return 404 to prevent enumeration (uniform error)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for unauthenticated user, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s, got %s", ErrCodeNotFound, errResp.Error.Code)
	}

	// Test access by authenticated non-member
	req2 := httptest.NewRequest(http.MethodGet, "/scenes/members-scene-id", nil)
	ctx := middleware.SetUserDID(req2.Context(), "did:plc:non-member")
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()

	handlers.GetScene(w2, req2)

	// Should return 404 to prevent enumeration
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for non-member, got %d", w2.Code)
	}
}

// TestGetScene_MembersOnlyScene_ActiveMember tests that active members can access members-only scenes.
func TestGetScene_MembersOnlyScene_ActiveMember(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a members-only scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "members-scene-id",
		Name:          "Members Only Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityMembersOnly,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Create an active membership
	activeMembership := &membership.Membership{
		SceneID: "members-scene-id",
		UserDID: "did:plc:active-member",
		Role:    "member",
		Status:  "active",
	}
	if _, err := membershipRepo.Upsert(activeMembership); err != nil {
		t.Fatalf("failed to create membership: %v", err)
	}

	// Test access by active member
	req := httptest.NewRequest(http.MethodGet, "/scenes/members-scene-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:active-member")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for active member, got %d: %s", w.Code, w.Body.String())
	}

	var retrievedScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&retrievedScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if retrievedScene.ID != "members-scene-id" {
		t.Errorf("expected scene ID 'members-scene-id', got %s", retrievedScene.ID)
	}
}

// TestGetScene_MembersOnlyScene_PendingMember tests that pending members cannot access members-only scenes.
func TestGetScene_MembersOnlyScene_PendingMember(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a members-only scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "members-scene-id",
		Name:          "Members Only Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityMembersOnly,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Create a pending membership
	pendingMembership := &membership.Membership{
		SceneID: "members-scene-id",
		UserDID: "did:plc:pending-member",
		Role:    "member",
		Status:  "pending",
	}
	if _, err := membershipRepo.Upsert(pendingMembership); err != nil {
		t.Fatalf("failed to create membership: %v", err)
	}

	// Test access by pending member (should be denied)
	req := httptest.NewRequest(http.MethodGet, "/scenes/members-scene-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:pending-member")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	// Should return 404 to prevent enumeration
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for pending member, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetScene_MembersOnlyScene_Owner tests that the owner can always access their scenes.
func TestGetScene_MembersOnlyScene_Owner(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a members-only scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "members-scene-id",
		Name:          "Members Only Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityMembersOnly,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Test access by owner
	req := httptest.NewRequest(http.MethodGet, "/scenes/members-scene-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for owner, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetScene_HiddenScene_Owner tests that owner can access hidden scene.
func TestGetScene_HiddenScene_Owner(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a hidden scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "hidden-scene-id",
		Name:          "Hidden Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityHidden,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Test access by owner
	req := httptest.NewRequest(http.MethodGet, "/scenes/hidden-scene-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for owner, got %d: %s", w.Code, w.Body.String())
	}

	var retrievedScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&retrievedScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if retrievedScene.ID != "hidden-scene-id" {
		t.Errorf("expected scene ID 'hidden-scene-id', got %s", retrievedScene.ID)
	}
}

// TestGetScene_HiddenScene_NonOwner tests that non-owners cannot access hidden scenes.
func TestGetScene_HiddenScene_NonOwner(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a hidden scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "hidden-scene-id",
		Name:          "Hidden Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityHidden,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Test access by different user (should be denied with uniform error)
	req := httptest.NewRequest(http.MethodGet, "/scenes/hidden-scene-id", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:other-user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	// Should return 404 to prevent enumeration (same as non-existent scene)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for non-owner, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s, got %s", ErrCodeNotFound, errResp.Error.Code)
	}

	// Also test unauthenticated access
	req2 := httptest.NewRequest(http.MethodGet, "/scenes/hidden-scene-id", nil)
	w2 := httptest.NewRecorder()

	handlers.GetScene(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for unauthenticated user, got %d", w2.Code)
	}
}

// TestGetScene_NotFound tests that non-existent scenes return 404.
func TestGetScene_NotFound(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Test access to non-existent scene
	req := httptest.NewRequest(http.MethodGet, "/scenes/non-existent-id", nil)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s, got %s", ErrCodeNotFound, errResp.Error.Code)
	}

	// Error message should be same as forbidden (uniform error)
	if errResp.Error.Message != "Scene not found" {
		t.Errorf("expected error message 'Scene not found', got %s", errResp.Error.Message)
	}
}

// TestGetScene_PrivacyEnforcement tests that precise location is omitted when allow_precise is false.
func TestGetScene_PrivacyEnforcement(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a public scene with allow_precise=false
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "privacy-scene-id",
		Name:          "Privacy Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		AllowPrecise:  false,
		PrecisePoint:  &scene.Point{Lat: 40.7128, Lng: -74.0060}, // Should be cleared
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Test that precise point is not returned
	req := httptest.NewRequest(http.MethodGet, "/scenes/privacy-scene-id", nil)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var retrievedScene scene.Scene
	if err := json.NewDecoder(w.Body).Decode(&retrievedScene); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Precise point should be nil (enforced by repository)
	if retrievedScene.PrecisePoint != nil {
		t.Errorf("expected precise_point to be nil when allow_precise=false, got %+v", retrievedScene.PrecisePoint)
	}
}

// TestGetScene_SoftDeleted tests that soft-deleted scenes return 404 with scene_deleted error code.
func TestGetScene_SoftDeleted(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create a scene
	now := time.Now()
	testScene := &scene.Scene{
		ID:            "deleted-scene-id",
		Name:          "Deleted Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Soft-delete the scene
	if err := repo.Delete("deleted-scene-id"); err != nil {
		t.Fatalf("failed to delete scene: %v", err)
	}

	// Try to get the deleted scene
	req := httptest.NewRequest(http.MethodGet, "/scenes/deleted-scene-id", nil)
	w := httptest.NewRecorder()

	handlers.GetScene(w, req)

	// Should return 404 with scene_deleted error code
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for deleted scene, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeSceneDeleted {
		t.Errorf("expected error code %s for deleted scene, got %s", ErrCodeSceneDeleted, errResp.Error.Code)
	}

	// Error message should still be "Scene not found" to avoid leaking deletion status
	if errResp.Error.Message != "Scene not found" {
		t.Errorf("expected error message 'Scene not found', got %s", errResp.Error.Message)
	}
}

// TestGetScene_NonExistentVsDeleted tests that non-existent and deleted scenes have different error codes.
func TestGetScene_NonExistentVsDeleted(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	// Create and delete a scene
	now := time.Now()
	deletedScene := &scene.Scene{
		ID:            "deleted-scene-id",
		Name:          "Deleted Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(deletedScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}
	if err := repo.Delete("deleted-scene-id"); err != nil {
		t.Fatalf("failed to delete scene: %v", err)
	}

	// Test deleted scene
	req1 := httptest.NewRequest(http.MethodGet, "/scenes/deleted-scene-id", nil)
	w1 := httptest.NewRecorder()
	handlers.GetScene(w1, req1)

	if w1.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for deleted scene, got %d", w1.Code)
	}

	var deletedErrResp ErrorResponse
	if err := json.NewDecoder(w1.Body).Decode(&deletedErrResp); err != nil {
		t.Fatalf("failed to decode deleted scene error response: %v", err)
	}

	if deletedErrResp.Error.Code != ErrCodeSceneDeleted {
		t.Errorf("expected error code %s for deleted scene, got %s", ErrCodeSceneDeleted, deletedErrResp.Error.Code)
	}

	// Test non-existent scene
	req2 := httptest.NewRequest(http.MethodGet, "/scenes/never-existed-id", nil)
	w2 := httptest.NewRecorder()
	handlers.GetScene(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for non-existent scene, got %d", w2.Code)
	}

	var notFoundErrResp ErrorResponse
	if err := json.NewDecoder(w2.Body).Decode(&notFoundErrResp); err != nil {
		t.Fatalf("failed to decode non-existent scene error response: %v", err)
	}

	if notFoundErrResp.Error.Code != ErrCodeNotFound {
		t.Errorf("expected error code %s for non-existent scene, got %s", ErrCodeNotFound, notFoundErrResp.Error.Code)
	}

	// Both should have the same user-facing message to prevent enumeration
	if deletedErrResp.Error.Message != notFoundErrResp.Error.Message {
		t.Errorf("deleted and non-existent scenes should have same error message for security, got '%s' vs '%s'",
			deletedErrResp.Error.Message, notFoundErrResp.Error.Message)
	}
}

// TestGetScene_OtherScenesAccessibleAfterDeletion tests that deleting one scene doesn't affect others.
func TestGetScene_OtherScenesAccessibleAfterDeletion(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	now := time.Now()

	// Create multiple scenes
	scene1 := &scene.Scene{
		ID:            "scene-1",
		Name:          "Scene One",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	scene2 := &scene.Scene{
		ID:            "scene-2",
		Name:          "Scene Two",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	scene3 := &scene.Scene{
		ID:            "scene-3",
		Name:          "Scene Three",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		Visibility:    scene.VisibilityPublic,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	for _, s := range []*scene.Scene{scene1, scene2, scene3} {
		if err := repo.Insert(s); err != nil {
			t.Fatalf("failed to insert scene %s: %v", s.ID, err)
		}
	}

	// Delete scene-2
	if err := repo.Delete("scene-2"); err != nil {
		t.Fatalf("failed to delete scene-2: %v", err)
	}

	// Verify scene-1 is still accessible
	req1 := httptest.NewRequest(http.MethodGet, "/scenes/scene-1", nil)
	w1 := httptest.NewRecorder()
	handlers.GetScene(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("expected scene-1 to be accessible (200), got %d: %s", w1.Code, w1.Body.String())
	}

	// Verify scene-2 is not accessible (deleted)
	req2 := httptest.NewRequest(http.MethodGet, "/scenes/scene-2", nil)
	w2 := httptest.NewRecorder()
	handlers.GetScene(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected scene-2 to be inaccessible (404), got %d", w2.Code)
	}

	var err2Resp ErrorResponse
	if err := json.NewDecoder(w2.Body).Decode(&err2Resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if err2Resp.Error.Code != ErrCodeSceneDeleted {
		t.Errorf("expected error code %s for deleted scene-2, got %s", ErrCodeSceneDeleted, err2Resp.Error.Code)
	}

	// Verify scene-3 is still accessible
	req3 := httptest.NewRequest(http.MethodGet, "/scenes/scene-3", nil)
	w3 := httptest.NewRecorder()
	handlers.GetScene(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected scene-3 to be accessible (200), got %d: %s", w3.Code, w3.Body.String())
	}
}

// TestDeleteScene_AlreadyDeletedReturnsSceneDeleted tests that deleting an already deleted scene returns scene_deleted error.
func TestDeleteScene_AlreadyDeletedReturnsSceneDeleted(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	now := time.Now()
	testScene := &scene.Scene{
		ID:            "test-scene-id",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:test123",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(testScene); err != nil {
		t.Fatalf("failed to insert test scene: %v", err)
	}

	// Delete once successfully
	req1 := httptest.NewRequest(http.MethodDelete, "/scenes/test-scene-id", nil)
	ctx1 := middleware.SetUserDID(req1.Context(), "did:plc:test123")
	req1 = req1.WithContext(ctx1)
	w1 := httptest.NewRecorder()
	handlers.DeleteScene(w1, req1)

	if w1.Code != http.StatusNoContent {
		t.Errorf("first deletion should succeed with 204, got %d", w1.Code)
	}

	// Try to delete again - should return scene_deleted error code
	req2 := httptest.NewRequest(http.MethodDelete, "/scenes/test-scene-id", nil)
	ctx2 := middleware.SetUserDID(req2.Context(), "did:plc:test123")
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	handlers.DeleteScene(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when deleting already deleted scene, got %d", w2.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w2.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeSceneDeleted {
		t.Errorf("expected error code %s when deleting already deleted scene, got %s", ErrCodeSceneDeleted, errResp.Error.Code)
	}
}

// TestRepository_DeletedSceneExcludedFromExistsByOwnerAndName tests that deleted scenes are excluded from duplicate checks.
func TestRepository_DeletedSceneExcludedFromExistsByOwnerAndName(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()

	now := time.Now()

	// Create and delete a scene
	scene1 := &scene.Scene{
		ID:            "scene-1",
		Name:          "My Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("failed to insert scene: %v", err)
	}
	if err := repo.Delete("scene-1"); err != nil {
		t.Fatalf("failed to delete scene: %v", err)
	}

	// Check if name exists (should not, since scene is deleted)
	exists, err := repo.ExistsByOwnerAndName("did:plc:owner", "My Scene", "")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}

	if exists {
		t.Error("deleted scene should not be counted in ExistsByOwnerAndName")
	}

	// Create a new scene with the same name (should be allowed)
	scene2 := &scene.Scene{
		ID:            "scene-2",
		Name:          "My Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "dr5regw",
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("failed to insert scene with same name as deleted scene: %v", err)
	}

	// Now the name should exist
	exists, err = repo.ExistsByOwnerAndName("did:plc:owner", "My Scene", "")
	if err != nil {
		t.Fatalf("ExistsByOwnerAndName failed: %v", err)
	}

	if !exists {
		t.Error("new scene with same name should be found by ExistsByOwnerAndName")
	}
}

func TestListOwnedScenes_Success(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	userDID := "did:plc:testuser"

	// Create scenes for the user
	scene1 := &scene.Scene{
		ID:            "scene-1",
		Name:          "Scene 1",
		Description:   "First scene",
		OwnerDID:      userDID,
		CoarseGeohash: "dr5regw",
		Visibility:    "public",
		Tags:          []string{"tag1"},
	}
	scene2 := &scene.Scene{
		ID:            "scene-2",
		Name:          "Scene 2",
		Description:   "Second scene",
		OwnerDID:      userDID,
		CoarseGeohash: "dr5regx",
		Visibility:    "private",
	}

	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("Insert scene1 failed: %v", err)
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("Insert scene2 failed: %v", err)
	}

	// Add memberships to scene1
	membership1 := &membership.Membership{
		ID:      "m1",
		SceneID: "scene-1",
		UserDID: "did:plc:member1",
		Role:    "member",
		Status:  "active",
	}
	membership2 := &membership.Membership{
		ID:      "m2",
		SceneID: "scene-1",
		UserDID: "did:plc:member2",
		Role:    "member",
		Status:  "active",
	}
	membership3 := &membership.Membership{
		ID:      "m3",
		SceneID: "scene-1",
		UserDID: "did:plc:member3",
		Role:    "member",
		Status:  "pending", // Should not be counted
	}

	if _, err := membershipRepo.Upsert(membership1); err != nil {
		t.Fatalf("Upsert membership1 failed: %v", err)
	}
	if _, err := membershipRepo.Upsert(membership2); err != nil {
		t.Fatalf("Upsert membership2 failed: %v", err)
	}
	if _, err := membershipRepo.Upsert(membership3); err != nil {
		t.Fatalf("Upsert membership3 failed: %v", err)
	}

	// Add active stream to scene1
	sceneID1 := "scene-1"
	session := &stream.Session{
		ID:               "stream-1",
		SceneID:          &sceneID1,
		RoomName:         "room-1",
		HostDID:          userDID,
		ParticipantCount: 5,
		EndedAt:          nil,
	}
	if _, err := streamRepo.Upsert(session); err != nil {
		t.Fatalf("Upsert stream failed: %v", err)
	}

	// Create request with authenticated user
	req := httptest.NewRequest(http.MethodGet, "/scenes/owned", nil)
	ctx := middleware.SetUserDID(req.Context(), userDID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.ListOwnedScenes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var summaries []OwnedSceneSummary
	if err := json.NewDecoder(w.Body).Decode(&summaries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("expected 2 scenes, got %d", len(summaries))
	}

	// Find scene1 in results
	var scene1Summary *OwnedSceneSummary
	for i := range summaries {
		if summaries[i].ID == "scene-1" {
			scene1Summary = &summaries[i]
			break
		}
	}

	if scene1Summary == nil {
		t.Fatal("scene-1 not found in results")
	}

	// Verify scene1 stats
	if scene1Summary.MembersCount != 2 {
		t.Errorf("expected 2 active members, got %d", scene1Summary.MembersCount)
	}
	if !scene1Summary.HasActiveStream {
		t.Error("expected active stream for scene-1")
	}

	// Find scene2 in results
	var scene2Summary *OwnedSceneSummary
	for i := range summaries {
		if summaries[i].ID == "scene-2" {
			scene2Summary = &summaries[i]
			break
		}
	}

	if scene2Summary == nil {
		t.Fatal("scene-2 not found in results")
	}

	// Verify scene2 stats
	if scene2Summary.MembersCount != 0 {
		t.Errorf("expected 0 members, got %d", scene2Summary.MembersCount)
	}
	if scene2Summary.HasActiveStream {
		t.Error("expected no active stream for scene-2")
	}
}

func TestListOwnedScenes_Unauthenticated(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	req := httptest.NewRequest(http.MethodGet, "/scenes/owned", nil)
	w := httptest.NewRecorder()

	handlers.ListOwnedScenes(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeAuthFailed {
		t.Errorf("expected error code %s, got %s", ErrCodeAuthFailed, errResp.Error.Code)
	}
}

func TestListOwnedScenes_ExcludesDeleted(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	userDID := "did:plc:testuser"

	// Create scenes
	scene1 := &scene.Scene{
		ID:            "scene-1",
		Name:          "Scene 1",
		OwnerDID:      userDID,
		CoarseGeohash: "dr5regw",
	}
	scene2 := &scene.Scene{
		ID:            "scene-2",
		Name:          "Scene 2",
		OwnerDID:      userDID,
		CoarseGeohash: "dr5regx",
	}

	if err := repo.Insert(scene1); err != nil {
		t.Fatalf("Insert scene1 failed: %v", err)
	}
	if err := repo.Insert(scene2); err != nil {
		t.Fatalf("Insert scene2 failed: %v", err)
	}

	// Delete scene1
	if err := repo.Delete("scene-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/scenes/owned", nil)
	ctx := middleware.SetUserDID(req.Context(), userDID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.ListOwnedScenes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var summaries []OwnedSceneSummary
	if err := json.NewDecoder(w.Body).Decode(&summaries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should only return non-deleted scene
	if len(summaries) != 1 {
		t.Errorf("expected 1 scene, got %d", len(summaries))
	}
	if len(summaries) > 0 && summaries[0].ID != "scene-2" {
		t.Errorf("expected scene-2, got %s", summaries[0].ID)
	}
}

func TestListOwnedScenes_EmptyList(t *testing.T) {
	repo := scene.NewInMemorySceneRepository()
	membershipRepo := membership.NewInMemoryMembershipRepository()
	streamRepo := stream.NewInMemorySessionRepository()
	handlers := NewSceneHandlers(repo, membershipRepo, streamRepo)

	userDID := "did:plc:testuser"

	// Create request without any scenes
	req := httptest.NewRequest(http.MethodGet, "/scenes/owned", nil)
	ctx := middleware.SetUserDID(req.Context(), userDID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.ListOwnedScenes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var summaries []OwnedSceneSummary
	if err := json.NewDecoder(w.Body).Decode(&summaries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(summaries) != 0 {
		t.Errorf("expected empty list, got %d scenes", len(summaries))
	}
}
