// Package middleware provides canary deployment routing and monitoring tests.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCanaryRouter_AssignCohort(t *testing.T) {
	config := CanaryConfig{
		Enabled:        true,
		TrafficPercent: 10.0, // 10% canary traffic
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	tests := []struct {
		name       string
		userID     string
		wantCohort string // Expected cohort based on hash
	}{
		{
			name:       "user assigned to canary",
			userID:     "user-canary-123",
			wantCohort: "", // We'll verify consistency, not specific assignment
		},
		{
			name:       "user assigned to stable",
			userID:     "user-stable-456",
			wantCohort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-User-ID", tt.userID)

			// Get cohort assignment
			cohort := router.assignCohort(req)

			// Verify cohort is either "canary" or "stable"
			if cohort != "canary" && cohort != "stable" {
				t.Errorf("assignCohort() returned invalid cohort: %s", cohort)
			}

			// Verify consistency: same user ID should always get same cohort
			for i := 0; i < 10; i++ {
				req2 := httptest.NewRequest("GET", "/test", nil)
				req2.Header.Set("X-User-ID", tt.userID)
				cohort2 := router.assignCohort(req2)
				if cohort != cohort2 {
					t.Errorf("assignCohort() is not deterministic: first=%s, subsequent=%s", cohort, cohort2)
				}
			}
		})
	}
}

func TestCanaryRouter_TrafficDistribution(t *testing.T) {
	config := CanaryConfig{
		Enabled:        true,
		TrafficPercent: 20.0, // 20% canary traffic
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	// Simulate 1000 requests with different user IDs
	canaryCount := 0
	stableCount := 0
	totalRequests := 1000

	for i := 0; i < totalRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("user-%d", i)) // Different user ID each time

		cohort := router.assignCohort(req)
		if cohort == "canary" {
			canaryCount++
		} else {
			stableCount++
		}
	}

	// Verify distribution is approximately 20% canary (allow 5% margin of error)
	canaryPercent := float64(canaryCount) / float64(totalRequests) * 100
	expectedPercent := config.TrafficPercent

	marginOfError := 5.0
	if canaryPercent < expectedPercent-marginOfError || canaryPercent > expectedPercent+marginOfError {
		t.Errorf("Traffic distribution outside expected range: got %.2f%%, want %.2f%% ± %.2f%%",
			canaryPercent, expectedPercent, marginOfError)
	}

	t.Logf("Traffic distribution: canary=%.2f%%, stable=%.2f%%",
		canaryPercent, float64(stableCount)/float64(totalRequests)*100)
}

func TestCanaryRouter_Middleware(t *testing.T) {
	config := CanaryConfig{
		Enabled:        true,
		TrafficPercent: 50.0, // 50% for easier testing
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers are set
		cohort := r.Header.Get("X-Deployment-Cohort")
		version := r.Header.Get("X-Deployment-Version")

		if cohort == "" {
			t.Error("X-Deployment-Cohort header not set in request")
		}
		if version == "" {
			t.Error("X-Deployment-Version header not set in request")
		}

		w.WriteHeader(http.StatusOK)
	})

	middleware := router.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "test-user-123")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify response headers are set
	if w.Header().Get("X-Deployment-Cohort") == "" {
		t.Error("X-Deployment-Cohort header not set in response")
	}
	if w.Header().Get("X-Deployment-Version") == "" {
		t.Error("X-Deployment-Version header not set in response")
	}
}

func TestCanaryRouter_MetricsRecording(t *testing.T) {
	config := CanaryConfig{
		Enabled:          true,
		TrafficPercent:   50.0,
		ErrorThreshold:   5.0,
		LatencyThreshold: 1.0,
		AutoRollback:     false, // Disable auto-rollback for this test
		Version:          "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	// Create test handler that returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate some latency
		w.WriteHeader(http.StatusOK)
	})

	middleware := router.Middleware(handler)

	// Send requests to both cohorts
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("user-%d", i))
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	// Get metrics
	snapshot := router.GetMetrics()

	// Verify metrics were recorded
	totalRequests := snapshot.CanaryRequests + snapshot.StableRequests
	if totalRequests != 10 {
		t.Errorf("Expected 10 total requests, got %d", totalRequests)
	}

	if snapshot.CanaryAvgLatency <= 0 && snapshot.CanaryRequests > 0 {
		t.Error("Canary latency should be > 0 when there are canary requests")
	}

	if snapshot.StableAvgLatency <= 0 && snapshot.StableRequests > 0 {
		t.Error("Stable latency should be > 0 when there are stable requests")
	}

	t.Logf("Metrics: Canary=%d requests, Stable=%d requests", snapshot.CanaryRequests, snapshot.StableRequests)
}

func TestCanaryRouter_ErrorTracking(t *testing.T) {
	config := CanaryConfig{
		Enabled:          true,
		TrafficPercent:   50.0,
		ErrorThreshold:   10.0,
		AutoRollback:     false,
		Version:          "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	// Create test handler that returns 500 errors for canary cohort
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cohort := r.Header.Get("X-Deployment-Cohort")
		if cohort == "canary" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	middleware := router.Middleware(handler)

	// Send 100 requests
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("user-%d", i))
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	// Get metrics
	snapshot := router.GetMetrics()

	// Canary should have higher error rate
	if snapshot.CanaryErrorRate <= snapshot.StableErrorRate {
		t.Errorf("Expected canary error rate (%.2f%%) > stable error rate (%.2f%%)",
			snapshot.CanaryErrorRate, snapshot.StableErrorRate)
	}

	t.Logf("Error rates: Canary=%.2f%%, Stable=%.2f%%",
		snapshot.CanaryErrorRate, snapshot.StableErrorRate)
}

func TestCanaryRouter_Rollback(t *testing.T) {
	config := CanaryConfig{
		Enabled:          true,
		TrafficPercent:   50.0,
		ErrorThreshold:   5.0, // Low threshold to trigger rollback
		LatencyThreshold: 0.1, // Low threshold
		AutoRollback:     true,
		Version:          "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	// Create handler that causes high error rate for canary
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cohort := r.Header.Get("X-Deployment-Cohort")
		if cohort == "canary" {
			w.WriteHeader(http.StatusInternalServerError) // Always error
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	middleware := router.Middleware(handler)

	// Send enough requests to trigger rollback (need 100+ canary requests for threshold check)
	// With 50% traffic split, send 250 requests to ensure >100 canary requests
	for i := 0; i < 250; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("user-%d", i))
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}

	// Check if rollback was triggered
	snapshot := router.GetMetrics()
	if snapshot.CanaryActive {
		t.Error("Expected canary to be rolled back due to high error rate")
	}

	// Send another request and verify it goes to stable
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "test-after-rollback")
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	cohort := w.Header().Get("X-Deployment-Cohort")
	if cohort != "stable" {
		t.Errorf("After rollback, expected stable cohort, got %s", cohort)
	}
}

func TestCanaryRouter_DisabledCanary(t *testing.T) {
	config := CanaryConfig{
		Enabled:        false, // Disabled
		TrafficPercent: 50.0,
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := router.Middleware(handler)

	// All requests should go to stable
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", fmt.Sprintf("user-%d", i))
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		cohort := w.Header().Get("X-Deployment-Cohort")
		if cohort != "stable" {
			t.Errorf("With canary disabled, expected stable cohort, got %s", cohort)
		}
	}

	// Verify no canary requests
	snapshot := router.GetMetrics()
	if snapshot.CanaryRequests > 0 {
		t.Errorf("With canary disabled, expected 0 canary requests, got %d", snapshot.CanaryRequests)
	}
}

func TestCanaryRouter_ResetMetrics(t *testing.T) {
	config := CanaryConfig{
		Enabled:        true,
		TrafficPercent: 50.0,
		Version:        "v1.2.0-canary",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	router := NewCanaryRouter(config, logger)

	// Record some metrics
	router.recordRequest("canary", 0.1, false)
	router.recordRequest("stable", 0.2, false)

	// Verify metrics exist
	snapshot := router.GetMetrics()
	if snapshot.CanaryRequests == 0 && snapshot.StableRequests == 0 {
		t.Error("Expected metrics to be recorded")
	}

	// Reset metrics
	router.ResetMetrics()

	// Verify metrics are reset
	snapshot = router.GetMetrics()
	if snapshot.CanaryRequests != 0 || snapshot.StableRequests != 0 {
		t.Errorf("Expected metrics to be reset, got canary=%d, stable=%d",
			snapshot.CanaryRequests, snapshot.StableRequests)
	}
}
