package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 44-character base64 string, as produced by `openssl rand -base64 32`
const testSecret = "wJ6Qk8Qn1v9Qw1Zb2l8Qk9J3p6Qk8Qn1v9Qw1Zb2l8Qk="

func TestGenerateAccessToken(t *testing.T) {
	svc := NewJWTService(testSecret)

	tests := []struct {
		name    string
		userID  string
		did     string
		wantErr bool
	}{
		{
			name:    "valid access token",
			userID:  "user-123",
			did:     "did:web:example.com",
			wantErr: false,
		},
		{
			name:    "empty userID",
			userID:  "",
			did:     "did:web:example.com",
			wantErr: true,
		},
		{
			name:    "empty did",
			userID:  "user-123",
			did:     "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.GenerateAccessToken(tt.userID, tt.did)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && token == "" {
				t.Error("GenerateAccessToken() returned empty token")
			}
		})
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	svc := NewJWTService(testSecret)

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "valid refresh token",
			userID:  "user-123",
			wantErr: false,
		},
		{
			name:    "empty userID",
			userID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.GenerateRefreshToken(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && token == "" {
				t.Error("GenerateRefreshToken() returned empty token")
			}
		})
	}
}

func TestValidateAccessToken(t *testing.T) {
	svc := NewJWTService(testSecret)

	// Generate a valid access token
	validToken, err := svc.GenerateAccessToken("user-123", "did:web:example.com")
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantUserID string
		wantDID    string
		wantType   string
		wantErr    error
	}{
		{
			name:       "valid access token",
			token:      validToken,
			wantUserID: "user-123",
			wantDID:    "did:web:example.com",
			wantType:   TokenTypeAccess,
			wantErr:    nil,
		},
		{
			name:    "invalid token format",
			token:   "not-a-valid-token",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := svc.ValidateToken(tt.token)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateToken() unexpected error = %v", err)
				return
			}
			if claims.Subject != tt.wantUserID {
				t.Errorf("ValidateToken() Subject = %v, want %v", claims.Subject, tt.wantUserID)
			}
			if claims.DID != tt.wantDID {
				t.Errorf("ValidateToken() DID = %v, want %v", claims.DID, tt.wantDID)
			}
			if claims.Type != tt.wantType {
				t.Errorf("ValidateToken() Type = %v, want %v", claims.Type, tt.wantType)
			}
		})
	}
}

func TestValidateRefreshToken(t *testing.T) {
	svc := NewJWTService(testSecret)

	// Generate a valid refresh token
	validToken, err := svc.GenerateRefreshToken("user-456")
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	claims, err := svc.ValidateToken(validToken)
	if err != nil {
		t.Fatalf("ValidateToken() unexpected error = %v", err)
	}
	if claims.Subject != "user-456" {
		t.Errorf("ValidateToken() Subject = %v, want user-456", claims.Subject)
	}
	if claims.DID != "" {
		t.Errorf("ValidateToken() DID = %v, want empty", claims.DID)
	}
	if claims.Type != TokenTypeRefresh {
		t.Errorf("ValidateToken() Type = %v, want %v", claims.Type, TokenTypeRefresh)
	}
}

func TestExpiredToken(t *testing.T) {
	svc := NewJWTServiceWithLeeway(testSecret, 0) // No leeway for this test

	// Create an expired token manually
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-expired",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // Expired 1 hour ago
		},
		Type: TokenTypeAccess,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	_, err = svc.ValidateToken(tokenString)
	if err != ErrExpiredToken {
		t.Errorf("ValidateToken() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestTamperedToken(t *testing.T) {
	svc := NewJWTService(testSecret)

	// Generate a valid token
	validToken, err := svc.GenerateAccessToken("user-123", "did:web:example.com")
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Tamper with the token by modifying the signature
	parts := strings.Split(validToken, ".")
	if len(parts) != 3 {
		t.Fatalf("Invalid token format")
	}

	// Corrupt the signature
	tamperedToken := parts[0] + "." + parts[1] + ".tamperedsignature"

	_, err = svc.ValidateToken(tamperedToken)
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestWrongSecretToken(t *testing.T) {
	svc1 := NewJWTService("secret-one")
	svc2 := NewJWTService("secret-two")

	// Generate a token with one secret
	token, err := svc1.GenerateAccessToken("user-123", "did:web:example.com")
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Try to validate with a different secret
	_, err = svc2.ValidateToken(token)
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestTokenClaims(t *testing.T) {
	svc := NewJWTService(testSecret)

	t.Run("access token claims", func(t *testing.T) {
		beforeGen := time.Now().Add(-1 * time.Second)
		token, err := svc.GenerateAccessToken("user-123", "did:web:example.com")
		if err != nil {
			t.Fatalf("Failed to generate access token: %v", err)
		}
		afterGen := time.Now().Add(1 * time.Second)

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		// Check sub claim
		if claims.Subject != "user-123" {
			t.Errorf("Subject = %v, want user-123", claims.Subject)
		}

		// Check did claim
		if claims.DID != "did:web:example.com" {
			t.Errorf("DID = %v, want did:web:example.com", claims.DID)
		}

		// Check typ claim
		if claims.Type != TokenTypeAccess {
			t.Errorf("Type = %v, want %v", claims.Type, TokenTypeAccess)
		}

		// Check iat claim (issued at)
		if claims.IssuedAt == nil {
			t.Error("IssuedAt is nil")
		} else {
			iat := claims.IssuedAt.Time
			if iat.Before(beforeGen) || iat.After(afterGen) {
				t.Errorf("IssuedAt = %v, want between %v and %v", iat, beforeGen, afterGen)
			}
		}

		// Check exp claim
		if claims.ExpiresAt == nil {
			t.Error("ExpiresAt is nil")
		} else {
			expectedExp := claims.IssuedAt.Time.Add(AccessTokenExpiry)
			if !claims.ExpiresAt.Time.Equal(expectedExp) {
				t.Errorf("ExpiresAt = %v, want %v", claims.ExpiresAt.Time, expectedExp)
			}
		}
	})

	t.Run("refresh token claims", func(t *testing.T) {
		beforeGen := time.Now().Add(-1 * time.Second)
		token, err := svc.GenerateRefreshToken("user-456")
		if err != nil {
			t.Fatalf("Failed to generate refresh token: %v", err)
		}
		afterGen := time.Now().Add(1 * time.Second)

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		// Check sub claim
		if claims.Subject != "user-456" {
			t.Errorf("Subject = %v, want user-456", claims.Subject)
		}

		// Check did claim (should be empty for refresh tokens)
		if claims.DID != "" {
			t.Errorf("DID = %v, want empty", claims.DID)
		}

		// Check typ claim
		if claims.Type != TokenTypeRefresh {
			t.Errorf("Type = %v, want %v", claims.Type, TokenTypeRefresh)
		}

		// Check iat claim
		if claims.IssuedAt == nil {
			t.Error("IssuedAt is nil")
		} else {
			iat := claims.IssuedAt.Time
			if iat.Before(beforeGen) || iat.After(afterGen) {
				t.Errorf("IssuedAt = %v, want between %v and %v", iat, beforeGen, afterGen)
			}
		}

		// Check exp claim
		if claims.ExpiresAt == nil {
			t.Error("ExpiresAt is nil")
		} else {
			expectedExp := claims.IssuedAt.Time.Add(RefreshTokenExpiry)
			if !claims.ExpiresAt.Time.Equal(expectedExp) {
				t.Errorf("ExpiresAt = %v, want %v", claims.ExpiresAt.Time, expectedExp)
			}
		}
	})
}

func TestLeewayValidation(t *testing.T) {
	// Create a token that expired just now (within leeway)
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-leeway",
			IssuedAt:  jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-10 * time.Second)), // Expired 10 seconds ago
		},
		Type: TokenTypeAccess,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	t.Run("with default leeway (30s) - should pass", func(t *testing.T) {
		svc := NewJWTService(testSecret) // Default 30s leeway
		_, err := svc.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, expected no error (within leeway)", err)
		}
	})

	t.Run("with no leeway - should fail", func(t *testing.T) {
		svc := NewJWTServiceWithLeeway(testSecret, 0)
		_, err := svc.ValidateToken(tokenString)
		if err != ErrExpiredToken {
			t.Errorf("ValidateToken() error = %v, want %v", err, ErrExpiredToken)
		}
	})
}

func TestEmptyUserIDError(t *testing.T) {
	svc := NewJWTService(testSecret)

	t.Run("access token with empty userID", func(t *testing.T) {
		_, err := svc.GenerateAccessToken("", "did:web:example.com")
		if err != ErrEmptyUserID {
			t.Errorf("GenerateAccessToken() error = %v, want %v", err, ErrEmptyUserID)
		}
	})

	t.Run("refresh token with empty userID", func(t *testing.T) {
		_, err := svc.GenerateRefreshToken("")
		if err != ErrEmptyUserID {
			t.Errorf("GenerateRefreshToken() error = %v, want %v", err, ErrEmptyUserID)
		}
	})
}

// TestKeyRotation tests the dual-key rotation feature for zero-downtime secret rotation.
func TestKeyRotation(t *testing.T) {
	currentSecret := "current-secret-key-12345678"
	previousSecret := "previous-secret-key-87654321"

	t.Run("token signed with current secret validates with current", func(t *testing.T) {
		svc := NewJWTServiceWithRotation(currentSecret, previousSecret)
		token, err := svc.GenerateAccessToken("user-123", "did:web:example.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() error = %v", err)
		}
		if claims.Subject != "user-123" {
			t.Errorf("ValidateToken() Subject = %v, want user-123", claims.Subject)
		}
	})

	t.Run("token signed with previous secret still validates", func(t *testing.T) {
		// Create token with previous secret (simulating old token)
		oldSvc := NewJWTService(previousSecret)
		oldToken, err := oldSvc.GenerateAccessToken("user-456", "did:web:old.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		// Validate with new service that has both secrets
		newSvc := NewJWTServiceWithRotation(currentSecret, previousSecret)
		claims, err := newSvc.ValidateToken(oldToken)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, expected old token to validate with previousSecret", err)
		}
		if claims.Subject != "user-456" {
			t.Errorf("ValidateToken() Subject = %v, want user-456", claims.Subject)
		}
	})

	t.Run("new tokens always use current secret", func(t *testing.T) {
		svc := NewJWTServiceWithRotation(currentSecret, previousSecret)
		token, err := svc.GenerateAccessToken("user-789", "did:web:new.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		// Should validate with current secret only
		currentOnlySvc := NewJWTService(currentSecret)
		claims, err := currentOnlySvc.ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, token should be signed with current secret", err)
		}
		if claims.Subject != "user-789" {
			t.Errorf("ValidateToken() Subject = %v, want user-789", claims.Subject)
		}

		// Should NOT validate with previous secret only
		previousOnlySvc := NewJWTService(previousSecret)
		_, err = previousOnlySvc.ValidateToken(token)
		if err != ErrInvalidToken {
			t.Errorf("ValidateToken() error = %v, want %v (token should not validate with previous secret only)", err, ErrInvalidToken)
		}
	})

	t.Run("rotation without previous secret works", func(t *testing.T) {
		svc := NewJWTServiceWithRotation(currentSecret, "")
		token, err := svc.GenerateAccessToken("user-single", "did:web:single.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() error = %v", err)
		}
		if claims.Subject != "user-single" {
			t.Errorf("ValidateToken() Subject = %v, want user-single", claims.Subject)
		}
	})

	t.Run("token with wrong secret fails", func(t *testing.T) {
		wrongSecret := "wrong-secret-key-99999999"
		wrongSvc := NewJWTService(wrongSecret)
		wrongToken, err := wrongSvc.GenerateAccessToken("user-wrong", "did:web:wrong.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		// Should not validate with rotation service
		svc := NewJWTServiceWithRotation(currentSecret, previousSecret)
		_, err = svc.ValidateToken(wrongToken)
		if err != ErrInvalidToken {
			t.Errorf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
		}
	})
}

func TestBackwardCompatibility(t *testing.T) {
	secret := "backward-compat-secret-12345"

	t.Run("NewJWTService still works as before", func(t *testing.T) {
		svc := NewJWTService(secret)
		token, err := svc.GenerateAccessToken("user-compat", "did:web:compat.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() error = %v", err)
		}
		if claims.Subject != "user-compat" {
			t.Errorf("ValidateToken() Subject = %v, want user-compat", claims.Subject)
		}
	})

	t.Run("NewJWTServiceWithLeeway still works as before", func(t *testing.T) {
		svc := NewJWTServiceWithLeeway(secret, 60*time.Second)
		token, err := svc.GenerateRefreshToken("user-leeway")
		if err != nil {
			t.Fatalf("GenerateRefreshToken() error = %v", err)
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() error = %v", err)
		}
		if claims.Subject != "user-leeway" {
			t.Errorf("ValidateToken() Subject = %v, want user-leeway", claims.Subject)
		}
	})
}

func TestRotationWithCustomLeeway(t *testing.T) {
	currentSecret := "current-leeway-key-123456"
	previousSecret := "previous-leeway-key-654321"

	// Create an expired token with previous secret
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-expired-leeway",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-10 * time.Second)), // Expired 10 seconds ago
		},
		Type: TokenTypeAccess,
	}

	oldSvc := NewJWTService(previousSecret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(previousSecret))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	t.Run("expired token with leeway validates through previous secret", func(t *testing.T) {
		svc := NewJWTServiceWithRotationAndLeeway(currentSecret, previousSecret, 30*time.Second)
		_, err := svc.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, expected token to validate with leeway", err)
		}
	})

	t.Run("expired token without leeway fails", func(t *testing.T) {
		svc := NewJWTServiceWithRotationAndLeeway(currentSecret, previousSecret, 0)
		_, err := svc.ValidateToken(tokenString)
		if err != ErrExpiredToken {
			t.Errorf("ValidateToken() error = %v, want %v", err, ErrExpiredToken)
		}
	})

	// Ensure oldSvc is "used" to avoid unused variable error
	_ = oldSvc
}

// TestKeyVersionTracking tests that tokens include the key version in the header.
func TestKeyVersionTracking(t *testing.T) {
	svc := NewJWTService(testSecret)

	t.Run("access token includes key version", func(t *testing.T) {
		tokenString, err := svc.GenerateAccessToken("user-123", "did:web:example.com")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		// Parse token without validation to inspect header
		token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
		if err != nil {
			t.Fatalf("ParseUnverified() error = %v", err)
		}

		kid, ok := token.Header["kid"]
		if !ok {
			t.Error("Token header missing 'kid' field")
		}
		if kid != KeyVersionCurrent {
			t.Errorf("Token kid = %v, want %v", kid, KeyVersionCurrent)
		}
	})

	t.Run("refresh token includes key version", func(t *testing.T) {
		tokenString, err := svc.GenerateRefreshToken("user-456")
		if err != nil {
			t.Fatalf("GenerateRefreshToken() error = %v", err)
		}

		// Parse token without validation to inspect header
		token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
		if err != nil {
			t.Fatalf("ParseUnverified() error = %v", err)
		}

		kid, ok := token.Header["kid"]
		if !ok {
			t.Error("Token header missing 'kid' field")
		}
		if kid != KeyVersionCurrent {
			t.Errorf("Token kid = %v, want %v", kid, KeyVersionCurrent)
		}
	})
}

// TestExpiredTokenErrorSemantics ensures expired tokens return ErrExpiredToken even during rotation.
func TestExpiredTokenErrorSemantics(t *testing.T) {
	currentSecret := "current-secret-key-12345678"
	previousSecret := "previous-secret-key-87654321"

	t.Run("expired token with current secret returns ErrExpiredToken", func(t *testing.T) {
		svc := NewJWTServiceWithRotationAndLeeway(currentSecret, previousSecret, 0)

		// Create an expired token with current secret
		now := time.Now()
		claims := Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-expired",
				IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			},
			Type: TokenTypeAccess,
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token.Header["kid"] = KeyVersionCurrent
		tokenString, err := token.SignedString([]byte(currentSecret))
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		_, err = svc.ValidateToken(tokenString)
		if err != ErrExpiredToken {
			t.Errorf("ValidateToken() error = %v, want %v", err, ErrExpiredToken)
		}
	})

	t.Run("expired token with previous secret returns ErrExpiredToken", func(t *testing.T) {
		svc := NewJWTServiceWithRotationAndLeeway(currentSecret, previousSecret, 0)

		// Create an expired token with previous secret
		now := time.Now()
		claims := Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-expired-old",
				IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			},
			Type: TokenTypeAccess,
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token.Header["kid"] = KeyVersionPrevious
		tokenString, err := token.SignedString([]byte(previousSecret))
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		_, err = svc.ValidateToken(tokenString)
		if err != ErrExpiredToken {
			t.Errorf("ValidateToken() error = %v, want %v (expired token should return ErrExpiredToken)", err, ErrExpiredToken)
		}
	})
}
