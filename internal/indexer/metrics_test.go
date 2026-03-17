package indexer

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

	// Verify all collectors are initialized
	collectors := m.Collectors()
	if len(collectors) != 13 {
		t.Errorf("expected 13 collectors, got %d", len(collectors))
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
			MetricMessagesProcessed:    false,
			MetricMessagesError:        false,
			MetricUpserts:              false,
			MetricTrustRecompute:       false,
			MetricIngestLatency:        false,
			MetricBackpressurePaused:   false,
			MetricBackpressureResumed:  false,
			MetricBackpressureDuration: false,
			MetricPendingMessages:      false,
			MetricProcessingLag:        false,
			MetricReconnectionAttempts: false,
			MetricReconnectionSuccess:  false,
			MetricDatabaseWritesFailed: false,
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

func TestMetrics_IncMessagesProcessed(t *testing.T) {
	m := NewMetrics()

	// Initial value should be 0
	initial := getCounterValue(m.messagesProcessed)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	// Increment 100 times
	for i := 0; i < 100; i++ {
		m.IncMessagesProcessed()
	}

	final := getCounterValue(m.messagesProcessed)
	if final != 100 {
		t.Errorf("final value = %f, want 100", final)
	}
}

func TestMetrics_IncMessagesError(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.messagesError)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 50; i++ {
		m.IncMessagesError()
	}

	final := getCounterValue(m.messagesError)
	if final != 50 {
		t.Errorf("final value = %f, want 50", final)
	}
}

func TestMetrics_IncUpserts(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.upserts)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 75; i++ {
		m.IncUpserts()
	}

	final := getCounterValue(m.upserts)
	if final != 75 {
		t.Errorf("final value = %f, want 75", final)
	}
}

func TestMetrics_IncTrustRecompute(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.trustRecompute)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 25; i++ {
		m.IncTrustRecompute()
	}

	final := getCounterValue(m.trustRecompute)
	if final != 25 {
		t.Errorf("final value = %f, want 25", final)
	}
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

func TestMetrics_ObserveIngestLatency(t *testing.T) {
	m := NewMetrics()

	// Initial count should be 0
	initial := getHistogramSampleCount(m.ingestLatency)
	if initial != 0 {
		t.Errorf("initial sample count = %d, want 0", initial)
	}

	// Observe some latencies
	latencies := []float64{0.001, 0.002, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0}
	var expectedSum float64
	for _, l := range latencies {
		m.ObserveIngestLatency(l)
		expectedSum += l
	}

	finalCount := getHistogramSampleCount(m.ingestLatency)
	if finalCount != uint64(len(latencies)) {
		t.Errorf("final sample count = %d, want %d", finalCount, len(latencies))
	}

	finalSum := getHistogramSampleSum(m.ingestLatency)
	// Use approximate comparison for floating point
	if finalSum < expectedSum*0.99 || finalSum > expectedSum*1.01 {
		t.Errorf("final sample sum = %f, want approximately %f", finalSum, expectedSum)
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
				m.IncMessagesProcessed()
				m.IncMessagesError()
				m.IncUpserts()
				m.IncTrustRecompute()
				m.ObserveIngestLatency(0.001)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	expected := float64(10 * iterations)

	if v := getCounterValue(m.messagesProcessed); v != expected {
		t.Errorf("messagesProcessed = %f, want %f", v, expected)
	}
	if v := getCounterValue(m.messagesError); v != expected {
		t.Errorf("messagesError = %f, want %f", v, expected)
	}
	if v := getCounterValue(m.upserts); v != expected {
		t.Errorf("upserts = %f, want %f", v, expected)
	}
	if v := getCounterValue(m.trustRecompute); v != expected {
		t.Errorf("trustRecompute = %f, want %f", v, expected)
	}

	expectedHistCount := uint64(10 * iterations)
	if c := getHistogramSampleCount(m.ingestLatency); c != expectedHistCount {
		t.Errorf("ingestLatency sample count = %d, want %d", c, expectedHistCount)
	}
}

func TestMetrics_IncBackpressurePaused(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.backpressurePaused)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 10; i++ {
		m.IncBackpressurePaused()
	}

	final := getCounterValue(m.backpressurePaused)
	if final != 10 {
		t.Errorf("final value = %f, want 10", final)
	}
}

func TestMetrics_IncBackpressureResumed(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.backpressureResumed)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 8; i++ {
		m.IncBackpressureResumed()
	}

	final := getCounterValue(m.backpressureResumed)
	if final != 8 {
		t.Errorf("final value = %f, want 8", final)
	}
}

func TestMetrics_ObserveBackpressureDuration(t *testing.T) {
	m := NewMetrics()

	// Initial count should be 0
	initial := getHistogramSampleCount(m.backpressureDuration)
	if initial != 0 {
		t.Errorf("initial sample count = %d, want 0", initial)
	}

	// Observe some durations
	durations := []float64{0.5, 1.0, 5.0, 10.0, 30.0}
	var expectedSum float64
	for _, d := range durations {
		m.ObserveBackpressureDuration(d)
		expectedSum += d
	}

	finalCount := getHistogramSampleCount(m.backpressureDuration)
	if finalCount != uint64(len(durations)) {
		t.Errorf("final sample count = %d, want %d", finalCount, len(durations))
	}

	finalSum := getHistogramSampleSum(m.backpressureDuration)
	if finalSum < expectedSum*0.99 || finalSum > expectedSum*1.01 {
		t.Errorf("final sample sum = %f, want approximately %f", finalSum, expectedSum)
	}
}

func getGaugeValue(g prometheus.Gauge) float64 {
	var m dto.Metric
	if err := g.(prometheus.Metric).Write(&m); err != nil {
		return -1
	}
	return m.GetGauge().GetValue()
}

func TestMetrics_SetPendingMessages(t *testing.T) {
	m := NewMetrics()

	// Initial value should be 0
	initial := getGaugeValue(m.pendingMessages)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	// Set to 500
	m.SetPendingMessages(500)
	val := getGaugeValue(m.pendingMessages)
	if val != 500 {
		t.Errorf("value after set = %f, want 500", val)
	}

	// Set to 1500
	m.SetPendingMessages(1500)
	val = getGaugeValue(m.pendingMessages)
	if val != 1500 {
		t.Errorf("value after second set = %f, want 1500", val)
	}

	// Set back to 50
	m.SetPendingMessages(50)
	val = getGaugeValue(m.pendingMessages)
	if val != 50 {
		t.Errorf("value after third set = %f, want 50", val)
	}
}

func TestMetrics_SetProcessingLag(t *testing.T) {
	m := NewMetrics()

	// Initial value should be 0
	initial := getGaugeValue(m.processingLag)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	// Set to 0.5 seconds
	m.SetProcessingLag(0.5)
	val := getGaugeValue(m.processingLag)
	if val != 0.5 {
		t.Errorf("value after set = %f, want 0.5", val)
	}

	// Set to 2.3 seconds
	m.SetProcessingLag(2.3)
	val = getGaugeValue(m.processingLag)
	if val != 2.3 {
		t.Errorf("value after second set = %f, want 2.3", val)
	}

	// Set to 0 (caught up)
	m.SetProcessingLag(0)
	val = getGaugeValue(m.processingLag)
	if val != 0 {
		t.Errorf("value after third set = %f, want 0", val)
	}
}

func TestMetrics_IncReconnectionAttempts(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.reconnectionAttempts)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 15; i++ {
		m.IncReconnectionAttempts()
	}

	final := getCounterValue(m.reconnectionAttempts)
	if final != 15 {
		t.Errorf("final value = %f, want 15", final)
	}
}

func TestMetrics_IncReconnectionSuccess(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.reconnectionSuccess)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 12; i++ {
		m.IncReconnectionSuccess()
	}

	final := getCounterValue(m.reconnectionSuccess)
	if final != 12 {
		t.Errorf("final value = %f, want 12", final)
	}
}

func TestMetrics_IncDatabaseWritesFailed(t *testing.T) {
	m := NewMetrics()

	initial := getCounterValue(m.databaseWritesFailed)
	if initial != 0 {
		t.Errorf("initial value = %f, want 0", initial)
	}

	for i := 0; i < 20; i++ {
		m.IncDatabaseWritesFailed()
	}

	final := getCounterValue(m.databaseWritesFailed)
	if final != 20 {
		t.Errorf("final value = %f, want 20", final)
	}
}
