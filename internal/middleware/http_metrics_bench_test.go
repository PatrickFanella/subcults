package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkHTTPMetrics_Overhead measures the performance overhead of the metrics middleware
func BenchmarkHTTPMetrics_Overhead(b *testing.B) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Benchmark without middleware (baseline)
	b.Run("without_middleware", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/test", nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	// Benchmark with middleware
	b.Run("with_middleware", func(b *testing.B) {
		m := NewMetrics()
		reg := prometheus.NewRegistry()
		if err := m.Register(reg); err != nil {
			b.Fatalf("Register() failed: %v", err)
		}

		wrapped := HTTPMetrics(m)(handler)
		req := httptest.NewRequest("GET", "/test", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkHTTPMetrics_HealthCheckExclusion verifies health check exclusion is fast
func BenchmarkHTTPMetrics_HealthCheckExclusion(b *testing.B) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		b.Fatalf("Register() failed: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := HTTPMetrics(m)(handler)
	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}
}

// BenchmarkHTTPMetrics_DifferentPaths tests performance with varying paths
func BenchmarkHTTPMetrics_DifferentPaths(b *testing.B) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		b.Fatalf("Register() failed: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := HTTPMetrics(m)(handler)

	paths := []string{"/events", "/scenes", "/search/events", "/streams"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}
}
