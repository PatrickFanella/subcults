package trust

import (
	"sync"
	"testing"
)

func TestSetRankingEnabled(t *testing.T) {
	// Reset cache before test
	configCache.mu.Lock()
	configCache.enabled = nil
	configCache.mu.Unlock()

	// Test setting to true
	SetRankingEnabled(true)
	if !IsRankingEnabled() {
		t.Error("IsRankingEnabled() = false, want true after SetRankingEnabled(true)")
	}

	// Test setting to false
	SetRankingEnabled(false)
	if IsRankingEnabled() {
		t.Error("IsRankingEnabled() = true, want false after SetRankingEnabled(false)")
	}

	// Test setting to true again
	SetRankingEnabled(true)
	if !IsRankingEnabled() {
		t.Error("IsRankingEnabled() = false, want true after SetRankingEnabled(true)")
	}
}

func TestIsRankingEnabled_NotInitialized(t *testing.T) {
	// Reset cache to simulate uninitialized state
	configCache.mu.Lock()
	configCache.enabled = nil
	configCache.mu.Unlock()

	// Should return false when not initialized
	if IsRankingEnabled() {
		t.Error("IsRankingEnabled() = true, want false when not initialized")
	}
}

func TestIsRankingEnabled_ThreadSafe(t *testing.T) {
	// Reset cache before test
	configCache.mu.Lock()
	configCache.enabled = nil
	configCache.mu.Unlock()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Start multiple goroutines reading and writing concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			SetRankingEnabled(true)
		}()

		go func() {
			defer wg.Done()
			_ = IsRankingEnabled()
		}()
	}

	wg.Wait()

	// Should not panic and should have a valid state
	enabled := IsRankingEnabled()
	// The final state should be true since we're setting it to true
	if !enabled {
		t.Error("IsRankingEnabled() = false, want true after concurrent writes")
	}
}

func TestIsRankingEnabled_Caching(t *testing.T) {
	// Reset cache before test
	configCache.mu.Lock()
	configCache.enabled = nil
	configCache.mu.Unlock()

	// Set value
	SetRankingEnabled(true)

	// Read multiple times - should return cached value
	for i := 0; i < 5; i++ {
		if !IsRankingEnabled() {
			t.Errorf("IsRankingEnabled() call %d = false, want true", i+1)
		}
	}

	// Change value
	SetRankingEnabled(false)

	// Read multiple times - should return new cached value
	for i := 0; i < 5; i++ {
		if IsRankingEnabled() {
			t.Errorf("IsRankingEnabled() call %d = true, want false", i+1)
		}
	}
}
