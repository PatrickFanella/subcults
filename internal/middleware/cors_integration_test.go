package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORS_IntegrationWithMiddlewareStack tests CORS in combination with other middleware.
func TestCORS_IntegrationWithMiddlewareStack(t *testing.T) {
	// Set up CORS middleware
	corsConfig := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	// Create a handler that goes through both RequestID and CORS middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware stack: RequestID -> CORS -> handler
	// (in reverse order of execution)
	wrappedHandler := RequestID(CORS(corsConfig)(handler))

	t.Run("preflight request with request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rr := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rr, req)

		// Should handle preflight
		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, rr.Code)
		}

		// Should have CORS headers
		if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
			t.Errorf("expected CORS origin header, got: %s", origin)
		}

		// Should have request ID (added by RequestID middleware)
		if reqID := rr.Header().Get("X-Request-ID"); reqID == "" {
			t.Error("expected X-Request-ID header to be set")
		}
	})

	t.Run("actual request with CORS and request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rr := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rr, req)

		// Should succeed
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		// Should have CORS headers
		if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
			t.Errorf("expected CORS origin header, got: %s", origin)
		}

		// Should have request ID
		if reqID := rr.Header().Get("X-Request-ID"); reqID == "" {
			t.Error("expected X-Request-ID header to be set")
		}

		// Should have response body
		if body := rr.Body.String(); body != "OK" {
			t.Errorf("expected body 'OK', got: %s", body)
		}
	})

	t.Run("unauthorized origin blocked before reaching handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://malicious.com")
		rr := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rr, req)

		// Should be rejected by CORS
		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rr.Code)
		}

		// Should still have request ID (outer middleware)
		if reqID := rr.Header().Get("X-Request-ID"); reqID == "" {
			t.Error("expected X-Request-ID header even for rejected requests")
		}

		// Should not have CORS headers for rejected origin
		if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
			t.Errorf("expected no CORS headers for rejected origin, got: %s", origin)
		}
	})
}
