package jobs

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}

	// Verify all collectors are initialized
	collectors := m.Collectors()
	if len(collectors) != 3 {
		t.Errorf("expected 3 collectors, got %d", len(collectors))
	}
}

func TestMetrics_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		m := NewMetrics()
		reg := prometheus.NewRegistry()

		if err := m.Register(reg); err != nil {
			t.Errorf("Register() returned error: %v", err)
		}

		// Record some metrics to ensure they appear in Gather()
		m.IncJobsTotal(JobTypeTrustRecompute, StatusSuccess)
		m.ObserveJobDuration(JobTypeTrustRecompute, 1.0)
		m.IncJobErrors(JobTypeTrustRecompute, "test_error")

		// Verify metrics are gathered
		families, err := reg.Gather()
		if err != nil {
			t.Errorf("Gather() returned error: %v", err)
		}

		expectedNames := map[string]bool{
			MetricBackgroundJobsTotal:      false,
			MetricBackgroundJobsDuration:   false,
			MetricBackgroundJobErrorsTotal: false,
		}

		for _, family := range families {
			if _, ok := expectedNames[family.GetName()]; ok {
				expectedNames[family.GetName()] = true
			}
		}

		for name, found := range expectedNames {
			if !found {
				t.Errorf("metric %s not found in gathered metrics", name)
			}
		}
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		m1 := NewMetrics()
		m2 := NewMetrics()
		reg := prometheus.NewRegistry()

		if err := m1.Register(reg); err != nil {
			t.Fatalf("first Register() returned error: %v", err)
		}

		if err := m2.Register(reg); err == nil {
			t.Error("second Register() should have returned an error")
		}
	})
}

func getCounterVecValue(vec *prometheus.CounterVec, labels ...string) float64 {
	metric, err := vec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return -1
	}
	// Need to convert to Metric interface to call Write
	metricInterface, ok := metric.(prometheus.Metric)
	if !ok {
		return -1
	}
	var m dto.Metric
	if err := metricInterface.Write(&m); err != nil {
		return -1
	}
	return m.GetCounter().GetValue()
}

func getHistogramVecSampleCount(vec *prometheus.HistogramVec, labels ...string) uint64 {
	metric, err := vec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}
	// Need to convert to Metric interface to call Write
	metricInterface, ok := metric.(prometheus.Metric)
	if !ok {
		return 0
	}
	var m dto.Metric
	if err := metricInterface.Write(&m); err != nil {
		return 0
	}
	return m.GetHistogram().GetSampleCount()
}

func getHistogramVecSampleSum(vec *prometheus.HistogramVec, labels ...string) float64 {
	metric, err := vec.GetMetricWithLabelValues(labels...)
	if err != nil {
		return -1
	}
	// Need to convert to Metric interface to call Write
	metricInterface, ok := metric.(prometheus.Metric)
	if !ok {
		return -1
	}
	var m dto.Metric
	if err := metricInterface.Write(&m); err != nil {
		return -1
	}
	return m.GetHistogram().GetSampleSum()
}

func TestMetrics_IncJobsTotal(t *testing.T) {
	m := NewMetrics()

	// Test different job types and statuses
	testCases := []struct {
		jobType string
		status  string
		count   int
	}{
		{JobTypeTrustRecompute, StatusSuccess, 10},
		{JobTypeTrustRecompute, StatusFailure, 2},
		{JobTypeIndexBackfill, StatusSuccess, 5},
		{JobTypePaymentProcess, StatusSuccess, 20},
		{JobTypeStreamCleanup, StatusFailure, 1},
	}

	for _, tc := range testCases {
		// Initial value should be 0
		initial := getCounterVecValue(m.jobsTotal, tc.jobType, tc.status)
		if initial != 0 {
			t.Errorf("initial value for %s/%s = %f, want 0", tc.jobType, tc.status, initial)
		}

		// Increment multiple times
		for i := 0; i < tc.count; i++ {
			m.IncJobsTotal(tc.jobType, tc.status)
		}

		final := getCounterVecValue(m.jobsTotal, tc.jobType, tc.status)
		if final != float64(tc.count) {
			t.Errorf("final value for %s/%s = %f, want %d", tc.jobType, tc.status, final, tc.count)
		}
	}
}

func TestMetrics_ObserveJobDuration(t *testing.T) {
	m := NewMetrics()

	testCases := []struct {
		jobType   string
		durations []float64
	}{
		{
			jobType:   JobTypeTrustRecompute,
			durations: []float64{0.5, 1.2, 0.8, 2.5, 1.0},
		},
		{
			jobType:   JobTypeIndexBackfill,
			durations: []float64{30.5, 45.2, 60.1},
		},
		{
			jobType:   JobTypePaymentProcess,
			durations: []float64{0.1, 0.15, 0.2, 0.12},
		},
	}

	for _, tc := range testCases {
		// Initial count should be 0
		initial := getHistogramVecSampleCount(m.jobsDuration, tc.jobType)
		if initial != 0 {
			t.Errorf("initial sample count for %s = %d, want 0", tc.jobType, initial)
		}

		// Observe durations
		var expectedSum float64
		for _, d := range tc.durations {
			m.ObserveJobDuration(tc.jobType, d)
			expectedSum += d
		}

		finalCount := getHistogramVecSampleCount(m.jobsDuration, tc.jobType)
		if finalCount != uint64(len(tc.durations)) {
			t.Errorf("final sample count for %s = %d, want %d", tc.jobType, finalCount, len(tc.durations))
		}

		finalSum := getHistogramVecSampleSum(m.jobsDuration, tc.jobType)
		// Use approximate comparison for floating point
		if finalSum < expectedSum*0.99 || finalSum > expectedSum*1.01 {
			t.Errorf("final sample sum for %s = %f, want approximately %f", tc.jobType, finalSum, expectedSum)
		}
	}
}

func TestMetrics_IncJobErrors(t *testing.T) {
	m := NewMetrics()

	testCases := []struct {
		jobType   string
		errorType string
		count     int
	}{
		{JobTypeTrustRecompute, "timeout", 5},
		{JobTypeTrustRecompute, "database_error", 3},
		{JobTypeIndexBackfill, "validation_error", 2},
		{JobTypePaymentProcess, "network_error", 1},
		{JobTypeStreamCleanup, "permission_denied", 4},
	}

	for _, tc := range testCases {
		// Initial value should be 0
		initial := getCounterVecValue(m.jobErrors, tc.jobType, tc.errorType)
		if initial != 0 {
			t.Errorf("initial value for %s/%s = %f, want 0", tc.jobType, tc.errorType, initial)
		}

		// Increment multiple times
		for i := 0; i < tc.count; i++ {
			m.IncJobErrors(tc.jobType, tc.errorType)
		}

		final := getCounterVecValue(m.jobErrors, tc.jobType, tc.errorType)
		if final != float64(tc.count) {
			t.Errorf("final value for %s/%s = %f, want %d", tc.jobType, tc.errorType, final, tc.count)
		}
	}
}

func TestMetrics_JobTypeConstants(t *testing.T) {
	// Verify all job type constants are unique
	jobTypes := []string{
		JobTypeTrustRecompute,
		JobTypeIndexBackfill,
		JobTypeIndexProcessing,
		JobTypePaymentProcess,
		JobTypeStreamCleanup,
		JobTypeCacheInvalidate,
		JobTypeReportGenerate,
	}

	seen := make(map[string]bool)
	for _, jt := range jobTypes {
		if seen[jt] {
			t.Errorf("duplicate job type constant: %s", jt)
		}
		seen[jt] = true

		// Verify constants are non-empty
		if jt == "" {
			t.Error("job type constant is empty")
		}
	}

	if len(seen) != len(jobTypes) {
		t.Errorf("expected %d unique job types, got %d", len(jobTypes), len(seen))
	}
}

func TestMetrics_StatusConstants(t *testing.T) {
	// Verify status constants
	if StatusSuccess == "" {
		t.Error("StatusSuccess is empty")
	}
	if StatusFailure == "" {
		t.Error("StatusFailure is empty")
	}
	if StatusSuccess == StatusFailure {
		t.Error("StatusSuccess and StatusFailure should be different")
	}
}

func TestMetrics_Concurrency(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup
	iterations := 100
	goroutines := 10

	// Run concurrent operations on all metrics
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.IncJobsTotal(JobTypeTrustRecompute, StatusSuccess)
				m.IncJobsTotal(JobTypeTrustRecompute, StatusFailure)
				m.ObserveJobDuration(JobTypeTrustRecompute, 1.5)
				m.IncJobErrors(JobTypeTrustRecompute, "test_error")
			}
		}(i)
	}

	wg.Wait()

	expected := float64(goroutines * iterations)

	// Verify counter values
	successCount := getCounterVecValue(m.jobsTotal, JobTypeTrustRecompute, StatusSuccess)
	if successCount != expected {
		t.Errorf("jobsTotal success count = %f, want %f", successCount, expected)
	}

	failureCount := getCounterVecValue(m.jobsTotal, JobTypeTrustRecompute, StatusFailure)
	if failureCount != expected {
		t.Errorf("jobsTotal failure count = %f, want %f", failureCount, expected)
	}

	errorCount := getCounterVecValue(m.jobErrors, JobTypeTrustRecompute, "test_error")
	if errorCount != expected {
		t.Errorf("jobErrors count = %f, want %f", errorCount, expected)
	}

	// Verify histogram count
	expectedHistCount := uint64(goroutines * iterations)
	histCount := getHistogramVecSampleCount(m.jobsDuration, JobTypeTrustRecompute)
	if histCount != expectedHistCount {
		t.Errorf("jobsDuration sample count = %d, want %d", histCount, expectedHistCount)
	}
}

func TestMetrics_MultipleJobTypes(t *testing.T) {
	m := NewMetrics()

	// Record metrics for multiple job types
	jobTypes := []string{
		JobTypeTrustRecompute,
		JobTypeIndexBackfill,
		JobTypePaymentProcess,
		JobTypeStreamCleanup,
	}

	for _, jt := range jobTypes {
		m.IncJobsTotal(jt, StatusSuccess)
		m.ObserveJobDuration(jt, 2.5)
		m.IncJobErrors(jt, "test_error")
	}

	// Verify each job type has its own metrics
	for _, jt := range jobTypes {
		successCount := getCounterVecValue(m.jobsTotal, jt, StatusSuccess)
		if successCount != 1.0 {
			t.Errorf("jobsTotal for %s = %f, want 1.0", jt, successCount)
		}

		histCount := getHistogramVecSampleCount(m.jobsDuration, jt)
		if histCount != 1 {
			t.Errorf("jobsDuration count for %s = %d, want 1", jt, histCount)
		}

		errorCount := getCounterVecValue(m.jobErrors, jt, "test_error")
		if errorCount != 1.0 {
			t.Errorf("jobErrors for %s = %f, want 1.0", jt, errorCount)
		}
	}
}

func TestMetrics_DurationBuckets(t *testing.T) {
	m := NewMetrics()

	// Test that histogram buckets are appropriate for various job durations
	durations := []float64{
		0.05,  // Very fast
		0.5,   // Fast
		5.0,   // Medium
		30.0,  // Slow
		120.0, // Very slow
	}

	for _, d := range durations {
		m.ObserveJobDuration(JobTypeTrustRecompute, d)
	}

	// Verify all samples were recorded
	count := getHistogramVecSampleCount(m.jobsDuration, JobTypeTrustRecompute)
	if count != uint64(len(durations)) {
		t.Errorf("sample count = %d, want %d", count, len(durations))
	}

	// Verify sum is correct
	var expectedSum float64
	for _, d := range durations {
		expectedSum += d
	}
	actualSum := getHistogramVecSampleSum(m.jobsDuration, JobTypeTrustRecompute)
	if actualSum < expectedSum*0.99 || actualSum > expectedSum*1.01 {
		t.Errorf("sample sum = %f, want approximately %f", actualSum, expectedSum)
	}
}
