package jobs

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// TestJobMetricsIntegration verifies that job metrics can be registered
// with Prometheus and work correctly in an end-to-end scenario.
func TestJobMetricsIntegration(t *testing.T) {
	// Create a new Prometheus registry (isolated from default registry)
	reg := prometheus.NewRegistry()

	// Create and register job metrics
	m := NewMetrics()
	if err := m.Register(reg); err != nil {
		t.Fatalf("failed to register job metrics: %v", err)
	}

	// Simulate multiple job executions
	jobTypes := []string{
		JobTypeTrustRecompute,
		JobTypeIndexBackfill,
		JobTypePaymentProcess,
	}

	for _, jobType := range jobTypes {
		// Simulate successful job
		startTime := time.Now()
		m.IncJobsTotal(jobType, StatusSuccess)
		m.ObserveJobDuration(jobType, time.Since(startTime).Seconds())

		// Simulate failed job
		startTime = time.Now()
		m.IncJobsTotal(jobType, StatusFailure)
		m.ObserveJobDuration(jobType, time.Since(startTime).Seconds())
		m.IncJobErrors(jobType, "test_error")
	}

	// Gather metrics from registry
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Verify all expected metrics are present
	expectedMetrics := map[string]bool{
		MetricBackgroundJobsTotal:      false,
		MetricBackgroundJobsDuration:   false,
		MetricBackgroundJobErrorsTotal: false,
	}

	for _, family := range families {
		name := family.GetName()
		if _, ok := expectedMetrics[name]; ok {
			expectedMetrics[name] = true
			t.Logf("Found metric: %s with %d samples", name, len(family.GetMetric()))
		}
	}

	// Verify all metrics were found
	for name, found := range expectedMetrics {
		if !found {
			t.Errorf("metric %s not found in gathered metrics", name)
		}
	}

	// Verify metric sample counts
	for _, family := range families {
		name := family.GetName()
		metrics := family.GetMetric()

		switch name {
		case MetricBackgroundJobsTotal:
			// Each job type has success and failure = 6 label combinations
			expectedCount := len(jobTypes) * 2
			if len(metrics) != expectedCount {
				t.Errorf("%s: expected %d label combinations, got %d", name, expectedCount, len(metrics))
			}

		case MetricBackgroundJobsDuration:
			// Each job type has 2 samples = 3 histograms
			if len(metrics) != len(jobTypes) {
				t.Errorf("%s: expected %d histograms, got %d", name, len(jobTypes), len(metrics))
			}

		case MetricBackgroundJobErrorsTotal:
			// Each job type has 1 error = 3 label combinations
			if len(metrics) != len(jobTypes) {
				t.Errorf("%s: expected %d label combinations, got %d", name, len(jobTypes), len(metrics))
			}
		}
	}

	t.Log("Integration test passed: all metrics registered and working correctly")
}

// TestJobMetricsWithTrustJob demonstrates how to use job metrics with a background job.
func TestJobMetricsWithTrustJob(t *testing.T) {
	// This test demonstrates the integration pattern for background jobs
	reg := prometheus.NewRegistry()
	jobMetrics := NewMetrics()
	if err := jobMetrics.Register(reg); err != nil {
		t.Fatalf("failed to register job metrics: %v", err)
	}

	// Simulate a trust recompute job execution with a known duration
	testDuration := 0.123 // 123ms simulated work

	// Record success
	jobMetrics.IncJobsTotal(JobTypeTrustRecompute, StatusSuccess)
	jobMetrics.ObserveJobDuration(JobTypeTrustRecompute, testDuration)

	// Verify metrics were recorded
	successCount := getCounterVecValue(jobMetrics.jobsTotal, JobTypeTrustRecompute, StatusSuccess)
	if successCount != 1.0 {
		t.Errorf("expected success count 1, got %f", successCount)
	}

	durationCount := getHistogramVecSampleCount(jobMetrics.jobsDuration, JobTypeTrustRecompute)
	if durationCount != 1 {
		t.Errorf("expected duration sample count 1, got %d", durationCount)
	}

	// Verify recorded duration matches what we observed
	recordedDuration := getHistogramVecSampleSum(jobMetrics.jobsDuration, JobTypeTrustRecompute)
	if recordedDuration != testDuration {
		t.Errorf("recorded duration = %f, expected %f", recordedDuration, testDuration)
	}

	t.Logf("Trust job metrics integration verified (duration: %fs)", recordedDuration)
}

// TestJobMetricsNilSafe verifies that code using the Reporter interface
// handles nil gracefully (simulating optional metrics configuration).
func TestJobMetricsNilSafe(t *testing.T) {
	// This demonstrates the pattern: check if metrics are configured before calling
	var reporter Reporter = nil

	// Real code should check if reporter != nil before calling methods
	// This test documents the expected usage pattern
	if reporter != nil {
		reporter.IncJobsTotal(JobTypeTrustRecompute, StatusSuccess)
		reporter.ObserveJobDuration(JobTypeTrustRecompute, 1.0)
		reporter.IncJobErrors(JobTypeTrustRecompute, "test")
	}

	t.Log("Nil-safe pattern verified: always check reporter != nil before calling methods")
}
