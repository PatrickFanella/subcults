package middleware

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}
	if m.rateLimitRequests == nil {
		t.Error("rateLimitRequests is nil")
	}
	if m.rateLimitBlocked == nil {
		t.Error("rateLimitBlocked is nil")
	}
}

func TestMetrics_Register(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()

	err := m.Register(reg)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Increment counters to create metrics entries
	m.IncRateLimitRequests("/test", "user")
	m.IncRateLimitBlocked("/test", "ip")

	// Verify metrics are registered by checking they can be collected
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Check that we have the expected metrics
	foundRequests := false
	foundBlocked := false
	for _, mf := range metrics {
		if mf.GetName() == MetricRateLimitRequests {
			foundRequests = true
		}
		if mf.GetName() == MetricRateLimitBlocked {
			foundBlocked = true
		}
	}

	if !foundRequests {
		t.Errorf("metric %s not found in registry", MetricRateLimitRequests)
	}
	if !foundBlocked {
		t.Errorf("metric %s not found in registry", MetricRateLimitBlocked)
	}
}

func TestMetrics_IncRateLimitRequests(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Increment counters
	m.IncRateLimitRequests("/search/events", "user")
	m.IncRateLimitRequests("/search/events", "user")
	m.IncRateLimitRequests("/search/scenes", "ip")

	// Gather metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Find the rate_limit_requests_total metric
	var requestsMetric *dto.MetricFamily
	for i := range metrics {
		if metrics[i].GetName() == MetricRateLimitRequests {
			requestsMetric = metrics[i]
			break
		}
	}

	if requestsMetric == nil {
		t.Fatal("rate_limit_requests_total metric not found")
	}

	// Verify the counter values
	if len(requestsMetric.GetMetric()) != 2 {
		t.Errorf("expected 2 metric entries, got %d", len(requestsMetric.GetMetric()))
	}
}

func TestMetrics_IncRateLimitBlocked(t *testing.T) {
	m := NewMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Increment counters
	m.IncRateLimitBlocked("/search/events", "user")
	m.IncRateLimitBlocked("/streams/join", "user")
	m.IncRateLimitBlocked("/streams/join", "user")

	// Gather metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Find the rate_limit_blocked_total metric
	var blockedMetric *dto.MetricFamily
	for i := range metrics {
		if metrics[i].GetName() == MetricRateLimitBlocked {
			blockedMetric = metrics[i]
			break
		}
	}

	if blockedMetric == nil {
		t.Fatal("rate_limit_blocked_total metric not found")
	}

	// Verify the counter values
	if len(blockedMetric.GetMetric()) != 2 {
		t.Errorf("expected 2 metric entries, got %d", len(blockedMetric.GetMetric()))
	}
}

func TestMetrics_Collectors(t *testing.T) {
	m := NewMetrics()
	collectors := m.Collectors()

	// Should have 3 rate limit metrics + 4 HTTP metrics + 4 canary metrics = 11 total
	if len(collectors) != 11 {
		t.Errorf("expected 11 collectors, got %d", len(collectors))
	}
}
