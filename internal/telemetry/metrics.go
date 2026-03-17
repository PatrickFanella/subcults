package telemetry

import "github.com/prometheus/client_golang/prometheus"

// Metrics contains Prometheus metrics for telemetry and error logging endpoints.
type Metrics struct {
	EventsReceived      *prometheus.CounterVec
	BatchesTotal        *prometheus.CounterVec
	ClientErrorsTotal   *prometheus.CounterVec
	ClientErrorsDeduped prometheus.Counter
}

// NewMetrics creates a new set of telemetry metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		EventsReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "subcults_telemetry_events_received_total",
				Help: "Total number of telemetry events received from the frontend",
			},
			[]string{"event_name"},
		),
		BatchesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "subcults_telemetry_batches_total",
				Help: "Total number of telemetry batches processed",
			},
			[]string{"status"},
		),
		ClientErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "subcults_client_errors_total",
				Help: "Total number of client-side errors received",
			},
			[]string{"error_type"},
		),
		ClientErrorsDeduped: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "subcults_client_errors_deduplicated_total",
				Help: "Total number of client errors skipped due to deduplication",
			},
		),
	}
}

// Register registers all metrics with the given Prometheus registry.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	for _, c := range []prometheus.Collector{
		m.EventsReceived,
		m.BatchesTotal,
		m.ClientErrorsTotal,
		m.ClientErrorsDeduped,
	} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}
