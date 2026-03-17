// Package livekit provides utilities for LiveKit token generation and management.
package livekit

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/livekit/protocol/auth"
)

// Token expiry configuration
const (
	DefaultTokenExpiry = 5 * time.Minute
	MinTokenExpiry     = 1 * time.Minute
	MaxTokenExpiry     = 15 * time.Minute
)

var (
	// ErrInvalidExpiry is returned when token expiry is outside valid bounds.
	ErrInvalidExpiry = errors.New("token expiry must be between 1 and 15 minutes")

	// ErrMissingAPIKey is returned when API key is empty.
	ErrMissingAPIKey = errors.New("livekit API key is required")

	// ErrMissingAPISecret is returned when API secret is empty.
	ErrMissingAPISecret = errors.New("livekit API secret is required")

	// ErrMissingRoomName is returned when room name is empty.
	ErrMissingRoomName = errors.New("room name is required")

	// ErrMissingIdentity is returned when identity is empty.
	ErrMissingIdentity = errors.New("participant identity is required")
)

// TokenService handles LiveKit token generation.
type TokenService struct {
	apiKey    string
	apiSecret string
}

// NewTokenService creates a new TokenService with the given API credentials.
func NewTokenService(apiKey, apiSecret string) (*TokenService, error) {
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	if apiSecret == "" {
		return nil, ErrMissingAPISecret
	}

	return &TokenService{
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}, nil
}

// TokenRequest represents the parameters for generating a LiveKit access token.
type TokenRequest struct {
	RoomName string                 // Required: LiveKit room name
	Identity string                 // Required: Participant identity (e.g., "user-{uuid}")
	Expiry   time.Duration          // Token expiry (defaults to DefaultTokenExpiry if zero)
	Metadata map[string]interface{} // Optional: Arbitrary metadata to attach to the token
}

// TokenResponse represents the generated token with expiry information.
type TokenResponse struct {
	Token     string    `json:"token"`      // The JWT access token
	ExpiresAt time.Time `json:"expires_at"` // Token expiration time in UTC (RFC3339)
}

// GenerateToken creates a new LiveKit access token with the specified parameters.
// Returns the token string and expiry timestamp, or an error if generation fails.
func (s *TokenService) GenerateToken(req *TokenRequest) (*TokenResponse, error) {
	// Validate required fields
	if req.RoomName == "" {
		return nil, ErrMissingRoomName
	}
	if req.Identity == "" {
		return nil, ErrMissingIdentity
	}

	// Use default expiry if not specified
	expiry := req.Expiry
	if expiry == 0 {
		expiry = DefaultTokenExpiry
	}

	// Validate expiry bounds
	if expiry < MinTokenExpiry || expiry > MaxTokenExpiry {
		return nil, ErrInvalidExpiry
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(expiry)

	// Create LiveKit access token using the auth package
	at := auth.NewAccessToken(s.apiKey, s.apiSecret)
	at.SetIdentity(req.Identity)
	at.AddGrant(&auth.VideoGrant{
		RoomJoin: true,
		Room:     req.RoomName,
	})
	at.SetValidFor(expiry)

	// Add metadata if provided
	if req.Metadata != nil && len(req.Metadata) > 0 {
		// Convert metadata map to string for the token
		// LiveKit expects a string metadata field
		metadataStr := formatMetadata(req.Metadata)
		at.SetMetadata(metadataStr)
	}

	// Generate the signed JWT token
	token, err := at.ToJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt.UTC(),
	}, nil
}

// formatMetadata converts a metadata map to a JSON string for the token.
// Uses proper JSON marshaling to handle escaping and special characters.
func formatMetadata(metadata map[string]interface{}) string {
	// Convert to JSON using standard library for proper escaping
	data, err := json.Marshal(metadata)
	if err != nil {
		// Fallback to empty JSON object if marshaling fails
		return "{}"
	}
	return string(data)
}
