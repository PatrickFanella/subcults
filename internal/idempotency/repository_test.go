package idempotency

import (
	"testing"
	"time"
)

func TestInMemoryRepository_Get(t *testing.T) {
	repo := NewInMemoryRepository()

	// Test key not found
	_, err := repo.Get("nonexistent")
	if err != ErrKeyNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrKeyNotFound)
	}

	// Store a key
	key := &IdempotencyKey{
		Key:                "test-key",
		Method:             "POST",
		Route:              "/test",
		ResponseHash:       "abc123",
		Status:             StatusCompleted,
		ResponseBody:       `{"result":"ok"}`,
		ResponseStatusCode: 200,
	}
	if err := repo.Store(key); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Retrieve the key
	retrieved, err := repo.Get("test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Key != key.Key {
		t.Errorf("Get() Key = %v, want %v", retrieved.Key, key.Key)
	}
	if retrieved.Method != key.Method {
		t.Errorf("Get() Method = %v, want %v", retrieved.Method, key.Method)
	}
	if retrieved.ResponseBody != key.ResponseBody {
		t.Errorf("Get() ResponseBody = %v, want %v", retrieved.ResponseBody, key.ResponseBody)
	}
}

func TestInMemoryRepository_Store(t *testing.T) {
	repo := NewInMemoryRepository()

	key := &IdempotencyKey{
		Key:                "test-key",
		Method:             "POST",
		Route:              "/test",
		ResponseHash:       "abc123",
		Status:             StatusCompleted,
		ResponseBody:       `{"result":"ok"}`,
		ResponseStatusCode: 200,
	}

	// First store should succeed
	if err := repo.Store(key); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Duplicate store should fail
	err := repo.Store(key)
	if err != ErrKeyExists {
		t.Errorf("Store() duplicate error = %v, want %v", err, ErrKeyExists)
	}
}

func TestInMemoryRepository_Store_InvalidKey(t *testing.T) {
	repo := NewInMemoryRepository()

	tests := []struct {
		name      string
		key       string
		expectErr error
	}{
		{
			name:      "empty key",
			key:       "",
			expectErr: ErrInvalidKey,
		},
		{
			name:      "key too long",
			key:       string(make([]byte, MaxKeyLength+1)),
			expectErr: ErrKeyTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &IdempotencyKey{
				Key:                tt.key,
				Method:             "POST",
				Route:              "/test",
				ResponseHash:       "abc123",
				Status:             StatusCompleted,
				ResponseBody:       `{"result":"ok"}`,
				ResponseStatusCode: 200,
			}

			err := repo.Store(record)
			if err != tt.expectErr {
				t.Errorf("Store() error = %v, want %v", err, tt.expectErr)
			}
		})
	}
}

func TestInMemoryRepository_Store_SetsCreatedAt(t *testing.T) {
	repo := NewInMemoryRepository()

	key := &IdempotencyKey{
		Key:                "test-key",
		Method:             "POST",
		Route:              "/test",
		ResponseHash:       "abc123",
		Status:             StatusCompleted,
		ResponseBody:       `{"result":"ok"}`,
		ResponseStatusCode: 200,
		// CreatedAt is zero value
	}

	if err := repo.Store(key); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Retrieve and check CreatedAt was set
	retrieved, err := repo.Get("test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.CreatedAt.IsZero() {
		t.Error("Store() should set CreatedAt but it's still zero")
	}
}

func TestInMemoryRepository_DeleteOlderThan(t *testing.T) {
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

	// Delete keys older than 24 hours
	deleted, err := repo.DeleteOlderThan(24 * time.Hour)
	if err != nil {
		t.Fatalf("DeleteOlderThan() error = %v", err)
	}

	if deleted != 1 {
		t.Errorf("DeleteOlderThan() deleted = %d, want 1", deleted)
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

func TestInMemoryRepository_Isolation(t *testing.T) {
	repo := NewInMemoryRepository()

	original := &IdempotencyKey{
		Key:                "test-key",
		Method:             "POST",
		Route:              "/test",
		ResponseHash:       "abc123",
		Status:             StatusCompleted,
		ResponseBody:       `{"result":"ok"}`,
		ResponseStatusCode: 200,
	}

	if err := repo.Store(original); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Modify original after storing
	original.ResponseBody = "modified"

	// Retrieve and verify it wasn't affected
	retrieved, err := repo.Get("test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.ResponseBody == "modified" {
		t.Error("External mutation affected stored record - deep copy not working")
	}
}
