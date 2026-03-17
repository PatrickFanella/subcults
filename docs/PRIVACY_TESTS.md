# Privacy Test Suite

This document describes the privacy-focused test suite for Subcults, ensuring that location data and user privacy are protected.

## Overview

The privacy test suite (`privacy_test.go`) validates that the platform's privacy-first architecture functions correctly and prevents accidental location data leaks.

## Test Categories

### 1. Location Consent Enforcement

Tests that validate scenes and events respect the `allow_precise` consent flag:

- **TestPrivacy_SceneFetch_WithoutConsent**: Ensures scenes without consent never expose precise coordinates
- **TestPrivacy_SceneFetch_WithConsent**: Validates that scenes with consent do expose precise coordinates when requested
- **TestPrivacy_EventFetch_WithoutConsent**: Ensures events without consent never expose precise coordinates
- **TestPrivacy_EventFetch_WithConsent**: Validates that events with consent do expose precise coordinates when requested

**Critical Assertion**: `fetched.PrecisePoint` must be `nil` when `allow_precise = false`

### 2. Geohash Precision

Tests that validate coarse geohash handling for public display:

- **TestPrivacy_GeohashPrecision**: Validates that geohashes are truncated to the default precision (6 characters, ~±0.61 km accuracy)

This ensures that even if precise location data exists, only coarse location is exposed for public discovery.

### 3. Consent Lifecycle

Tests that validate consent can be granted, revoked, and updated:

- **TestPrivacy_ConsentRevocation**: Validates that removing consent immediately removes precise coordinates from storage
- **TestPrivacy_Upsert_PreservesConsent**: Validates that upsert operations respect consent on both insert and update paths

**Key Behavior**: Consent revocation must immediately remove precise coordinates, not just hide them.

### 4. Batch Operations

Tests that validate privacy is maintained in bulk scenarios:

- **TestPrivacy_MultipleScenes_MixedConsent**: Validates that in a batch scenario, consent is enforced independently for each scene

This prevents "consent bleeding" where one scene's consent affects another.

### 5. API Documentation

Tests that document expected behavior:

- **TestPrivacy_CoarseGeohash_PublicAPI**: Documents that public API should return coarse geohash for scenes without precise consent

This is a living documentation test showing the expected JSON structure.

### 6. Planned Features (Placeholder Tests)

Tests marked with `t.Skip()` for features not yet implemented:

- **TestPrivacy_EXIF_Placeholder**: Placeholder for EXIF metadata stripping (tracked in Privacy & Safety Epic #6)
- **TestPrivacy_LocationJitter_Placeholder**: Placeholder for location jitter implementation (tracked in Privacy & Safety Epic #6)

These tests include detailed comments describing the expected behavior when implemented.

## Running the Tests

### Run only privacy tests:
```bash
go test ./internal/scene -run TestPrivacy
```

### Run all scene tests (including privacy):
```bash
go test ./internal/scene/...
```

### Run with coverage:
```bash
go test -cover ./internal/scene/...
```

### Run with race detector:
```bash
go test -race ./internal/scene/...
```

## Expected Test Results

All implemented privacy tests should **PASS**:
- ✅ TestPrivacy_SceneFetch_WithoutConsent
- ✅ TestPrivacy_SceneFetch_WithConsent
- ✅ TestPrivacy_EventFetch_WithoutConsent
- ✅ TestPrivacy_EventFetch_WithConsent
- ✅ TestPrivacy_GeohashPrecision
- ✅ TestPrivacy_ConsentRevocation
- ✅ TestPrivacy_MultipleScenes_MixedConsent
- ✅ TestPrivacy_Upsert_PreservesConsent
- ✅ TestPrivacy_CoarseGeohash_PublicAPI

Placeholder tests should **SKIP** (until features are implemented):
- ⏭️ TestPrivacy_EXIF_Placeholder
- ⏭️ TestPrivacy_LocationJitter_Placeholder

## Privacy Guarantees Validated

1. ✅ **No Precise Location Leak**: Scenes/events without consent never expose precise coordinates
2. ✅ **Coarse Geohash Only**: Public APIs return only coarse geohash (~±0.61 km precision)
3. ✅ **Consent Enforcement**: Repository layer enforces location consent before persistence
4. ✅ **Consent Revocation**: Removing consent immediately removes precise coordinates
5. ✅ **Independent Consent**: Each scene/event has independent consent control
6. ⏳ **EXIF Stripping**: Not yet implemented (tracked in Epic #6)
7. ⏳ **Location Jitter**: Not yet implemented (tracked in Epic #6)

## Adding New Privacy Tests

When adding privacy features:

1. Add tests to `privacy_test.go` following the naming convention `TestPrivacy_*`
2. Use table-driven tests for multiple scenarios
3. Include clear error messages that indicate privacy violations
4. Document expected behavior in test comments
5. Update this README with the new test

Example test structure:
```go
func TestPrivacy_NewFeature(t *testing.T) {
    tests := []struct {
        name string
        // test fields
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Critical privacy assertion with clear error message
            if privacyViolation {
                t.Errorf("Privacy violation: description of what leaked")
            }
        })
    }
}
```

## Related Documentation

- [PRIVACY.md](/docs/PRIVACY.md): Technical privacy overview
- [Privacy & Safety Epic #6](https://github.com/subculture-collective/subcults/issues/6): Planned privacy features
- [Repository Instructions](/.github/copilot-instructions.md): Code conventions including privacy

## Continuous Integration

These tests are run in CI via:
```bash
make test
```

All privacy tests must pass before merging to main branch. Any test failures indicate a potential privacy regression and must be investigated immediately.

## Contact

For privacy concerns or questions about tests:
- Open a GitHub issue with the `privacy` label
- See `SECURITY.md` for security vulnerability reporting (when available)
