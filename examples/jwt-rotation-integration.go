// Package main demonstrates JWT secret rotation integration
package main

import (
	"fmt"
	"log"

	"github.com/onnwee/subcults/internal/auth"
	"github.com/onnwee/subcults/internal/config"
)

// This example demonstrates how to integrate JWT secret rotation in your application.
func main() {
	// 1. Load configuration (from environment or file)
	cfg, errs := config.Load("")
	if len(errs) > 0 {
		log.Fatalf("Configuration errors: %v", errs)
	}

	// 2. Get JWT secrets with rotation support
	currentSecret, previousSecret := cfg.GetJWTSecrets()

	// 3. Create JWT service with rotation support
	var jwtService *auth.JWTService
	if previousSecret != "" {
		// Dual-key mode: rotation in progress
		jwtService = auth.NewJWTServiceWithRotation(currentSecret, previousSecret)
		fmt.Println("✓ JWT service initialized with rotation support")
		fmt.Println("  - Current secret: [REDACTED]")
		fmt.Println("  - Previous secret: [REDACTED]")
		fmt.Println("  - Old tokens will remain valid during rotation window")
	} else {
		// Single-key mode: no rotation or rotation complete
		jwtService = auth.NewJWTService(currentSecret)
		fmt.Println("✓ JWT service initialized with single key")
		fmt.Println("  - Current secret: [REDACTED]")
	}

	// 4. Use the service to generate tokens
	// Tokens are always signed with the current secret
	accessToken, err := jwtService.GenerateAccessToken("user-123", "did:web:example.com")
	if err != nil {
		log.Fatalf("Failed to generate access token: %v", err)
	}
	fmt.Printf("✓ Generated access token: %s...\n", accessToken[:20])

	refreshToken, err := jwtService.GenerateRefreshToken("user-123")
	if err != nil {
		log.Fatalf("Failed to generate refresh token: %v", err)
	}
	fmt.Printf("✓ Generated refresh token: %s...\n", refreshToken[:20])

	// 5. Validate tokens (works with both current and previous secrets)
	claims, err := jwtService.ValidateToken(accessToken)
	if err != nil {
		log.Fatalf("Failed to validate token: %v", err)
	}
	fmt.Printf("✓ Token validated successfully\n")
	fmt.Printf("  - User ID: %s\n", claims.Subject)
	fmt.Printf("  - DID: %s\n", claims.DID)
	fmt.Printf("  - Token Type: %s\n", claims.Type)
}

// Example migration from legacy setup:
//
// Before (legacy single-key):
//   jwtService := auth.NewJWTService(cfg.JWTSecret)
//
// After (rotation-aware):
//   currentSecret, previousSecret := cfg.GetJWTSecrets()
//   jwtService := auth.NewJWTServiceWithRotation(currentSecret, previousSecret)
//
// The new approach is backward compatible. If previousSecret is empty,
// it behaves exactly like the legacy single-key setup.
