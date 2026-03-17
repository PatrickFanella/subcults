package indexer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsHandler(t *testing.T) {
	t.Run("returns metrics in text format", func(t *testing.T) {
		// Create metrics and register them
		m := NewMetrics()
		reg := prometheus.NewRegistry()
		if err := m.Register(reg); err != nil {
			t.Fatalf("failed to register metrics: %v", err)
		}

		// Increment some counters
		m.IncMessagesProcessed()
		m.IncMessagesProcessed()
		m.IncMessagesError()
		m.IncUpserts()
		m.IncTrustRecompute()
		m.ObserveIngestLatency(0.1)

		// Create handler and test server
		handler := MetricsHandler(reg)
		req := httptest.NewRequest(http.MethodGet, "/internal/indexer/metrics", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
		}

		body, err := io.ReadAll(rec.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		bodyStr := string(body)

		// Check for expected metrics in output
		expectedMetrics := []string{
			"indexer_messages_processed_total 2",
			"indexer_messages_error_total 1",
			"indexer_upserts_total 1",
			"indexer_trust_recompute_total 1",
			"indexer_ingest_latency_seconds_bucket",
			"indexer_ingest_latency_seconds_sum",
			"indexer_ingest_latency_seconds_count 1",
		}

		for _, expected := range expectedMetrics {
			if !strings.Contains(bodyStr, expected) {
				t.Errorf("expected response to contain %q, got:\n%s", expected, bodyStr)
			}
		}
	})

	t.Run("returns empty registry correctly", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		handler := MetricsHandler(reg)

		req := httptest.NewRequest(http.MethodGet, "/internal/indexer/metrics", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestInternalAuthMiddleware(t *testing.T) {
	// Simple handler that returns OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	t.Run("no token configured allows all requests", func(t *testing.T) {
		middleware := InternalAuthMiddleware("")
		handler := middleware(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("missing token returns forbidden", func(t *testing.T) {
		middleware := InternalAuthMiddleware("secret-token")
		handler := middleware(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("wrong token returns forbidden", func(t *testing.T) {
		middleware := InternalAuthMiddleware("secret-token")
		handler := middleware(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Internal-Token", "wrong-token")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("correct token allows request", func(t *testing.T) {
		middleware := InternalAuthMiddleware("secret-token")
		handler := middleware(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Internal-Token", "secret-token")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("empty header with token configured returns forbidden", func(t *testing.T) {
		middleware := InternalAuthMiddleware("secret-token")
		handler := middleware(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Internal-Token", "")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})
}

func TestMetricsEndpoint_Integration(t *testing.T) {
	// Integration test: register metrics, apply middleware, verify endpoint works
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("failed to register metrics: %v", err)
	}

	token := "integration-test-token"
	handler := InternalAuthMiddleware(token)(MetricsHandler(reg))

	t.Run("authenticated request returns metrics", func(t *testing.T) {
		// Simulate some ingestion activity
		m.IncMessagesProcessed()
		m.IncUpserts()
		m.ObserveIngestLatency(0.05)

		req := httptest.NewRequest(http.MethodGet, "/internal/indexer/metrics", nil)
		req.Header.Set("X-Internal-Token", token)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "indexer_messages_processed_total") {
			t.Error("expected response to contain indexer_messages_processed_total")
		}
	})

	t.Run("unauthenticated request returns forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/indexer/metrics", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})
}
