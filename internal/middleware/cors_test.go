package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORS_DisabledWhenNoOrigins tests that CORS is disabled when no origins are configured.
func TestCORS_DisabledWhenNoOrigins(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{},
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should pass through without CORS headers when disabled
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Errorf("expected no CORS headers when disabled, got Access-Control-Allow-Origin: %s", origin)
	}
}

// TestCORS_AllowedOrigin tests that requests from allowed origins receive CORS headers.
func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name   string
		origin string
	}{
		{"localhost", "http://localhost:3000"},
		{"example.com", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Check status
			if rr.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
			}

			// Check CORS headers for actual requests
			// Only origin and credentials should be set (not methods/headers)
			if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != tt.origin {
				t.Errorf("expected Access-Control-Allow-Origin: %s, got: %s", tt.origin, origin)
			}

			if creds := rr.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
				t.Errorf("expected Access-Control-Allow-Credentials: true, got: %s", creds)
			}

			// Methods and headers should NOT be present on actual requests (only preflight)
			if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods != "" {
				t.Errorf("expected no Access-Control-Allow-Methods on actual request, got: %s", methods)
			}

			if headers := rr.Header().Get("Access-Control-Allow-Headers"); headers != "" {
				t.Errorf("expected no Access-Control-Allow-Headers on actual request, got: %s", headers)
			}
		})
	}
}

// TestCORS_UnauthorizedOrigin tests that requests from unauthorized origins are rejected.
func TestCORS_UnauthorizedOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://malicious.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should reject with 403 Forbidden
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status %d for unauthorized origin, got %d", http.StatusForbidden, rr.Code)
	}

	// Should not have CORS headers
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header for unauthorized origin, got: %s", origin)
	}
}

// TestCORS_NoOriginHeader tests that requests without Origin header are allowed (same-origin).
func TestCORS_NoOriginHeader(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Origin header set
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should pass through for same-origin requests
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d for same-origin request, got %d", http.StatusOK, rr.Code)
	}

	if body := rr.Body.String(); body != "OK" {
		t.Errorf("expected body 'OK', got: %s", body)
	}

	// No CORS headers needed for same-origin
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Errorf("expected no CORS headers for same-origin request, got Access-Control-Allow-Origin: %s", origin)
	}
}

// TestCORS_PreflightRequest tests that preflight OPTIONS requests are handled correctly.
func TestCORS_PreflightRequest(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should not be called for OPTIONS requests
		t.Error("handler should not be called for preflight OPTIONS request")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should return 204 No Content
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status %d for preflight request, got %d", http.StatusNoContent, rr.Code)
	}

	// Check preflight headers
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin: http://localhost:3000, got: %s", origin)
	}

	if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, PUT, DELETE" {
		t.Errorf("expected Access-Control-Allow-Methods: GET, POST, PUT, DELETE, got: %s", methods)
	}

	if headers := rr.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type, Authorization, X-Request-ID" {
		t.Errorf("expected Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID, got: %s", headers)
	}

	if creds := rr.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials: true, got: %s", creds)
	}

	if maxAge := rr.Header().Get("Access-Control-Max-Age"); maxAge != "3600" {
		t.Errorf("expected Access-Control-Max-Age: 3600, got: %s", maxAge)
	}
}

// TestCORS_PreflightUnauthorizedOrigin tests that preflight requests from unauthorized origins are rejected.
func TestCORS_PreflightUnauthorizedOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for rejected preflight request")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://malicious.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should reject with 403 Forbidden
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status %d for unauthorized preflight, got %d", http.StatusForbidden, rr.Code)
	}
}

// TestCORS_CredentialsDisabled tests CORS when credentials are disabled.
func TestCORS_CredentialsDisabled(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should not have credentials header when disabled
	if creds := rr.Header().Get("Access-Control-Allow-Credentials"); creds != "" {
		t.Errorf("expected no Access-Control-Allow-Credentials header when disabled, got: %s", creds)
	}
}

// TestCORS_OriginWithWhitespace tests that origins with whitespace are handled correctly.
func TestCORS_OriginWithWhitespace(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"  http://localhost:3000  ", "https://example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should allow origin after trimming whitespace
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin: http://localhost:3000, got: %s", origin)
	}
}

// TestCORS_EmptyOriginInList tests that empty strings in allowed origins are ignored.
func TestCORS_EmptyOriginInList(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"", "http://localhost:3000", ""},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin: http://localhost:3000, got: %s", origin)
	}
}
