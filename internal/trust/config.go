// Package trust provides trust score computation for scenes based on
// membership and alliance relationships.
package trust

import "sync"

// configCache holds the cached configuration state for trust ranking.
var configCache struct {
	mu      sync.RWMutex
	enabled *bool
}

// SetRankingEnabled sets the trust ranking feature flag state.
// This should be called once during application initialization.
// Thread-safe via mutex.
func SetRankingEnabled(enabled bool) {
	configCache.mu.Lock()
	defer configCache.mu.Unlock()
	configCache.enabled = &enabled
}

// IsRankingEnabled returns whether trust-weighted ranking is enabled.
// Returns false if not initialized (safe default).
// The value is cached after the first call to SetRankingEnabled.
// Thread-safe via mutex.
func IsRankingEnabled() bool {
	configCache.mu.RLock()
	defer configCache.mu.RUnlock()
	if configCache.enabled == nil {
		return false // Safe default when not initialized
	}
	return *configCache.enabled
}
