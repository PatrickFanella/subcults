# JWT Rotation Integration Example

This example demonstrates how to integrate JWT secret rotation in your application.

## Overview

The example shows:

- How to load configuration with JWT rotation support
- How to create a JWT service with dual-key rotation
- How to generate and validate tokens during rotation
- Migration path from legacy single-key setup

## Running the Example

1. Set up environment variables:

```bash
# Single-key mode (no rotation)
export JWT_SECRET="your-secret-key-32-characters!!"

# Or dual-key mode (rotation in progress)
export JWT_SECRET_CURRENT="new-secret-key-32-characters!!"
export JWT_SECRET_PREVIOUS="old-secret-key-32-characters!!"
```

2. Set required config (for example to compile/run):

```bash
export DATABASE_URL="postgres://localhost/test"
export LIVEKIT_URL="wss://livekit.example.com"
export LIVEKIT_API_KEY="api_key"
export LIVEKIT_API_SECRET="api_secret"
export STRIPE_API_KEY="sk_test_123"
export STRIPE_WEBHOOK_SECRET="whsec_123"
export STRIPE_ONBOARDING_RETURN_URL="https://example.com/return"
export STRIPE_ONBOARDING_REFRESH_URL="https://example.com/refresh"
export MAPTILER_API_KEY="maptiler_key"
export JETSTREAM_URL="wss://jetstream.example.com"
```

3. Run the example:

```bash
go run examples/jwt-rotation-integration.go
```

## Expected Output

With single-key mode:

```
✓ JWT service initialized with single key
  - Current secret: [REDACTED]
✓ Generated access token: eyJhbGciOiJIUzI1NiI...
✓ Generated refresh token: eyJhbGciOiJIUzI1NiI...
✓ Token validated successfully
  - User ID: user-123
  - DID: did:web:example.com
  - Token Type: access
```

With dual-key mode (rotation):

```
✓ JWT service initialized with rotation support
  - Current secret: [REDACTED]
  - Previous secret: [REDACTED]
  - Old tokens will remain valid during rotation window
✓ Generated access token: eyJhbGciOiJIUzI1NiI...
✓ Generated refresh token: eyJhbGciOiJIUzI1NiI...
✓ Token validated successfully
  - User ID: user-123
  - DID: did:web:example.com
  - Token Type: access
```

## Integration in Your Code

### Before (Legacy Single-Key)

```go
import "github.com/onnwee/subcults/internal/auth"

// Old way: directly use JWT_SECRET
jwtService := auth.NewJWTService(cfg.JWTSecret)
```

### After (Rotation-Aware)

```go
import (
    "github.com/onnwee/subcults/internal/auth"
    "github.com/onnwee/subcults/internal/config"
)

// New way: use config helper for rotation support
currentSecret, previousSecret := cfg.GetJWTSecrets()
jwtService := auth.NewJWTServiceWithRotation(currentSecret, previousSecret)
```

The new approach is **100% backward compatible**. If `previousSecret` is empty, it behaves exactly like the legacy single-key setup.

## See Also

- [JWT Rotation Guide](../docs/JWT_ROTATION_GUIDE.md) - Complete rotation documentation
- [Rotation Script](../scripts/rotate-jwt-secret.sh) - Automated secret generation
