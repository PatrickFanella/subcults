// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics names as constants for consistency.
const (
	MetricMessagesProcessed    = "indexer_messages_processed_total"
	MetricMessagesError        = "indexer_messages_error_total"
	MetricUpserts              = "indexer_upserts_total"
	MetricTrustRecompute       = "indexer_trust_recompute_total"
	MetricIngestLatency        = "indexer_ingest_latency_seconds"
	MetricBackpressurePaused   = "indexer_backpressure_paused_total"
	MetricBackpressureResumed  = "indexer_backpressure_resumed_total"
	MetricBackpressureDuration = "indexer_backpressure_pause_duration_seconds"
	MetricPendingMessages      = "indexer_pending_messages"
	MetricProcessingLag        = "indexer_processing_lag_seconds"
	MetricReconnectionAttempts = "indexer_reconnection_attempts_total"
	MetricReconnectionSuccess  = "indexer_reconnection_success_total"
	MetricDatabaseWritesFailed = "indexer_database_writes_failed_total"
)

// Metrics contains Prometheus metrics for the indexer.
// All operations are thread-safe.
type Metrics struct {
	messagesProcessed    prometheus.Counter
	messagesError        prometheus.Counter
	upserts              prometheus.Counter
	trustRecompute       prometheus.Counter
	ingestLatency        prometheus.Histogram
	backpressurePaused   prometheus.Counter
	backpressureResumed  prometheus.Counter
	backpressureDuration prometheus.Histogram
	pendingMessages      prometheus.Gauge
	processingLag        prometheus.Gauge
	reconnectionAttempts prometheus.Counter
	reconnectionSuccess  prometheus.Counter
	databaseWritesFailed prometheus.Counter
}

// NewMetrics creates and returns a new Metrics instance with all collectors initialized.
// The metrics are not registered; call Register to register them with a registry.
func NewMetrics() *Metrics {
	return &Metrics{
		messagesProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricMessagesProcessed,
			Help: "Total number of messages processed by the indexer",
		}),
		messagesError: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricMessagesError,
			Help: "Total number of messages that resulted in processing errors",
		}),
		upserts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricUpserts,
			Help: "Total number of database upsert operations",
		}),
		trustRecompute: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricTrustRecompute,
			Help: "Total number of trust score recomputation operations",
		}),
		ingestLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricIngestLatency,
			Help:    "Histogram of message ingestion latency in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		backpressurePaused: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricBackpressurePaused,
			Help: "Total number of times backpressure caused message consumption to pause",
		}),
		backpressureResumed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricBackpressureResumed,
			Help: "Total number of times message consumption resumed after backpressure",
		}),
		backpressureDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    MetricBackpressureDuration,
			Help:    "Histogram of backpressure pause duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60},
		}),
		pendingMessages: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: MetricPendingMessages,
			Help: "Current number of pending messages in the processing queue",
		}),
		processingLag: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: MetricProcessingLag,
			Help: "Time difference in seconds between message timestamp and processing time",
		}),
		reconnectionAttempts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricReconnectionAttempts,
			Help: "Total number of reconnection attempts to Jetstream",
		}),
		reconnectionSuccess: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricReconnectionSuccess,
			Help: "Total number of successful reconnections to Jetstream",
		}),
		databaseWritesFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: MetricDatabaseWritesFailed,
			Help: "Total number of failed database write operations",
		}),
	}
}

// Register registers all metrics with the given registry.
// Returns an error if registration fails.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.messagesProcessed,
		m.messagesError,
		m.upserts,
		m.trustRecompute,
		m.ingestLatency,
		m.backpressurePaused,
		m.backpressureResumed,
		m.backpressureDuration,
		m.pendingMessages,
		m.processingLag,
		m.reconnectionAttempts,
		m.reconnectionSuccess,
		m.databaseWritesFailed,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// IncMessagesProcessed increments the messages processed counter.
func (m *Metrics) IncMessagesProcessed() {
	m.messagesProcessed.Inc()
}

// IncMessagesError increments the messages error counter.
func (m *Metrics) IncMessagesError() {
	m.messagesError.Inc()
}

// IncUpserts increments the upserts counter.
func (m *Metrics) IncUpserts() {
	m.upserts.Inc()
}

// IncTrustRecompute increments the trust recompute counter.
func (m *Metrics) IncTrustRecompute() {
	m.trustRecompute.Inc()
}

// ObserveIngestLatency records an ingestion latency sample.
func (m *Metrics) ObserveIngestLatency(seconds float64) {
	m.ingestLatency.Observe(seconds)
}

// IncBackpressurePaused increments the backpressure paused counter.
func (m *Metrics) IncBackpressurePaused() {
	m.backpressurePaused.Inc()
}

// IncBackpressureResumed increments the backpressure resumed counter.
func (m *Metrics) IncBackpressureResumed() {
	m.backpressureResumed.Inc()
}

// ObserveBackpressureDuration records a backpressure pause duration sample.
func (m *Metrics) ObserveBackpressureDuration(seconds float64) {
	m.backpressureDuration.Observe(seconds)
}

// SetPendingMessages sets the current number of pending messages.
func (m *Metrics) SetPendingMessages(count int) {
	m.pendingMessages.Set(float64(count))
}

// SetProcessingLag sets the current processing lag in seconds.
func (m *Metrics) SetProcessingLag(seconds float64) {
	m.processingLag.Set(seconds)
}

// IncReconnectionAttempts increments the reconnection attempts counter.
func (m *Metrics) IncReconnectionAttempts() {
	m.reconnectionAttempts.Inc()
}

// IncReconnectionSuccess increments the reconnection success counter.
func (m *Metrics) IncReconnectionSuccess() {
	m.reconnectionSuccess.Inc()
}

// IncDatabaseWritesFailed increments the database writes failed counter.
func (m *Metrics) IncDatabaseWritesFailed() {
	m.databaseWritesFailed.Inc()
}

// Collectors returns all Prometheus collectors for testing.
func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.messagesProcessed,
		m.messagesError,
		m.upserts,
		m.trustRecompute,
		m.ingestLatency,
		m.backpressurePaused,
		m.backpressureResumed,
		m.backpressureDuration,
		m.pendingMessages,
		m.processingLag,
		m.reconnectionAttempts,
		m.reconnectionSuccess,
		m.databaseWritesFailed,
	}
}
