package idempotency

import (
	"testing"
	"time"
)

func TestCleanupOldKeys(t *testing.T) {
	repo := NewInMemoryRepository()

	// Store keys with different timestamps
	oldTime := time.Now().Add(-25 * time.Hour)
	recentTime := time.Now().Add(-1 * time.Hour)

	oldKey := &IdempotencyKey{
		Key:                "old-key",
		Method:             "POST",
		Route:              "/test",
		CreatedAt:          oldTime,
		ResponseHash:       "abc123",
		Status:             StatusCompleted,
		ResponseBody:       `{"result":"ok"}`,
		ResponseStatusCode: 200,
	}

	recentKey := &IdempotencyKey{
		Key:                "recent-key",
		Method:             "POST",
		Route:              "/test",
		CreatedAt:          recentTime,
		ResponseHash:       "def456",
		Status:             StatusCompleted,
		ResponseBody:       `{"result":"ok"}`,
		ResponseStatusCode: 200,
	}

	if err := repo.Store(oldKey); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if err := repo.Store(recentKey); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Cleanup keys older than default expiry (24 hours)
	deleted, err := CleanupOldKeys(repo, DefaultExpiry)
	if err != nil {
		t.Fatalf("CleanupOldKeys() error = %v", err)
	}

	if deleted != 1 {
		t.Errorf("CleanupOldKeys() deleted = %d, want 1", deleted)
	}

	// Old key should be gone
	_, err = repo.Get("old-key")
	if err != ErrKeyNotFound {
		t.Errorf("Get() old key error = %v, want %v", err, ErrKeyNotFound)
	}

	// Recent key should still exist
	_, err = repo.Get("recent-key")
	if err != nil {
		t.Errorf("Get() recent key error = %v, want nil", err)
	}
}

func TestCleanupOldKeys_NoKeys(t *testing.T) {
	repo := NewInMemoryRepository()

	deleted, err := CleanupOldKeys(repo, DefaultExpiry)
	if err != nil {
		t.Fatalf("CleanupOldKeys() error = %v", err)
	}

	if deleted != 0 {
		t.Errorf("CleanupOldKeys() deleted = %d, want 0", deleted)
	}
}

func TestRunPeriodicCleanup_Stop(t *testing.T) {
	repo := NewInMemoryRepository()

	// Store an old key
	oldTime := time.Now().Add(-25 * time.Hour)
	oldKey := &IdempotencyKey{
		Key:                "old-key",
		Method:             "POST",
		Route:              "/test",
		CreatedAt:          oldTime,
		ResponseHash:       "abc123",
		Status:             StatusCompleted,
		ResponseBody:       `{"result":"ok"}`,
		ResponseStatusCode: 200,
	}

	if err := repo.Store(oldKey); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	stopChan := make(chan struct{})

	// Run periodic cleanup in background
	done := make(chan struct{})
	go func() {
		RunPeriodicCleanup(repo, 100*time.Millisecond, DefaultExpiry, stopChan)
		close(done)
	}()

	// Wait a bit for initial cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Old key should be cleaned up
	_, err := repo.Get("old-key")
	if err != ErrKeyNotFound {
		t.Errorf("Get() old key error = %v, want %v", err, ErrKeyNotFound)
	}

	// Stop the cleanup
	close(stopChan)

	// Wait for cleanup to stop
	select {
	case <-done:
		// Success - cleanup stopped
	case <-time.After(1 * time.Second):
		t.Fatal("RunPeriodicCleanup() did not stop within timeout")
	}
}
