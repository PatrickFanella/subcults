package backfill

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetrics_Registration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	if m == nil {
		t.Fatal("expected non-nil metrics")
	}

	// Initialize counter vecs so they appear in Gather
	m.RecordProcessed("test", "test")
	m.RecordSkipped("test", "test")
	m.RecordError("test", "test")
	m.CheckpointCreated("test", "test")

	// Verify all metrics are registered by gathering
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("unexpected error gathering metrics: %v", err)
	}

	expected := map[string]bool{
		MetricRecordsProcessed: false,
		MetricRecordsSkipped:   false,
		MetricErrors:           false,
		MetricDuration:         false,
		MetricCheckpoints:      false,
		MetricActiveBackfills:  false,
		MetricBatchDuration:    false,
		MetricConsistencyScore: false,
	}

	for _, f := range families {
		if _, ok := expected[f.GetName()]; ok {
			expected[f.GetName()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("metric %s not registered", name)
		}
	}
}

func TestMetrics_Counters(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.RecordProcessed("jetstream", "app.subcult.scene")
	m.RecordProcessed("jetstream", "app.subcult.scene")
	m.RecordSkipped("jetstream", "non_matching")
	m.RecordError("jetstream", "parse_error")

	families, _ := reg.Gather()
	for _, f := range families {
		switch f.GetName() {
		case MetricRecordsProcessed:
			val := f.GetMetric()[0].GetCounter().GetValue()
			if val != 2 {
				t.Errorf("expected processed=2, got %f", val)
			}
		case MetricRecordsSkipped:
			val := f.GetMetric()[0].GetCounter().GetValue()
			if val != 1 {
				t.Errorf("expected skipped=1, got %f", val)
			}
		case MetricErrors:
			val := f.GetMetric()[0].GetCounter().GetValue()
			if val != 1 {
				t.Errorf("expected errors=1, got %f", val)
			}
		}
	}
}

func TestMetrics_Gauges(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.SetActive()
	m.SetConsistencyScore(0.95)

	families, _ := reg.Gather()
	for _, f := range families {
		switch f.GetName() {
		case MetricActiveBackfills:
			val := f.GetMetric()[0].GetGauge().GetValue()
			if val != 1 {
				t.Errorf("expected active=1, got %f", val)
			}
		case MetricConsistencyScore:
			val := f.GetMetric()[0].GetGauge().GetValue()
			if val != 0.95 {
				t.Errorf("expected score=0.95, got %f", val)
			}
		}
	}

	m.SetInactive()
	families, _ = reg.Gather()
	for _, f := range families {
		if f.GetName() == MetricActiveBackfills {
			val := f.GetMetric()[0].GetGauge().GetValue()
			if val != 0 {
				t.Errorf("expected active=0 after inactive, got %f", val)
			}
		}
	}
}

func TestMetrics_Histograms(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.ObserveDuration(120.5)
	m.ObserveBatchDuration(0.5)

	families, _ := reg.Gather()
	for _, f := range families {
		switch f.GetName() {
		case MetricDuration:
			count := f.GetMetric()[0].GetHistogram().GetSampleCount()
			if count != 1 {
				t.Errorf("expected 1 duration observation, got %d", count)
			}
		case MetricBatchDuration:
			count := f.GetMetric()[0].GetHistogram().GetSampleCount()
			if count != 1 {
				t.Errorf("expected 1 batch duration observation, got %d", count)
			}
		}
	}
}
