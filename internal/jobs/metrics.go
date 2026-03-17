// Package jobs provides metrics for background job operations.
package jobs

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics names as constants for consistency.
const (
	MetricBackgroundJobsTotal      = "background_jobs_total"
	MetricBackgroundJobsDuration   = "background_jobs_duration_seconds"
	MetricBackgroundJobErrorsTotal = "background_job_errors_total"
)

// Job type constants for labeling.
const (
	JobTypeTrustRecompute  = "trust_recompute"
	JobTypeIndexBackfill   = "index_backfill"
	JobTypeIndexProcessing = "index_processing"
	JobTypePaymentProcess  = "payment_processing"
	JobTypeStreamCleanup   = "stream_cleanup"
	JobTypeCacheInvalidate = "cache_invalidation"
	JobTypeReportGenerate  = "report_generation"
)

// Status constants for job completion.
const (
	StatusSuccess = "success"
	StatusFailure = "failure"
)

// Reporter provides the interface for centralized background job metrics tracking.
// This interface allows jobs to report to the centralized job metrics system.
type Reporter interface {
	IncJobsTotal(jobType, status string)
	ObserveJobDuration(jobType string, seconds float64)
	IncJobErrors(jobType, errorType string)
}

// Metrics contains Prometheus metrics for background job operations.
// All operations are thread-safe.
type Metrics struct {
	jobsTotal    *prometheus.CounterVec
	jobsDuration *prometheus.HistogramVec
	jobErrors    *prometheus.CounterVec
}

// NewMetrics creates and returns a new Metrics instance with all collectors initialized.
// The metrics are not registered; call Register to register them with a registry.
func NewMetrics() *Metrics {
	return &Metrics{
		jobsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricBackgroundJobsTotal,
				Help: "Total number of background job executions by type and status",
			},
			[]string{"job_type", "status"},
		),
		jobsDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    MetricBackgroundJobsDuration,
				Help:    "Histogram of background job duration in seconds by job type",
				Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0},
			},
			[]string{"job_type"},
		),
		jobErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: MetricBackgroundJobErrorsTotal,
				Help: "Total number of background job errors by type and error type",
			},
			[]string{"job_type", "error_type"},
		),
	}
}

// Register registers all metrics with the given registry.
// Returns an error if registration fails.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.jobsTotal,
		m.jobsDuration,
		m.jobErrors,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// IncJobsTotal increments the jobs total counter.
// jobType: The type of job (e.g., JobTypeTrustRecompute)
// status: The completion status (StatusSuccess or StatusFailure)
func (m *Metrics) IncJobsTotal(jobType, status string) {
	m.jobsTotal.WithLabelValues(jobType, status).Inc()
}

// ObserveJobDuration records a job duration sample.
// jobType: The type of job (e.g., JobTypeTrustRecompute)
// seconds: Duration of the job in seconds
func (m *Metrics) ObserveJobDuration(jobType string, seconds float64) {
	m.jobsDuration.WithLabelValues(jobType).Observe(seconds)
}

// IncJobErrors increments the job errors counter.
// jobType: The type of job (e.g., JobTypeTrustRecompute)
// errorType: The type of error (e.g., "timeout", "database_error", "validation_error")
func (m *Metrics) IncJobErrors(jobType, errorType string) {
	m.jobErrors.WithLabelValues(jobType, errorType).Inc()
}

// Collectors returns all Prometheus collectors for testing.
func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.jobsTotal,
		m.jobsDuration,
		m.jobErrors,
	}
}
