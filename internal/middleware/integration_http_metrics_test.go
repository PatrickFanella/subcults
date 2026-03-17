package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestHTTPMetrics_Integration verifies the full middleware chain works correctly
func TestHTTPMetrics_Integration(t *testing.T) {
	// Create metrics
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Create a simple handler that returns 200
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Apply the middleware
	handler := HTTPMetrics(m)(testHandler)

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Gather metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Verify metrics were recorded
	foundMetrics := 0
	for _, mf := range metrics {
		switch mf.GetName() {
		case MetricHTTPRequestDuration,
			MetricHTTPRequestsTotal,
			MetricHTTPRequestSizeBytes,
			MetricHTTPResponseSizeBytes:
			foundMetrics++
		}
	}

	if foundMetrics != 4 {
		t.Errorf("expected 4 HTTP metrics, found %d", foundMetrics)
	}
}

// TestHTTPMetrics_MiddlewareOrdering verifies the middleware can be composed with other middleware
func TestHTTPMetrics_MiddlewareOrdering(t *testing.T) {
	// Create metrics
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Create a test handler
	called := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Create a dummy middleware that adds a header
	headerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "value")
			next.ServeHTTP(w, r)
		})
	}

	// Compose middleware: headerMiddleware -> HTTPMetrics -> handler
	handler := headerMiddleware(HTTPMetrics(m)(testHandler))

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify the handler was called
	if !called {
		t.Error("handler was not called")
	}

	// Verify the header was set
	if rec.Header().Get("X-Test") != "value" {
		t.Error("header middleware did not run")
	}

	// Verify metrics were recorded
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	found := false
	for _, mf := range metrics {
		if mf.GetName() == MetricHTTPRequestsTotal {
			found = true
			break
		}
	}

	if !found {
		t.Error("HTTP metrics were not recorded")
	}
}

func TestHTTPMetrics_PathNormalization(t *testing.T) {
	// Create metrics
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Wrap with metrics middleware
	wrapped := HTTPMetrics(m)(handler)

	// Make requests with different IDs but same pattern
	paths := []string{
		"/events/123",
		"/events/456",
		"/events/abc-def-ghi",
		"/events/550e8400-e29b-41d4-a716-446655440000",
	}

	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	}

	// Gather metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Find the counter metric
	var totalMetric *dto.MetricFamily
	for i := range metrics {
		if metrics[i].GetName() == MetricHTTPRequestsTotal {
			totalMetric = metrics[i]
			break
		}
	}

	if totalMetric == nil {
		t.Fatal("total metric not found")
	}

	// Should have exactly 1 label set (all normalized to /events/{id})
	if len(totalMetric.GetMetric()) != 1 {
		t.Errorf("expected 1 label set (normalized path), got %d", len(totalMetric.GetMetric()))
	}

	// Verify the label is the normalized path
	if len(totalMetric.GetMetric()) > 0 {
		labels := totalMetric.GetMetric()[0].GetLabel()
		pathLabel := ""
		for _, label := range labels {
			if label.GetName() == "path" {
				pathLabel = label.GetValue()
				break
			}
		}

		expectedPath := "/events/{id}"
		if pathLabel != expectedPath {
			t.Errorf("path label = %s, want %s", pathLabel, expectedPath)
		}

		// Verify the counter value is 4 (all requests counted under same label)
		counter := totalMetric.GetMetric()[0].GetCounter()
		if counter.GetValue() != 4 {
			t.Errorf("counter value = %f, want 4", counter.GetValue())
		}
	}
}
