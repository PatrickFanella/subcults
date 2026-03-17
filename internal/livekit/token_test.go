package livekit

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/livekit/protocol/auth"
)

func TestNewTokenService(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		apiSecret string
		wantErr   error
	}{
		{
			name:      "valid credentials",
			apiKey:    "test-api-key",
			apiSecret: "test-api-secret",
			wantErr:   nil,
		},
		{
			name:      "missing API key",
			apiKey:    "",
			apiSecret: "test-api-secret",
			wantErr:   ErrMissingAPIKey,
		},
		{
			name:      "missing API secret",
			apiKey:    "test-api-key",
			apiSecret: "",
			wantErr:   ErrMissingAPISecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewTokenService(tt.apiKey, tt.apiSecret)
			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && svc == nil {
				t.Error("expected service to be non-nil")
			}
		})
	}
}

func TestGenerateToken_Success(t *testing.T) {
	svc, err := NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	req := &TokenRequest{
		RoomName: "test-room",
		Identity: "user-123",
		Metadata: map[string]interface{}{
			"sceneId": "scene-456",
			"eventId": "event-789",
		},
	}

	before := time.Now()
	resp, err := svc.GenerateToken(req)
	after := time.Now()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Token == "" {
		t.Error("expected token to be non-empty")
	}

	// Verify expiry is within expected range (default 5m)
	expectedExpiry := before.Add(DefaultTokenExpiry)
	if resp.ExpiresAt.Before(expectedExpiry) || resp.ExpiresAt.After(after.Add(DefaultTokenExpiry).Add(time.Second)) {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, resp.ExpiresAt)
	}

	// Verify token claims
	token, err := jwt.Parse(resp.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-api-secret"), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}

	// Verify video grant
	videoGrant, ok := claims["video"].(map[string]interface{})
	if !ok {
		t.Fatal("expected video grant in claims")
	}

	if room, ok := videoGrant["room"].(string); !ok || room != "test-room" {
		t.Errorf("expected room 'test-room', got %v", videoGrant["room"])
	}

	if roomJoin, ok := videoGrant["roomJoin"].(bool); !ok || !roomJoin {
		t.Errorf("expected roomJoin to be true, got %v", videoGrant["roomJoin"])
	}

	// Verify identity (sub claim)
	if sub, ok := claims["sub"].(string); !ok || sub != "user-123" {
		t.Errorf("expected sub 'user-123', got %v", claims["sub"])
	}

	// Verify metadata
	if metadata, ok := claims["metadata"].(string); !ok || metadata == "" {
		t.Errorf("expected metadata to be present, got %v", claims["metadata"])
	}
}

func TestGenerateToken_CustomExpiry(t *testing.T) {
	svc, err := NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	customExpiry := 10 * time.Minute
	req := &TokenRequest{
		RoomName: "test-room",
		Identity: "user-123",
		Expiry:   customExpiry,
	}

	before := time.Now()
	resp, err := svc.GenerateToken(req)
	after := time.Now()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify expiry matches custom duration
	expectedExpiry := before.Add(customExpiry)
	if resp.ExpiresAt.Before(expectedExpiry) || resp.ExpiresAt.After(after.Add(customExpiry).Add(time.Second)) {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, resp.ExpiresAt)
	}
}

func TestGenerateToken_ValidationErrors(t *testing.T) {
	svc, err := NewTokenService("test-api-key", "test-api-secret")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name    string
		req     *TokenRequest
		wantErr error
	}{
		{
			name: "missing room name",
			req: &TokenRequest{
				Identity: "user-123",
			},
			wantErr: ErrMissingRoomName,
		},
		{
			name: "missing identity",
			req: &TokenRequest{
				RoomName: "test-room",
			},
			wantErr: ErrMissingIdentity,
		},
		{
			name: "expiry too short",
			req: &TokenRequest{
				RoomName: "test-room",
				Identity: "user-123",
				Expiry:   30 * time.Second,
			},
			wantErr: ErrInvalidExpiry,
		},
		{
			name: "expiry too long",
			req: &TokenRequest{
				RoomName: "test-room",
				Identity: "user-123",
				Expiry:   20 * time.Minute,
			},
			wantErr: ErrInvalidExpiry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GenerateToken(tt.req)
			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGenerateToken_DecodeAndVerify(t *testing.T) {
	apiKey := "test-api-key"
	apiSecret := "test-api-secret"

	svc, err := NewTokenService(apiKey, apiSecret)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	req := &TokenRequest{
		RoomName: "test-room-verify",
		Identity: "user-verify-123",
	}

	resp, err := svc.GenerateToken(req)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Use LiveKit's auth package to parse and verify the token
	verifier, err := auth.ParseAPIToken(resp.Token)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	// Verify the token with the secret
	claims, err := verifier.Verify([]byte(apiSecret))
	if err != nil {
		t.Fatalf("failed to verify token with LiveKit auth: %v", err)
	}

	// Verify claims match our request
	if verifier.Identity() != req.Identity {
		t.Errorf("expected identity %s, got %s", req.Identity, verifier.Identity())
	}

	if claims.Video == nil {
		t.Fatal("expected video grant in claims")
	}

	if claims.Video.Room != req.RoomName {
		t.Errorf("expected room %s, got %s", req.RoomName, claims.Video.Room)
	}

	if !claims.Video.RoomJoin {
		t.Error("expected roomJoin grant to be true")
	}
}
