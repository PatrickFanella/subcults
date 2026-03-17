// Package trust provides trust score computation for scenes based on
// membership and alliance relationships.
package trust

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics names as constants for consistency.
const (
	MetricTrustRecomputeTotal             = "trust_recompute_total"
	MetricTrustRecomputeErrors            = "trust_recompute_errors_total"
	MetricTrustRecomputeDuration          = "trust_recompute_duration_seconds"
	MetricTrustLastRecomputeTimestamp     = "trust_last_recompute_timestamp"
	MetricTrustLastRecomputeSceneCount    = "trust_last_recompute_scene_count"
	MetricTrustRecomputeBatchDuration     = "trust_recompute_batch_duration_ms"
	MetricTrustRecomputeEntitiesPerSecond = "trust_recompute_entities_per_sec"
)

// Metrics contains Prometheus metrics for trust score recomputation.
// All operations are thread-safe.
type Metrics struct {
	recomputeTotal          prometheus.Counter
	recomputeErrors         prometheus.Counter
	recomputeDuration       prometheus.Histogram
	lastRecomputeTimestamp  prometheus.Gauge
	lastRecomputeSceneCount prometheus.Gauge
	batchDuration           prometheus.Histogram
	entitiesPerSecond       prometheus.Histogram
}

// NewMetrics creates and returns a new Metrics instance with all collectors initialized.
// The metrics are not registered; call Register to register them with a registry.
func NewMetrics() *Metrics {
	return &Metrics{
		recomputeTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricTrustRecomputeTotal,
			Help: "Total number of trust score recomputation operations",
		}),
		recomputeErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricTrustRecomputeErrors,
			Help: "Total number of trust score recomputation errors",
		}),
		recomputeDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricTrustRecomputeDuration,
			Help:    "Histogram of trust score recomputation duration in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0},
		}),
		lastRecomputeTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: MetricTrustLastRecomputeTimestamp,
			Help: "Unix timestamp of the last trust score recomputation",
		}),
		lastRecomputeSceneCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: MetricTrustLastRecomputeSceneCount,
			Help: "Number of scenes processed in the last trust score recomputation",
		}),
		batchDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricTrustRecomputeBatchDuration,
			Help:    "Histogram of trust score recomputation batch duration in milliseconds",
			Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500, 5000},
		}),
		entitiesPerSecond: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricTrustRecomputeEntitiesPerSecond,
			Help:    "Histogram of trust score recomputation throughput in entities per second",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
		}),
	}
}

// Register registers all metrics with the given registry.
// Returns an error if registration fails.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.recomputeTotal,
		m.recomputeErrors,
		m.recomputeDuration,
		m.lastRecomputeTimestamp,
		m.lastRecomputeSceneCount,
		m.batchDuration,
		m.entitiesPerSecond,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// IncRecomputeTotal increments the recompute total counter.
func (m *Metrics) IncRecomputeTotal() {
	m.recomputeTotal.Inc()
}

// IncRecomputeErrors increments the recompute errors counter.
func (m *Metrics) IncRecomputeErrors() {
	m.recomputeErrors.Inc()
}

// ObserveRecomputeDuration records a recompute duration sample.
func (m *Metrics) ObserveRecomputeDuration(seconds float64) {
	m.recomputeDuration.Observe(seconds)
}

// SetLastRecomputeTimestamp sets the last recompute timestamp gauge.
func (m *Metrics) SetLastRecomputeTimestamp(timestamp float64) {
	m.lastRecomputeTimestamp.Set(timestamp)
}

// SetLastRecomputeSceneCount sets the last recompute scene count gauge.
func (m *Metrics) SetLastRecomputeSceneCount(count float64) {
	m.lastRecomputeSceneCount.Set(count)
}

// ObserveBatchDuration records a batch processing duration sample in milliseconds.
func (m *Metrics) ObserveBatchDuration(ms float64) {
	m.batchDuration.Observe(ms)
}

// ObserveEntitiesPerSecond records a throughput sample in entities per second.
func (m *Metrics) ObserveEntitiesPerSecond(rate float64) {
	m.entitiesPerSecond.Observe(rate)
}

// Collectors returns all Prometheus collectors for testing.
func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.recomputeTotal,
		m.recomputeErrors,
		m.recomputeDuration,
		m.lastRecomputeTimestamp,
		m.lastRecomputeSceneCount,
		m.batchDuration,
		m.entitiesPerSecond,
	}
}
