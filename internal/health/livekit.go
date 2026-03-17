package health

import (
"context"
"fmt"
"net/http"
"time"
)

// LiveKitChecker implements health checking for LiveKit.
type LiveKitChecker struct {
url    string
client *http.Client
}

// NewLiveKitChecker creates a new LiveKit health checker.
// The url should be the base URL of the LiveKit server (e.g., "https://livekit.example.com").
func NewLiveKitChecker(url string) *LiveKitChecker {
return &LiveKitChecker{
url: url,
client: &http.Client{
Timeout: 3 * time.Second,
Transport: &http.Transport{
MaxIdleConns:        16,
MaxIdleConnsPerHost: 4,
IdleConnTimeout:     30 * time.Second,
},
},
}
}

// HealthCheck performs a health check on LiveKit by making an HTTP request.
// LiveKit doesn't have a standard health endpoint, so we check if the server is reachable.
func (l *LiveKitChecker) HealthCheck(ctx context.Context) error {
if l.url == "" {
return fmt.Errorf("livekit url not configured")
}

req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url, nil)
if err != nil {
return fmt.Errorf("failed to create request: %w", err)
}

resp, err := l.client.Do(req)
if err != nil {
return fmt.Errorf("failed to reach livekit server: %w", err)
}
defer resp.Body.Close()

// Consider the server healthy only for successful (2xx) responses.
// Non-2xx status codes likely indicate the service is unavailable or misconfigured.
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
return fmt.Errorf("livekit unhealthy: unexpected status code %d", resp.StatusCode)
}

return nil
}
