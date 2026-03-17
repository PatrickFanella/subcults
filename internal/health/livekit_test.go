package health

import (
"context"
"net/http"
"net/http/httptest"
"testing"
)

// TestLiveKitChecker_Creation tests that the LiveKit checker is created correctly.
func TestLiveKitChecker_Creation(t *testing.T) {
url := "https://livekit.example.com"

checker := NewLiveKitChecker(url)
if checker == nil {
t.Fatal("expected checker to be non-nil")
}

if checker.url != url {
t.Errorf("expected checker url to be %s, got %s", url, checker.url)
}

if checker.client == nil {
t.Error("expected HTTP client to be initialized")
}

if checker.client.Timeout == 0 {
t.Error("expected HTTP client timeout to be set")
}
}

// TestLiveKitChecker_EmptyURL tests that an empty URL returns an error.
func TestLiveKitChecker_EmptyURL(t *testing.T) {
checker := NewLiveKitChecker("")

ctx := context.Background()
err := checker.HealthCheck(ctx)

if err == nil {
t.Error("expected error with empty URL")
}

expectedMsg := "livekit url not configured"
if err.Error() != expectedMsg {
t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
}
}

// TestLiveKitChecker_SuccessfulResponse tests health check with 2xx response.
func TestLiveKitChecker_SuccessfulResponse(t *testing.T) {
// Create a test server that returns 200 OK
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
}))
defer server.Close()

checker := NewLiveKitChecker(server.URL)
ctx := context.Background()

err := checker.HealthCheck(ctx)
if err != nil {
t.Errorf("expected no error for 200 OK response, got %v", err)
}
}

// TestLiveKitChecker_ErrorResponse tests health check with non-2xx response.
func TestLiveKitChecker_ErrorResponse(t *testing.T) {
testCases := []struct {
name       string
statusCode int
}{
{"404 Not Found", http.StatusNotFound},
{"500 Internal Server Error", http.StatusInternalServerError},
{"503 Service Unavailable", http.StatusServiceUnavailable},
}

for _, tc := range testCases {
t.Run(tc.name, func(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(tc.statusCode)
}))
defer server.Close()

checker := NewLiveKitChecker(server.URL)
ctx := context.Background()

err := checker.HealthCheck(ctx)
if err == nil {
t.Errorf("expected error for %d response, got nil", tc.statusCode)
}
})
}
}

// TestLiveKitChecker_ContextCancellation tests that context cancellation is handled.
func TestLiveKitChecker_ContextCancellation(t *testing.T) {
// Create a server that never responds
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
<-r.Context().Done()
}))
defer server.Close()

checker := NewLiveKitChecker(server.URL)

ctx, cancel := context.WithCancel(context.Background())
cancel() // Cancel immediately

err := checker.HealthCheck(ctx)
if err == nil {
t.Error("expected error for cancelled context")
}
}
