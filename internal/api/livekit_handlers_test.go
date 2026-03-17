package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/livekit"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/stream"
)

func TestIssueToken_Success(t *testing.T) {
	// Setup
	tokenService, err := livekit.NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewLiveKitHandlers(tokenService, auditRepo)

	// Create request
	reqBody := LiveKitTokenRequest{
		RoomID: "test-room-123",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/livekit/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set user DID in context (simulating auth middleware)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute
	handlers.IssueToken(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp LiveKitTokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Token == "" {
		t.Error("expected token to be non-empty")
	}

	if resp.ExpiresAt == "" {
		t.Error("expected expiresAt to be non-empty")
	}

	// Verify token expiry is in RFC3339 format
	expiryTime, err := time.Parse(time.RFC3339, resp.ExpiresAt)
	if err != nil {
		t.Errorf("failed to parse expiresAt as RFC3339: %v", err)
	}

	// Verify expiry is in the future (within 5-6 minutes from now)
	now := time.Now()
	if expiryTime.Before(now) {
		t.Error("expected expiresAt to be in the future")
	}
	expectedMax := now.Add(6 * time.Minute)
	if expiryTime.After(expectedMax) {
		t.Error("expected expiresAt to be within 6 minutes from now")
	}

	// Verify token can be decoded and contains expected claims
	verifier, err := auth.ParseAPIToken(resp.Token)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, err := verifier.Verify([]byte("test-api-secret"))
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	// Verify room name
	if claims.Video == nil || claims.Video.Room != "test-room-123" {
		t.Errorf("expected room 'test-room-123', got %v", claims.Video)
	}

	// Verify participant identity format (user-{uuid})
	identity := verifier.Identity()
	if len(identity) < 6 || identity[:5] != "user-" {
		t.Errorf("expected identity to start with 'user-', got %s", identity)
	}
}

func TestIssueToken_WithSceneAndEvent(t *testing.T) {
	// Setup
	tokenService, err := livekit.NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewLiveKitHandlers(tokenService, auditRepo)

	sceneID := "scene-456"
	eventID := "event-789"
	reqBody := LiveKitTokenRequest{
		RoomID:  "test-room-123",
		SceneID: &sceneID,
		EventID: &eventID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/livekit/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Execute
	handlers.IssueToken(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp LiveKitTokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify token contains metadata
	verifier, err := auth.ParseAPIToken(resp.Token)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, err := verifier.Verify([]byte("test-api-secret"))
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	// Metadata should be present
	if claims.Metadata == "" {
		t.Error("expected metadata to be present in token")
	}

	// Verify audit log was created
	entries, err := auditRepo.QueryByEntity("livekit_room", "test-room-123", 10)
	if err != nil {
		t.Fatalf("failed to query audit entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one audit entry")
	}

	entry := entries[0]
	if entry.Action != "token_issued" {
		t.Errorf("expected action 'token_issued', got %s", entry.Action)
	}
	if entry.UserDID != "did:plc:test123" {
		t.Errorf("expected user DID 'did:plc:test123', got %s", entry.UserDID)
	}
	if entry.EntityID != "test-room-123" {
		t.Errorf("expected entity ID 'test-room-123', got %s", entry.EntityID)
	}
	if entry.EntityType != "livekit_room" {
		t.Errorf("expected entity type 'livekit_room', got %s", entry.EntityType)
	}
}

func TestIssueToken_Unauthorized(t *testing.T) {
	// Setup
	tokenService, err := livekit.NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewLiveKitHandlers(tokenService, auditRepo)

	reqBody := LiveKitTokenRequest{
		RoomID: "test-room-123",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/livekit/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Note: NOT setting user DID in context (simulating missing auth)
	w := httptest.NewRecorder()

	// Execute
	handlers.IssueToken(w, req)

	// Verify
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

func TestIssueToken_InvalidRoomID(t *testing.T) {
	tokenService, err := livekit.NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewLiveKitHandlers(tokenService, auditRepo)

	tests := []struct {
		name   string
		roomID string
	}{
		{
			name:   "empty room ID",
			roomID: "",
		},
		{
			name:   "room ID with spaces",
			roomID: "test room",
		},
		{
			name:   "room ID with special chars",
			roomID: "test@room",
		},
		{
			name:   "room ID too long",
			roomID: strings.Repeat("a", 129), // 129 characters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := LiveKitTokenRequest{
				RoomID: tt.roomID,
			}
			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/livekit/token", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handlers.IssueToken(w, req)

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
		})
	}
}

func TestIssueToken_InvalidJSON(t *testing.T) {
	tokenService, err := livekit.NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewLiveKitHandlers(tokenService, auditRepo)

	req := httptest.NewRequest(http.MethodPost, "/livekit/token", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.IssueToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeBadRequest {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeBadRequest, errResp.Error.Code)
	}
}

func TestValidateRoomID(t *testing.T) {
	tests := []struct {
		name   string
		roomID string
		valid  bool
	}{
		{
			name:   "valid alphanumeric",
			roomID: "room123",
			valid:  true,
		},
		{
			name:   "valid with hyphens",
			roomID: "test-room-123",
			valid:  true,
		},
		{
			name:   "valid with underscores",
			roomID: "test_room_123",
			valid:  true,
		},
		{
			name:   "valid with colons",
			roomID: "scene:event:123",
			valid:  true,
		},
		{
			name:   "valid mixed",
			roomID: "test-room_123:abc",
			valid:  true,
		},
		{
			name:   "invalid with spaces",
			roomID: "test room",
			valid:  false,
		},
		{
			name:   "invalid with special chars",
			roomID: "test@room",
			valid:  false,
		},
		{
			name:   "invalid empty",
			roomID: "",
			valid:  false,
		},
		{
			name:   "invalid too long",
			roomID: strings.Repeat("a", 129),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateRoomID(tt.roomID)
			if result != tt.valid {
				t.Errorf("expected validateRoomID(%q) = %v, got %v", tt.roomID, tt.valid, result)
			}
		})
	}
}

func TestGenerateParticipantID(t *testing.T) {
	tests := []struct {
		name        string
		did         string
		expectStart string
	}{
		{
			name:        "standard DID format",
			did:         "did:plc:abc123def456",
			expectStart: "user-abc123def456",
		},
		{
			name:        "DID with long identifier",
			did:         "did:plc:verylongidentifier123456789012345678901234567890",
			expectStart: "user-verylongidentifier1234567890123456789012345678",
		},
		{
			name:        "short DID",
			did:         "did:plc:abc",
			expectStart: "user-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stream.GenerateParticipantID(tt.did)
			if !strings.HasPrefix(result, "user-") {
				t.Errorf("expected participant ID to start with 'user-', got %s", result)
			}

			// Verify determinism: same DID should produce same ID
			result2 := stream.GenerateParticipantID(tt.did)
			if result != result2 {
				t.Errorf("expected deterministic ID generation, got %s and %s", result, result2)
			}

			// Verify different DIDs produce different IDs (when not truncated)
			differentDID := tt.did + "x"
			result3 := stream.GenerateParticipantID(differentDID)

			// Only check if they're different when the change would be preserved after truncation
			// For long identifiers that get truncated, the extra 'x' might be cut off
			if len(strings.Split(tt.did, ":")[len(strings.Split(tt.did, ":"))-1]) < 48 {
				if result == result3 {
					t.Errorf("expected different DIDs to produce different IDs, got %s for both", result)
				}
			}
		})
	}
}
