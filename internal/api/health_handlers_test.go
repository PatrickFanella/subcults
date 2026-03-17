package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockHealthChecker is a mock implementation of HealthChecker for testing.
type mockHealthChecker struct {
	shouldFail bool
	err        error
}

func (m *mockHealthChecker) HealthCheck(ctx context.Context) error {
	if m.shouldFail {
		if m.err != nil {
			return m.err
		}
		return errors.New("health check failed")
	}
	return nil
}

// TestHealth_Success tests the basic health check endpoint.
func TestHealth_Success(t *testing.T) {
	handlers := NewHealthHandlers(HealthHandlersConfig{
		MetricsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	handlers.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "up" {
		t.Errorf("expected status 'up', got %s", response.Status)
	}

	if response.UptimeS < 0 {
		t.Errorf("expected uptime_s to be >= 0, got %d", response.UptimeS)
	}
}

// TestHealth_MethodNotAllowed tests that non-GET requests are rejected.
func TestHealth_MethodNotAllowed(t *testing.T) {
	handlers := NewHealthHandlers(HealthHandlersConfig{})

	req := httptest.NewRequest(http.MethodPost, "/health/live", nil)
	w := httptest.NewRecorder()

	handlers.Health(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	// Verify it returns structured JSON error
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type to contain application/json, got %s", contentType)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeBadRequest {
		t.Errorf("expected error code %s, got %s", ErrCodeBadRequest, errResp.Error.Code)
	}
}

// TestReady_AllHealthy tests readiness when all services are healthy.
func TestReady_AllHealthy(t *testing.T) {
	dbChecker := &mockHealthChecker{shouldFail: false}
	livekitChecker := &mockHealthChecker{shouldFail: false}
	redisChecker := &mockHealthChecker{shouldFail: false}

	handlers := NewHealthHandlers(HealthHandlersConfig{
		DBChecker:      dbChecker,
		LiveKitChecker: livekitChecker,
		RedisChecker:   redisChecker,
		MetricsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "up" {
		t.Errorf("expected status 'up', got %s", response.Status)
	}

	expectedChecks := map[string]string{
		"db":      "ok",
		"livekit": "ok",
		"redis":   "ok",
	}

	for check, expectedStatus := range expectedChecks {
		if response.Checks[check] != expectedStatus {
			t.Errorf("expected %s check to be %s, got %s", check, expectedStatus, response.Checks[check])
		}
	}

	if response.UptimeS < 0 {
		t.Errorf("expected uptime_s to be >= 0, got %d", response.UptimeS)
	}
}

// TestReady_DatabaseUnhealthy tests readiness when database is unhealthy.
func TestReady_DatabaseUnhealthy(t *testing.T) {
	dbChecker := &mockHealthChecker{shouldFail: true}
	livekitChecker := &mockHealthChecker{shouldFail: false}

	handlers := NewHealthHandlers(HealthHandlersConfig{
		DBChecker:      dbChecker,
		LiveKitChecker: livekitChecker,
		MetricsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %s", response.Status)
	}

	if response.Checks["db"] != "error" {
		t.Errorf("expected db check to be 'error', got %s", response.Checks["db"])
	}

	// Other checks should still be ok
	if response.Checks["livekit"] != "ok" {
		t.Errorf("expected livekit check to be 'ok', got %s", response.Checks["livekit"])
	}
}

// TestReady_LiveKitUnhealthy tests readiness when LiveKit is unhealthy.
func TestReady_LiveKitUnhealthy(t *testing.T) {
	dbChecker := &mockHealthChecker{shouldFail: false}
	livekitChecker := &mockHealthChecker{shouldFail: true}

	handlers := NewHealthHandlers(HealthHandlersConfig{
		DBChecker:      dbChecker,
		LiveKitChecker: livekitChecker,
		MetricsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %s", response.Status)
	}

	if response.Checks["livekit"] != "error" {
		t.Errorf("expected livekit check to be 'error', got %s", response.Checks["livekit"])
	}
}

// TestReady_RedisUnhealthy tests readiness when Redis is unhealthy.
func TestReady_RedisUnhealthy(t *testing.T) {
	dbChecker := &mockHealthChecker{shouldFail: false}
	redisChecker := &mockHealthChecker{shouldFail: true}

	handlers := NewHealthHandlers(HealthHandlersConfig{
		DBChecker:    dbChecker,
		RedisChecker: redisChecker,
		MetricsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %s", response.Status)
	}

	if response.Checks["redis"] != "error" {
		t.Errorf("expected redis check to be 'error', got %s", response.Checks["redis"])
	}
}

// TestReady_MultipleUnhealthy tests readiness when multiple services are unhealthy.
func TestReady_MultipleUnhealthy(t *testing.T) {
	dbChecker := &mockHealthChecker{shouldFail: true}
	livekitChecker := &mockHealthChecker{shouldFail: true}

	handlers := NewHealthHandlers(HealthHandlersConfig{
		DBChecker:      dbChecker,
		LiveKitChecker: livekitChecker,
		MetricsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %s", response.Status)
	}

	if response.Checks["db"] != "error" {
		t.Errorf("expected db check to be 'error', got %s", response.Checks["db"])
	}
	if response.Checks["livekit"] != "error" {
		t.Errorf("expected livekit check to be 'error', got %s", response.Checks["livekit"])
	}
}

// TestReady_NoCheckers tests readiness when no external checkers are configured.
func TestReady_NoCheckers(t *testing.T) {
	handlers := NewHealthHandlers(HealthHandlersConfig{
		MetricsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "up" {
		t.Errorf("expected status 'up', got %s", response.Status)
	}

	// No checks should be present when no checkers configured
	if len(response.Checks) != 0 {
		t.Errorf("expected no checks when no checkers configured, got %d checks", len(response.Checks))
	}

	if response.UptimeS < 0 {
		t.Errorf("expected uptime_s to be >= 0, got %d", response.UptimeS)
	}
}

// TestReady_MethodNotAllowed tests that non-GET requests are rejected.
func TestReady_MethodNotAllowed(t *testing.T) {
	handlers := NewHealthHandlers(HealthHandlersConfig{})

	req := httptest.NewRequest(http.MethodPost, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	// Verify it returns structured JSON error
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type to contain application/json, got %s", contentType)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeBadRequest {
		t.Errorf("expected error code %s, got %s", ErrCodeBadRequest, errResp.Error.Code)
	}
}

// TestReady_ContentType tests that the response has correct Content-Type.
func TestReady_ContentType(t *testing.T) {
	handlers := NewHealthHandlers(HealthHandlersConfig{})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handlers.Ready(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}
}

// TestHealth_ContentType tests that the health response has correct Content-Type.
func TestHealth_ContentType(t *testing.T) {
	handlers := NewHealthHandlers(HealthHandlersConfig{})

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	handlers.Health(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}
}
