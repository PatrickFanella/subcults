package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProfiling_Disabled(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Wrap with profiling middleware (disabled)
	wrapped := Profiling(ProfilingConfig{
		Enabled:     false,
		Environment: "development",
	})(handler)

	// Request profiling endpoint
	req := httptest.NewRequest("GET", "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should pass through to handler (not serve profiling page)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "ok" {
		t.Errorf("expected 'ok', got %q", body)
	}
}

func TestProfiling_EnabledInDevelopment(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should not reach here"))
	})

	// Wrap with profiling middleware (enabled in development)
	wrapped := Profiling(ProfilingConfig{
		Enabled:     true,
		Environment: "development",
	})(handler)

	// Request profiling index
	req := httptest.NewRequest("GET", "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should serve profiling page (HTML content)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Profile") && !strings.Contains(body, "pprof") {
		t.Errorf("expected profiling page content, got %q", body)
	}
}

func TestProfiling_BlockedInProduction(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Attempt to enable profiling in production (should be blocked)
	wrapped := Profiling(ProfilingConfig{
		Enabled:     true,
		Environment: "production",
	})(handler)

	// Request profiling endpoint
	req := httptest.NewRequest("GET", "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should pass through to handler (profiling blocked in production)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "ok" {
		t.Errorf("expected 'ok', got %q", body)
	}
}

func TestProfiling_CPUProfile(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should not reach here"))
	})

	// Wrap with profiling middleware (enabled)
	wrapped := Profiling(ProfilingConfig{
		Enabled:     true,
		Environment: "development",
	})(handler)

	// Request CPU profile with short duration
	req := httptest.NewRequest("GET", "/debug/pprof/profile?seconds=1", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should serve CPU profile
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestProfiling_HeapProfile(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should not reach here"))
	})

	// Wrap with profiling middleware (enabled)
	wrapped := Profiling(ProfilingConfig{
		Enabled:     true,
		Environment: "development",
	})(handler)

	// Request heap profile
	req := httptest.NewRequest("GET", "/debug/pprof/heap", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should serve heap profile
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestProfiling_GoroutineProfile(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should not reach here"))
	})

	// Wrap with profiling middleware (enabled)
	wrapped := Profiling(ProfilingConfig{
		Enabled:     true,
		Environment: "development",
	})(handler)

	// Request goroutine profile
	req := httptest.NewRequest("GET", "/debug/pprof/goroutine", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should serve goroutine profile
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestProfiling_NonProfilingRoute(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("normal route"))
	})

	// Wrap with profiling middleware (enabled)
	wrapped := Profiling(ProfilingConfig{
		Enabled:     true,
		Environment: "development",
	})(handler)

	// Request non-profiling route
	req := httptest.NewRequest("GET", "/api/scenes", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should pass through to normal handler
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "normal route" {
		t.Errorf("expected 'normal route', got %q", body)
	}
}

func TestProfilingStatus_Disabled(t *testing.T) {
	handler := ProfilingStatus(ProfilingConfig{
		Enabled:     false,
		Environment: "production",
	})

	req := httptest.NewRequest("GET", "/profiling/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"profiling_enabled": false`) {
		t.Errorf("expected profiling_enabled: false, got %q", body)
	}
	if !strings.Contains(body, `"status": "disabled"`) {
		t.Errorf("expected status: disabled, got %q", body)
	}
}

func TestProfilingStatus_Enabled(t *testing.T) {
	handler := ProfilingStatus(ProfilingConfig{
		Enabled:     true,
		Environment: "development",
	})

	req := httptest.NewRequest("GET", "/profiling/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"profiling_enabled": true`) {
		t.Errorf("expected profiling_enabled: true, got %q", body)
	}
	if !strings.Contains(body, `"status": "enabled"`) {
		t.Errorf("expected status: enabled, got %q", body)
	}
	if !strings.Contains(body, "/debug/pprof/") {
		t.Errorf("expected endpoint list, got %q", body)
	}
}

// BenchmarkProfiling_Overhead measures the performance overhead of the profiling middleware
func BenchmarkProfiling_Overhead(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Benchmark without middleware
	b.Run("without_middleware", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/test", nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	// Benchmark with middleware (disabled)
	b.Run("with_middleware_disabled", func(b *testing.B) {
		wrapped := Profiling(ProfilingConfig{
			Enabled:     false,
			Environment: "development",
		})(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}
	})

	// Benchmark with middleware (enabled, non-profiling route)
	b.Run("with_middleware_enabled_normal_route", func(b *testing.B) {
		wrapped := Profiling(ProfilingConfig{
			Enabled:     true,
			Environment: "development",
		})(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}
	})

	// Benchmark with middleware (enabled, profiling route)
	b.Run("with_middleware_enabled_profiling_route", func(b *testing.B) {
		wrapped := Profiling(ProfilingConfig{
			Enabled:     true,
			Environment: "development",
		})(handler)
		req := httptest.NewRequest("GET", "/debug/pprof/goroutine", nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
			// Consume response body to simulate real usage
			_, _ = io.ReadAll(rec.Body)
		}
	})
}
