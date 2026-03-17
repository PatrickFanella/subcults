// Package auth provides authentication utilities for JWT token management.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token type constants for the typ claim.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Key version constants for tracking which key signed the token.
const (
	KeyVersionCurrent  = "current"
	KeyVersionPrevious = "previous"
)

// Token expiration durations.
const (
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

// Default leeway for token validation.
const DefaultLeeway = 30 * time.Second

// ErrInvalidToken is returned when token validation fails.
var ErrInvalidToken = errors.New("invalid token")

// ErrExpiredToken is returned when the token has expired.
var ErrExpiredToken = errors.New("token has expired")

// ErrEmptyUserID is returned when userID is empty.
var ErrEmptyUserID = errors.New("userID cannot be empty")

// Claims represents custom JWT claims for the application.
type Claims struct {
	jwt.RegisteredClaims
	DID  string `json:"did,omitempty"` // Decentralized Identifier (for access tokens)
	Type string `json:"typ"`           // Token type: "access" or "refresh"
}

// JWTService handles JWT token operations.
// Supports dual-key rotation: tokens are signed with currentSecret,
// but can be validated with either currentSecret or previousSecret.
type JWTService struct {
	currentSecret  []byte
	previousSecret []byte
	leeway         time.Duration
	keyVersion     string // Key version identifier for current secret
}

// NewJWTService creates a new JWTService with the given secret.
// For backward compatibility, accepts a single secret which is used as currentSecret.
func NewJWTService(secret string) *JWTService {
	return &JWTService{
		currentSecret:  []byte(secret),
		previousSecret: nil,
		leeway:         DefaultLeeway,
		keyVersion:     KeyVersionCurrent,
	}
}

// NewJWTServiceWithLeeway creates a new JWTService with custom leeway.
// For backward compatibility, accepts a single secret which is used as currentSecret.
func NewJWTServiceWithLeeway(secret string, leeway time.Duration) *JWTService {
	return &JWTService{
		currentSecret:  []byte(secret),
		previousSecret: nil,
		leeway:         leeway,
		keyVersion:     KeyVersionCurrent,
	}
}

// NewJWTServiceWithRotation creates a new JWTService with dual-key support for zero-downtime rotation.
// Tokens are always signed with currentSecret, but can be validated with either currentSecret or previousSecret.
// Set previousSecret to empty string if no rotation is in progress.
func NewJWTServiceWithRotation(currentSecret, previousSecret string) *JWTService {
	svc := &JWTService{
		currentSecret: []byte(currentSecret),
		leeway:        DefaultLeeway,
		keyVersion:    KeyVersionCurrent,
	}
	if previousSecret != "" {
		svc.previousSecret = []byte(previousSecret)
	}
	return svc
}

// NewJWTServiceWithRotationAndLeeway creates a new JWTService with dual-key support and custom leeway.
func NewJWTServiceWithRotationAndLeeway(currentSecret, previousSecret string, leeway time.Duration) *JWTService {
	svc := &JWTService{
		currentSecret: []byte(currentSecret),
		leeway:        leeway,
		keyVersion:    KeyVersionCurrent,
	}
	if previousSecret != "" {
		svc.previousSecret = []byte(previousSecret)
	}
	return svc
}

// GenerateAccessToken creates a new access token (15m expiry) with userID and DID.
func (s *JWTService) GenerateAccessToken(userID, did string) (string, error) {
	if userID == "" {
		return "", ErrEmptyUserID
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenExpiry)),
		},
		DID:  did,
		Type: TokenTypeAccess,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Set the key ID (kid) header to track which key version signed this token
	token.Header["kid"] = s.keyVersion
	return token.SignedString(s.currentSecret)
}

// GenerateRefreshToken creates a new refresh token (7d expiry) with userID.
func (s *JWTService) GenerateRefreshToken(userID string) (string, error) {
	if userID == "" {
		return "", ErrEmptyUserID
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenExpiry)),
		},
		Type: TokenTypeRefresh,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Set the key ID (kid) header to track which key version signed this token
	token.Header["kid"] = s.keyVersion
	return token.SignedString(s.currentSecret)
}

// ValidateToken parses and validates a JWT token, returning the claims if valid.
// Supports dual-key rotation: tries currentSecret first, then previousSecret if available.
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	var firstErr error

	// Try validating with current secret first
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method is HS256
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}
		return s.currentSecret, nil
	}, jwt.WithLeeway(s.leeway))

	if err == nil {
		claims, ok := token.Claims.(*Claims)
		if ok && token.Valid {
			return claims, nil
		}
		return nil, ErrInvalidToken
	}

	// Remember the first error in case it was an expiration error
	firstErr = err

	// If current secret fails and previous secret is available, try previous secret
	if s.previousSecret != nil {
		token, err = jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidToken
			}
			return s.previousSecret, nil
		}, jwt.WithLeeway(s.leeway))

		if err == nil {
			claims, ok := token.Claims.(*Claims)
			if ok && token.Valid {
				return claims, nil
			}
		}

		// If second attempt also failed, check if either error was expiration
		if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(firstErr, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
	} else {
		// No previous secret, so check the first error directly
		if errors.Is(firstErr, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
	}

	return nil, ErrInvalidToken
}
