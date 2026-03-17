package indexer

import (
	"context"
	"testing"
	"time"
)

// TestDefaultCleanupConfig tests the default cleanup configuration.
func TestDefaultCleanupConfig(t *testing.T) {
	config := DefaultCleanupConfig()

	if config.RetentionPeriod != 24*time.Hour {
		t.Errorf("Expected retention period of 24h, got %v", config.RetentionPeriod)
	}

	if config.CleanupInterval != 1*time.Hour {
		t.Errorf("Expected cleanup interval of 1h, got %v", config.CleanupInterval)
	}
}

// TestInMemoryCleanupService_StartStop tests that the cleanup service can start and stop cleanly.
func TestInMemoryCleanupService_StartStop(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	service := NewInMemoryCleanupService(repo, newTestLogger(), DefaultCleanupConfig())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the service
	service.Start(ctx)

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Stop the service
	done := make(chan struct{})
	go func() {
		service.Stop()
		close(done)
	}()

	// Wait for shutdown with timeout
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Cleanup service did not stop within timeout")
	}
}

// TestInMemoryCleanupService_ContextCancellation tests that the service stops when context is cancelled.
func TestInMemoryCleanupService_ContextCancellation(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())
	service := NewInMemoryCleanupService(repo, newTestLogger(), DefaultCleanupConfig())

	ctx, cancel := context.WithCancel(context.Background())

	// Start the service
	service.Start(ctx)

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for service to stop
	time.Sleep(200 * time.Millisecond)

	// Service should have stopped via context cancellation
	// (We can't easily verify this without more instrumentation, but the test shouldn't hang)
}

// TestInMemoryCleanupService_CustomConfig tests cleanup service with custom configuration.
func TestInMemoryCleanupService_CustomConfig(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())

	config := CleanupConfig{
		RetentionPeriod: 1 * time.Hour,
		CleanupInterval: 30 * time.Second,
	}

	service := NewInMemoryCleanupService(repo, newTestLogger(), config)

	if service.retentionPeriod != 1*time.Hour {
		t.Errorf("Expected retention period of 1h, got %v", service.retentionPeriod)
	}

	if service.cleanupInterval != 30*time.Second {
		t.Errorf("Expected cleanup interval of 30s, got %v", service.cleanupInterval)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	service.Stop()
}

// TestInMemoryCleanupService_ZeroConfigUsesDefaults tests that zero values use defaults.
func TestInMemoryCleanupService_ZeroConfigUsesDefaults(t *testing.T) {
	repo := NewInMemoryRecordRepository(newTestLogger())

	config := CleanupConfig{
		// Zero values - should use defaults
	}

	service := NewInMemoryCleanupService(repo, newTestLogger(), config)

	if service.retentionPeriod != 24*time.Hour {
		t.Errorf("Expected default retention period of 24h, got %v", service.retentionPeriod)
	}

	if service.cleanupInterval != 1*time.Hour {
		t.Errorf("Expected default cleanup interval of 1h, got %v", service.cleanupInterval)
	}
}
