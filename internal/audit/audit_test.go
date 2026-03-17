package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
)

func TestInMemoryRepository_LogAccess(t *testing.T) {
	repo := NewInMemoryRepository()

	entry := LogEntry{
		UserDID:    "did:web:example.com:user123",
		EntityType: "scene",
		EntityID:   "scene-123",
		Action:     "access_precise_location",
		RequestID:  "req-456",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	log, err := repo.LogAccess(entry)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// Verify returned log has all fields
	if log.ID == "" {
		t.Error("LogAccess() should generate an ID")
	}
	if log.UserDID != entry.UserDID {
		t.Errorf("LogAccess() UserDID = %q, want %q", log.UserDID, entry.UserDID)
	}
	if log.EntityType != entry.EntityType {
		t.Errorf("LogAccess() EntityType = %q, want %q", log.EntityType, entry.EntityType)
	}
	if log.EntityID != entry.EntityID {
		t.Errorf("LogAccess() EntityID = %q, want %q", log.EntityID, entry.EntityID)
	}
	if log.Action != entry.Action {
		t.Errorf("LogAccess() Action = %q, want %q", log.Action, entry.Action)
	}
	if log.RequestID != entry.RequestID {
		t.Errorf("LogAccess() RequestID = %q, want %q", log.RequestID, entry.RequestID)
	}
	if log.IPAddress != entry.IPAddress {
		t.Errorf("LogAccess() IPAddress = %q, want %q", log.IPAddress, entry.IPAddress)
	}
	if log.UserAgent != entry.UserAgent {
		t.Errorf("LogAccess() UserAgent = %q, want %q", log.UserAgent, entry.UserAgent)
	}
	if log.CreatedAt.IsZero() {
		t.Error("LogAccess() should set CreatedAt timestamp")
	}

	// Verify timestamp is recent (within last 5 seconds)
	if time.Since(log.CreatedAt) > 5*time.Second {
		t.Error("LogAccess() CreatedAt should be recent")
	}
}

func TestInMemoryRepository_QueryByEntity(t *testing.T) {
	repo := NewInMemoryRepository()

	// Insert multiple logs for different entities
	entries := []LogEntry{
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "access_precise_location"},
		{UserDID: "user2", EntityType: "scene", EntityID: "scene-1", Action: "view_details"},
		{UserDID: "user3", EntityType: "scene", EntityID: "scene-2", Action: "access_precise_location"},
		{UserDID: "user1", EntityType: "event", EntityID: "event-1", Action: "access_precise_location"},
		{UserDID: "user4", EntityType: "scene", EntityID: "scene-1", Action: "access_precise_location"},
	}

	for _, entry := range entries {
		_, err := repo.LogAccess(entry)
		if err != nil {
			t.Fatalf("LogAccess() error = %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Query for scene-1 logs
	results, err := repo.QueryByEntity("scene", "scene-1", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	// Should return 3 logs for scene-1
	if len(results) != 3 {
		t.Errorf("QueryByEntity() returned %d logs, want 3", len(results))
	}

	// Verify results are sorted by time (newest first)
	for i := 0; i < len(results)-1; i++ {
		if results[i].CreatedAt.Before(results[i+1].CreatedAt) {
			t.Error("QueryByEntity() results should be sorted by time (newest first)")
		}
	}

	// Verify all results match the query
	for _, log := range results {
		if log.EntityType != "scene" || log.EntityID != "scene-1" {
			t.Errorf("QueryByEntity() returned log with EntityType=%q, EntityID=%q, want scene/scene-1",
				log.EntityType, log.EntityID)
		}
	}
}

func TestInMemoryRepository_QueryByEntity_WithLimit(t *testing.T) {
	repo := NewInMemoryRepository()

	// Insert 5 logs for the same entity
	for i := 0; i < 5; i++ {
		entry := LogEntry{
			UserDID:    "user1",
			EntityType: "scene",
			EntityID:   "scene-1",
			Action:     "access_precise_location",
		}
		_, err := repo.LogAccess(entry)
		if err != nil {
			t.Fatalf("LogAccess() error = %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Query with limit=2
	results, err := repo.QueryByEntity("scene", "scene-1", 2)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("QueryByEntity(limit=2) returned %d logs, want 2", len(results))
	}
}

func TestInMemoryRepository_QueryByUser(t *testing.T) {
	repo := NewInMemoryRepository()

	// Insert multiple logs for different users
	entries := []LogEntry{
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "access_precise_location"},
		{UserDID: "user2", EntityType: "scene", EntityID: "scene-1", Action: "view_details"},
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-2", Action: "access_precise_location"},
		{UserDID: "user1", EntityType: "event", EntityID: "event-1", Action: "access_precise_location"},
		{UserDID: "user3", EntityType: "scene", EntityID: "scene-1", Action: "access_precise_location"},
	}

	for _, entry := range entries {
		_, err := repo.LogAccess(entry)
		if err != nil {
			t.Fatalf("LogAccess() error = %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Query for user1 logs
	results, err := repo.QueryByUser("user1", 0)
	if err != nil {
		t.Fatalf("QueryByUser() error = %v", err)
	}

	// Should return 3 logs for user1
	if len(results) != 3 {
		t.Errorf("QueryByUser() returned %d logs, want 3", len(results))
	}

	// Verify results are sorted by time (newest first)
	for i := 0; i < len(results)-1; i++ {
		if results[i].CreatedAt.Before(results[i+1].CreatedAt) {
			t.Error("QueryByUser() results should be sorted by time (newest first)")
		}
	}

	// Verify all results match the query
	for _, log := range results {
		if log.UserDID != "user1" {
			t.Errorf("QueryByUser() returned log with UserDID=%q, want user1", log.UserDID)
		}
	}
}

func TestInMemoryRepository_QueryByUser_WithLimit(t *testing.T) {
	repo := NewInMemoryRepository()

	// Insert 5 logs for the same user
	for i := 0; i < 5; i++ {
		entry := LogEntry{
			UserDID:    "user1",
			EntityType: "scene",
			EntityID:   "scene-1",
			Action:     "access_precise_location",
		}
		_, err := repo.LogAccess(entry)
		if err != nil {
			t.Fatalf("LogAccess() error = %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Query with limit=3
	results, err := repo.QueryByUser("user1", 3)
	if err != nil {
		t.Fatalf("QueryByUser() error = %v", err)
	}

	if len(results) != 3 {
		t.Errorf("QueryByUser(limit=3) returned %d logs, want 3", len(results))
	}
}

func TestInMemoryRepository_QueryByEntity_NoResults(t *testing.T) {
	repo := NewInMemoryRepository()

	results, err := repo.QueryByEntity("scene", "nonexistent", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("QueryByEntity() for nonexistent entity returned %d logs, want 0", len(results))
	}
}

func TestInMemoryRepository_QueryByUser_NoResults(t *testing.T) {
	repo := NewInMemoryRepository()

	results, err := repo.QueryByUser("nonexistent", 0)
	if err != nil {
		t.Fatalf("QueryByUser() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("QueryByUser() for nonexistent user returned %d logs, want 0", len(results))
	}
}

func TestLogAccess_WithContext(t *testing.T) {
	repo := NewInMemoryRepository()

	// Create a test HTTP request to set request ID properly through middleware
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(middleware.RequestIDHeader, "req-789")

	// Run through middleware to set request ID in context
	var ctx context.Context
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Set user DID in context
	ctx = middleware.SetUserDID(ctx, "did:web:test.com:user123")

	err := LogAccess(ctx, repo, "scene", "scene-123", "access_precise_location", "")
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	// Verify the log was created with context values
	results, err := repo.QueryByEntity("scene", "scene-123", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(results))
	}

	log := results[0]
	if log.UserDID != "did:web:test.com:user123" {
		t.Errorf("LogAccess() UserDID = %q, want did:web:test.com:user123", log.UserDID)
	}
	if log.RequestID != "req-789" {
		t.Errorf("LogAccess() RequestID = %q, want req-789", log.RequestID)
	}
	if log.EntityType != "scene" {
		t.Errorf("LogAccess() EntityType = %q, want scene", log.EntityType)
	}
	if log.EntityID != "scene-123" {
		t.Errorf("LogAccess() EntityID = %q, want scene-123", log.EntityID)
	}
	if log.Action != "access_precise_location" {
		t.Errorf("LogAccess() Action = %q, want access_precise_location", log.Action)
	}
}

func TestLogAccessFromRequest(t *testing.T) {
	repo := NewInMemoryRepository()

	// Create a test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/scene-123", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set(middleware.RequestIDHeader, "req-abc")
	req.RemoteAddr = "192.168.1.100:12345"

	// Run through middleware to set request ID in context
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set user DID in context
		ctx := middleware.SetUserDID(r.Context(), "did:web:test.com:user456")
		req = r.WithContext(ctx)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	err := LogAccessFromRequest(req, repo, "scene", "scene-123", "access_precise_location", "")
	if err != nil {
		t.Fatalf("LogAccessFromRequest() error = %v", err)
	}

	// Verify the log was created with request metadata
	results, err := repo.QueryByEntity("scene", "scene-123", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(results))
	}

	log := results[0]
	if log.UserDID != "did:web:test.com:user456" {
		t.Errorf("LogAccessFromRequest() UserDID = %q, want did:web:test.com:user456", log.UserDID)
	}
	if log.RequestID != "req-abc" {
		t.Errorf("LogAccessFromRequest() RequestID = %q, want req-abc", log.RequestID)
	}
	// IP address should have port stripped
	if log.IPAddress != "192.168.1.100" {
		t.Errorf("LogAccessFromRequest() IPAddress = %q, want 192.168.1.100 (port stripped)", log.IPAddress)
	}
	if log.UserAgent != "TestAgent/1.0" {
		t.Errorf("LogAccessFromRequest() UserAgent = %q, want TestAgent/1.0", log.UserAgent)
	}
}

func TestLogAccessFromRequest_WithXForwardedFor(t *testing.T) {
	repo := NewInMemoryRepository()

	// Create a test HTTP request with X-Forwarded-For header containing multiple IPs
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/scene-123", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 198.51.100.178, 192.0.2.1")
	req.RemoteAddr = "192.168.1.100:12345"

	ctx := middleware.SetUserDID(req.Context(), "did:web:test.com:user789")
	req = req.WithContext(ctx)

	err := LogAccessFromRequest(req, repo, "scene", "scene-123", "access_precise_location", "")
	if err != nil {
		t.Fatalf("LogAccessFromRequest() error = %v", err)
	}

	results, err := repo.QueryByEntity("scene", "scene-123", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(results))
	}

	log := results[0]
	// X-Forwarded-For should use first IP (original client)
	if log.IPAddress != "203.0.113.195" {
		t.Errorf("LogAccessFromRequest() IPAddress = %q, want 203.0.113.195 (first IP from X-Forwarded-For)", log.IPAddress)
	}
}

func TestLogAccessFromRequest_WithEmptyXForwardedFor(t *testing.T) {
	repo := NewInMemoryRepository()

	// Create a test HTTP request with empty X-Forwarded-For header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/scene-456", nil)
	req.Header.Set("X-Forwarded-For", "  ,  ")
	req.RemoteAddr = "192.168.1.100:12345"

	ctx := middleware.SetUserDID(req.Context(), "did:web:test.com:user789")
	req = req.WithContext(ctx)

	err := LogAccessFromRequest(req, repo, "scene", "scene-456", "access_precise_location", "")
	if err != nil {
		t.Fatalf("LogAccessFromRequest() error = %v", err)
	}

	results, err := repo.QueryByEntity("scene", "scene-456", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(results))
	}

	log := results[0]
	// Should fall back to RemoteAddr with port stripped when X-Forwarded-For is empty
	if log.IPAddress != "192.168.1.100" {
		t.Errorf("LogAccessFromRequest() IPAddress = %q, want 192.168.1.100 (from RemoteAddr, port stripped)", log.IPAddress)
	}
}

func TestLogAccessFromRequest_WithXRealIP(t *testing.T) {
	repo := NewInMemoryRepository()

	// Create a test HTTP request with X-Real-IP header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/scene-789", nil)
	req.Header.Set("X-Real-IP", "198.51.100.50")
	req.RemoteAddr = "192.168.1.100:12345"

	ctx := middleware.SetUserDID(req.Context(), "did:web:test.com:user999")
	req = req.WithContext(ctx)

	err := LogAccessFromRequest(req, repo, "scene", "scene-789", "access_precise_location", "")
	if err != nil {
		t.Fatalf("LogAccessFromRequest() error = %v", err)
	}

	results, err := repo.QueryByEntity("scene", "scene-789", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(results))
	}

	log := results[0]
	// X-Real-IP should be used when X-Forwarded-For is not present
	if log.IPAddress != "198.51.100.50" {
		t.Errorf("LogAccessFromRequest() IPAddress = %q, want 198.51.100.50 (from X-Real-IP)", log.IPAddress)
	}
}

func TestLogAccessFromRequest_WithXRealIPAndPort(t *testing.T) {
	repo := NewInMemoryRepository()

	// Create a test HTTP request with X-Real-IP header containing port
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/scene-890", nil)
	req.Header.Set("X-Real-IP", "198.51.100.60:8080")
	req.RemoteAddr = "192.168.1.100:12345"

	ctx := middleware.SetUserDID(req.Context(), "did:web:test.com:user1000")
	req = req.WithContext(ctx)

	err := LogAccessFromRequest(req, repo, "scene", "scene-890", "access_precise_location", "")
	if err != nil {
		t.Fatalf("LogAccessFromRequest() error = %v", err)
	}

	results, err := repo.QueryByEntity("scene", "scene-890", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(results))
	}

	log := results[0]
	// Port should be stripped from X-Real-IP
	if log.IPAddress != "198.51.100.60" {
		t.Errorf("LogAccessFromRequest() IPAddress = %q, want 198.51.100.60 (port stripped from X-Real-IP)", log.IPAddress)
	}
}

func TestLogAccessFromRequest_WithXForwardedForAndPort(t *testing.T) {
	repo := NewInMemoryRepository()

	// Create a test HTTP request with X-Forwarded-For containing port in first IP
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/scene-891", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.200:9000, 198.51.100.178")
	req.RemoteAddr = "192.168.1.100:12345"

	ctx := middleware.SetUserDID(req.Context(), "did:web:test.com:user1001")
	req = req.WithContext(ctx)

	err := LogAccessFromRequest(req, repo, "scene", "scene-891", "access_precise_location", "")
	if err != nil {
		t.Fatalf("LogAccessFromRequest() error = %v", err)
	}

	results, err := repo.QueryByEntity("scene", "scene-891", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(results))
	}

	log := results[0]
	// Port should be stripped from first IP in X-Forwarded-For
	if log.IPAddress != "203.0.113.200" {
		t.Errorf("LogAccessFromRequest() IPAddress = %q, want 203.0.113.200 (port stripped from X-Forwarded-For)", log.IPAddress)
	}
}

func TestInMemoryRepository_ThreadSafety(t *testing.T) {
	repo := NewInMemoryRepository()

	// Run concurrent LogAccess operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			entry := LogEntry{
				UserDID:    "user1",
				EntityType: "scene",
				EntityID:   "scene-1",
				Action:     "access_precise_location",
			}
			_, err := repo.LogAccess(entry)
			if err != nil {
				t.Errorf("LogAccess() error = %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all logs were recorded
	results, err := repo.QueryByEntity("scene", "scene-1", 0)
	if err != nil {
		t.Fatalf("QueryByEntity() error = %v", err)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 log entries after concurrent writes, got %d", len(results))
	}
}

func TestLogAccess_NilRepository(t *testing.T) {
	ctx := context.Background()

	err := LogAccess(ctx, nil, "scene", "scene-123", "access_precise_location", "")
	if err != ErrNilRepository {
		t.Errorf("LogAccess() with nil repo error = %v, want %v", err, ErrNilRepository)
	}
}

func TestLogAccessFromRequest_NilRepository(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	err := LogAccessFromRequest(req, nil, "scene", "scene-123", "access_precise_location", "")
	if err != ErrNilRepository {
		t.Errorf("LogAccessFromRequest() with nil repo error = %v, want %v", err, ErrNilRepository)
	}
}

func TestLogAccess_EmptyEntityType(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	err := LogAccess(ctx, repo, "", "scene-123", "access_precise_location", "")
	if err != ErrInvalidEntityType {
		t.Errorf("LogAccess() with empty entityType error = %v, want %v", err, ErrInvalidEntityType)
	}
}

func TestLogAccess_InvalidEntityType(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	err := LogAccess(ctx, repo, "invalid_type", "scene-123", "access_precise_location", "")
	if err != ErrInvalidEntityType {
		t.Errorf("LogAccess() with invalid entityType error = %v, want %v", err, ErrInvalidEntityType)
	}
}

func TestLogAccess_EmptyEntityID(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	err := LogAccess(ctx, repo, "scene", "", "access_precise_location", "")
	if err != ErrInvalidEntityID {
		t.Errorf("LogAccess() with empty entityID error = %v, want %v", err, ErrInvalidEntityID)
	}
}

func TestLogAccess_EmptyAction(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	err := LogAccess(ctx, repo, "scene", "scene-123", "", "")
	if err != ErrInvalidAction {
		t.Errorf("LogAccess() with empty action error = %v, want %v", err, ErrInvalidAction)
	}
}

func TestLogAccess_InvalidAction(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	err := LogAccess(ctx, repo, "scene", "scene-123", "invalid_action", "")
	if err != ErrInvalidAction {
		t.Errorf("LogAccess() with invalid action error = %v, want %v", err, ErrInvalidAction)
	}
}

func TestLogAccessFromRequest_ValidationErrors(t *testing.T) {
	repo := NewInMemoryRepository()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	tests := []struct {
		name       string
		entityType string
		entityID   string
		action     string
		wantErr    error
	}{
		{
			name:       "empty entity type",
			entityType: "",
			entityID:   "id-123",
			action:     "access_precise_location",
			wantErr:    ErrInvalidEntityType,
		},
		{
			name:       "invalid entity type",
			entityType: "bad_type",
			entityID:   "id-123",
			action:     "access_precise_location",
			wantErr:    ErrInvalidEntityType,
		},
		{
			name:       "empty entity ID",
			entityType: "scene",
			entityID:   "",
			action:     "access_precise_location",
			wantErr:    ErrInvalidEntityID,
		},
		{
			name:       "empty action",
			entityType: "scene",
			entityID:   "id-123",
			action:     "",
			wantErr:    ErrInvalidAction,
		},
		{
			name:       "invalid action",
			entityType: "scene",
			entityID:   "id-123",
			action:     "bad_action",
			wantErr:    ErrInvalidAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LogAccessFromRequest(req, repo, tt.entityType, tt.entityID, tt.action, "")
			if err != tt.wantErr {
				t.Errorf("LogAccessFromRequest() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
