package backfill

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metric names for backfill operations.
const (
	MetricRecordsProcessed = "backfill_records_processed_total"
	MetricRecordsSkipped   = "backfill_records_skipped_total"
	MetricErrors           = "backfill_errors_total"
	MetricDuration         = "backfill_duration_seconds"
	MetricCheckpoints      = "backfill_checkpoints_total"
	MetricActiveBackfills  = "backfill_active"
	MetricBatchDuration    = "backfill_batch_duration_seconds"
	MetricConsistencyScore = "backfill_consistency_score"
)

// Metrics holds Prometheus metrics for backfill operations.
type Metrics struct {
	RecordsProcessed *prometheus.CounterVec
	RecordsSkipped   *prometheus.CounterVec
	Errors           *prometheus.CounterVec
	Duration         prometheus.Histogram
	Checkpoints      *prometheus.CounterVec
	ActiveBackfills  prometheus.Gauge
	BatchDuration    prometheus.Histogram
	ConsistencyScore prometheus.Gauge
}

// NewMetrics creates and registers backfill metrics.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		RecordsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricRecordsProcessed,
				Help: "Total number of records processed during backfill",
			},
			[]string{"source", "collection"},
		),
		RecordsSkipped: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricRecordsSkipped,
				Help: "Total number of records skipped during backfill",
			},
			[]string{"source", "reason"},
		),
		Errors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricErrors,
				Help: "Total number of errors during backfill",
			},
			[]string{"source", "type"},
		),
		Duration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    MetricDuration,
				Help:    "Total duration of backfill operations in seconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 16), // 1s to ~9h
			},
		),
		Checkpoints: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricCheckpoints,
				Help: "Total number of checkpoints created",
			},
			[]string{"source", "status"},
		),
		ActiveBackfills: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: MetricActiveBackfills,
				Help: "Number of currently running backfill operations",
			},
		),
		BatchDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    MetricBatchDuration,
				Help:    "Duration of individual batch processing in seconds",
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 12), // 10ms to ~40s
			},
		),
		ConsistencyScore: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: MetricConsistencyScore,
				Help: "Latest consistency verification score (0.0 to 1.0)",
			},
		),
	}

	reg.MustRegister(
		m.RecordsProcessed,
		m.RecordsSkipped,
		m.Errors,
		m.Duration,
		m.Checkpoints,
		m.ActiveBackfills,
		m.BatchDuration,
		m.ConsistencyScore,
	)
	return m
}

// RecordProcessed increments the processed counter.
func (m *Metrics) RecordProcessed(source, collection string) {
	m.RecordsProcessed.WithLabelValues(source, collection).Inc()
}

// RecordSkipped increments the skipped counter.
func (m *Metrics) RecordSkipped(source, reason string) {
	m.RecordsSkipped.WithLabelValues(source, reason).Inc()
}

// RecordError increments the error counter.
func (m *Metrics) RecordError(source, errType string) {
	m.Errors.WithLabelValues(source, errType).Inc()
}

// CheckpointCreated records checkpoint creation.
func (m *Metrics) CheckpointCreated(source, status string) {
	m.Checkpoints.WithLabelValues(source, status).Inc()
}

// ObserveDuration records the total backfill duration.
func (m *Metrics) ObserveDuration(seconds float64) {
	m.Duration.Observe(seconds)
}

// ObserveBatchDuration records batch processing duration.
func (m *Metrics) ObserveBatchDuration(seconds float64) {
	m.BatchDuration.Observe(seconds)
}

// SetActive marks a backfill as active.
func (m *Metrics) SetActive() {
	m.ActiveBackfills.Inc()
}

// SetInactive marks a backfill as finished.
func (m *Metrics) SetInactive() {
	m.ActiveBackfills.Dec()
}

// SetConsistencyScore updates the consistency score gauge.
func (m *Metrics) SetConsistencyScore(score float64) {
	m.ConsistencyScore.Set(score)
}
