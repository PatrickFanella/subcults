// Package api provides HTTP handlers for canary deployment management tests.
package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/onnwee/subcults/internal/middleware"
)

func TestCanaryHandler_GetMetrics(t *testing.T) {
	config := middleware.CanaryConfig{
		Enabled:        true,
		TrafficPercent: 10.0,
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := middleware.NewCanaryRouter(config, logger)
	handler := NewCanaryHandler(router, logger)

	// Record some test metrics
	router.GetMetrics() // Initialize

	req := httptest.NewRequest("GET", "/canary/metrics", nil)
	w := httptest.NewRecorder()

	handler.GetMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var snapshot middleware.MetricsSnapshot
	if err := json.NewDecoder(w.Body).Decode(&snapshot); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if snapshot.CanaryVersion != "v1.2.0-canary" {
		t.Errorf("Expected canary version v1.2.0-canary, got %s", snapshot.CanaryVersion)
	}
}

func TestCanaryHandler_GetMetrics_MethodNotAllowed(t *testing.T) {
	config := middleware.CanaryConfig{
		Enabled: true,
		Version: "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := middleware.NewCanaryRouter(config, logger)
	handler := NewCanaryHandler(router, logger)

	req := httptest.NewRequest("POST", "/canary/metrics", nil)
	w := httptest.NewRecorder()

	handler.GetMetrics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestCanaryHandler_Rollback(t *testing.T) {
	config := middleware.CanaryConfig{
		Enabled:        true,
		TrafficPercent: 10.0,
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := middleware.NewCanaryRouter(config, logger)
	handler := NewCanaryHandler(router, logger)

	// Verify canary is active initially
	snapshot := router.GetMetrics()
	if !snapshot.CanaryActive {
		t.Error("Expected canary to be active initially")
	}

	// Trigger rollback
	reqBody := map[string]string{"reason": "test_rollback"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/canary/rollback", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Rollback(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success to be true")
	}

	if response["reason"] != "test_rollback" {
		t.Errorf("Expected reason test_rollback, got %v", response["reason"])
	}

	// Verify canary is no longer active
	snapshot = router.GetMetrics()
	if snapshot.CanaryActive {
		t.Error("Expected canary to be inactive after rollback")
	}
}

func TestCanaryHandler_Rollback_NoReason(t *testing.T) {
	config := middleware.CanaryConfig{
		Enabled:        true,
		TrafficPercent: 10.0,
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := middleware.NewCanaryRouter(config, logger)
	handler := NewCanaryHandler(router, logger)

	// Trigger rollback without reason
	req := httptest.NewRequest("POST", "/canary/rollback", nil)
	w := httptest.NewRecorder()

	handler.Rollback(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should default to "manual_rollback"
	if response["reason"] != "manual_rollback" {
		t.Errorf("Expected default reason manual_rollback, got %v", response["reason"])
	}
}

func TestCanaryHandler_Rollback_MethodNotAllowed(t *testing.T) {
	config := middleware.CanaryConfig{
		Enabled: true,
		Version: "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := middleware.NewCanaryRouter(config, logger)
	handler := NewCanaryHandler(router, logger)

	req := httptest.NewRequest("GET", "/canary/rollback", nil)
	w := httptest.NewRecorder()

	handler.Rollback(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestCanaryHandler_ResetMetrics(t *testing.T) {
	config := middleware.CanaryConfig{
		Enabled:        true,
		TrafficPercent: 10.0,
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := middleware.NewCanaryRouter(config, logger)
	handler := NewCanaryHandler(router, logger)

	req := httptest.NewRequest("POST", "/canary/metrics/reset", nil)
	w := httptest.NewRecorder()

	handler.ResetMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success to be true")
	}
}

func TestCanaryHandler_ResetMetrics_MethodNotAllowed(t *testing.T) {
	config := middleware.CanaryConfig{
		Enabled: true,
		Version: "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := middleware.NewCanaryRouter(config, logger)
	handler := NewCanaryHandler(router, logger)

	req := httptest.NewRequest("GET", "/canary/metrics/reset", nil)
	w := httptest.NewRecorder()

	handler.ResetMetrics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
