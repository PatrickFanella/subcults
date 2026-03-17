package trust

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

	// Verify all collectors are initialized (includes 2 new metrics from issue #172)
	collectors := m.Collectors()
	if len(collectors) != 7 {
		t.Errorf("expected 7 collectors, got %d", len(collectors))
	}
}

func TestMetrics_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		m := NewMetrics()
		reg := prometheus.NewRegistry()

		if err := m.Register(reg); err != nil {
			t.Errorf("Register() returned error: %v", err)
		}

		// Verify metrics are gathered
		families, err := reg.Gather()
		if err != nil {
			t.Errorf("Gather() returned error: %v", err)
		}

		expectedNames := map[string]bool{
			MetricTrustRecomputeTotal:             false,
			MetricTrustRecomputeErrors:            false,
			MetricTrustRecomputeDuration:          false,
			MetricTrustLastRecomputeTimestamp:     false,
			MetricTrustLastRecomputeSceneCount:    false,
			MetricTrustRecomputeBatchDuration:     false, // Added in issue #172
			MetricTrustRecomputeEntitiesPerSecond: false, // Added in issue #172
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

func getCounterValue(c prometheus.Counter) float64 {
	var m dto.Metric
	if err := c.(prometheus.Metric).Write(&m); err != nil {
		return -1
	}
	return m.GetCounter().GetValue()
}

func getGaugeValue(g prometheus.Gauge) float64 {
	var m dto.Metric
	if err := g.(prometheus.Metric).Write(&m); err != nil {
		return -1
	}
	return m.GetGauge().GetValue()
}

func getHistogramSampleCount(h prometheus.Histogram) uint64 {
	var m dto.Metric
	if err := h.(prometheus.Metric).Write(&m); err != nil {
		return 0
	}
	return m.GetHistogram().GetSampleCount()
}

func getHistogramSampleSum(h prometheus.Histogram) float64 {
	var m dto.Metric
	if err := h.(prometheus.Metric).Write(&m); err != nil {
		return -1
	}
	return m.GetHistogram().GetSampleSum()
}

func TestMetrics_IncRecomputeTotal(t *testing.T) {
	m := NewMetrics()

	// Initial value should be 0
	initial := getCounterValue(m.recomputeTotal)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	// Increment 50 times
	for i := 0; i < 50; i++ {
		m.IncRecomputeTotal()
	}

	final := getCounterValue(m.recomputeTotal)
	if final != 50 {
		t.Errorf("final value = %f, want 50", final)
	}
}

func TestMetrics_IncRecomputeErrors(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.recomputeErrors)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 10; i++ {
		m.IncRecomputeErrors()
	}

	final := getCounterValue(m.recomputeErrors)
	if final != 10 {
		t.Errorf("final value = %f, want 10", final)
	}
}

func TestMetrics_ObserveRecomputeDuration(t *testing.T) {
	m := NewMetrics()

	// Initial count should be 0
	initial := getHistogramSampleCount(m.recomputeDuration)
	if initial != 0 {
		t.Errorf("initial sample count = %d, want 0", initial)
	}

	// Observe some durations (simulating real-world recompute times)
	durations := []float64{0.5, 1.2, 0.8, 2.5, 1.0, 3.2, 0.9, 1.5}
	var expectedSum float64
	for _, d := range durations {
		m.ObserveRecomputeDuration(d)
		expectedSum += d
	}

	finalCount := getHistogramSampleCount(m.recomputeDuration)
	if finalCount != uint64(len(durations)) {
		t.Errorf("final sample count = %d, want %d", finalCount, len(durations))
	}

	finalSum := getHistogramSampleSum(m.recomputeDuration)
	// Use approximate comparison for floating point
	if finalSum < expectedSum*0.99 || finalSum > expectedSum*1.01 {
		t.Errorf("final sample sum = %f, want approximately %f", finalSum, expectedSum)
	}
}

func TestMetrics_SetLastRecomputeTimestamp(t *testing.T) {
	m := NewMetrics()

	// Initial value should be 0
	initial := getGaugeValue(m.lastRecomputeTimestamp)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	// Set to a timestamp
	timestamp := 1234567890.0
	m.SetLastRecomputeTimestamp(timestamp)

	final := getGaugeValue(m.lastRecomputeTimestamp)
	if final != timestamp {
		t.Errorf("final value = %f, want %f", final, timestamp)
	}
}

func TestMetrics_SetLastRecomputeSceneCount(t *testing.T) {
	m := NewMetrics()

	// Initial value should be 0
	initial := getGaugeValue(m.lastRecomputeSceneCount)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	// Set to a scene count
	count := 50.0
	m.SetLastRecomputeSceneCount(count)

	final := getGaugeValue(m.lastRecomputeSceneCount)
	if final != count {
		t.Errorf("final value = %f, want %f", final, count)
	}
}

func TestMetrics_Concurrency(t *testing.T) {
	m := NewMetrics()
	done := make(chan bool)
	iterations := 100

	// Run concurrent operations on all metrics
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				m.IncRecomputeTotal()
				m.IncRecomputeErrors()
				m.ObserveRecomputeDuration(1.5)
				m.SetLastRecomputeTimestamp(float64(j))
				m.SetLastRecomputeSceneCount(float64(j))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	expected := float64(10 * iterations)

	if v := getCounterValue(m.recomputeTotal); v != expected {
		t.Errorf("recomputeTotal = %f, want %f", v, expected)
	}
	if v := getCounterValue(m.recomputeErrors); v != expected {
		t.Errorf("recomputeErrors = %f, want %f", v, expected)
	}

	expectedHistCount := uint64(10 * iterations)
	if c := getHistogramSampleCount(m.recomputeDuration); c != expectedHistCount {
		t.Errorf("recomputeDuration sample count = %d, want %d", c, expectedHistCount)
	}

	// Gauge values are non-deterministic in concurrent scenario, just verify they're set
	if v := getGaugeValue(m.lastRecomputeTimestamp); v < 0 {
		t.Errorf("lastRecomputeTimestamp = %f, should be >= 0", v)
	}
	if v := getGaugeValue(m.lastRecomputeSceneCount); v < 0 {
		t.Errorf("lastRecomputeSceneCount = %f, should be >= 0", v)
	}
}
