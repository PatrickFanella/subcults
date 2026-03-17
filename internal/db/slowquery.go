// Package db provides database utilities and connection handling for Subcults.
package db

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Default slow query thresholds.
const (
	DefaultSlowQueryThreshold     = 100 * time.Millisecond
	DefaultVerySlowQueryThreshold = 5 * time.Second
)

// SlowQueryMetrics contains Prometheus metrics for database query performance.
type SlowQueryMetrics struct {
	queryDuration   *prometheus.HistogramVec
	slowQueries     prometheus.Counter
	verySlowQueries prometheus.Counter
}

// NewSlowQueryMetrics creates metrics collectors for slow query tracking.
func NewSlowQueryMetrics() *SlowQueryMetrics {
	return &SlowQueryMetrics{
		queryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"operation"},
		),
		slowQueries: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_slow_queries_total",
			Help: "Total number of queries exceeding the slow query threshold (100ms)",
		}),
		verySlowQueries: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_very_slow_queries_total",
			Help: "Total number of queries exceeding the very slow query threshold (5s)",
		}),
	}
}

// Register registers all metrics with the given registry.
func (m *SlowQueryMetrics) Register(reg prometheus.Registerer) error {
	for _, c := range []prometheus.Collector{m.queryDuration, m.slowQueries, m.verySlowQueries} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// Collectors returns all Prometheus collectors for testing.
func (m *SlowQueryMetrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{m.queryDuration, m.slowQueries, m.verySlowQueries}
}

// InstrumentedDB wraps a *sql.DB with slow query logging and metrics.
type InstrumentedDB struct {
	db                    *sql.DB
	logger                *slog.Logger
	metrics               *SlowQueryMetrics
	slowQueryThreshold    time.Duration
	verySlowQueryThreshold time.Duration
}

// InstrumentedDBConfig holds configuration for InstrumentedDB.
type InstrumentedDBConfig struct {
	DB                     *sql.DB
	Logger                 *slog.Logger
	Metrics                *SlowQueryMetrics
	SlowQueryThreshold     time.Duration
	VerySlowQueryThreshold time.Duration
}

// NewInstrumentedDB creates a new InstrumentedDB with slow query tracking.
func NewInstrumentedDB(cfg InstrumentedDBConfig) *InstrumentedDB {
	slow := cfg.SlowQueryThreshold
	if slow == 0 {
		slow = DefaultSlowQueryThreshold
	}
	verySlow := cfg.VerySlowQueryThreshold
	if verySlow == 0 {
		verySlow = DefaultVerySlowQueryThreshold
	}
	return &InstrumentedDB{
		db:                     cfg.DB,
		logger:                 cfg.Logger,
		metrics:                cfg.Metrics,
		slowQueryThreshold:     slow,
		verySlowQueryThreshold: verySlow,
	}
}

// DB returns the underlying *sql.DB for operations not covered by this wrapper.
func (idb *InstrumentedDB) DB() *sql.DB {
	return idb.db
}

// QueryContext executes a query with slow query tracking.
func (idb *InstrumentedDB) QueryContext(ctx context.Context, operation, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := idb.db.QueryContext(ctx, query, args...)
	idb.recordDuration(operation, query, start, err)
	return rows, err
}

// QueryRowContext executes a single-row query with slow query tracking.
func (idb *InstrumentedDB) QueryRowContext(ctx context.Context, operation, query string, args ...any) *sql.Row {
	start := time.Now()
	row := idb.db.QueryRowContext(ctx, query, args...)
	idb.recordDuration(operation, query, start, nil)
	return row
}

// ExecContext executes a statement with slow query tracking.
func (idb *InstrumentedDB) ExecContext(ctx context.Context, operation, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := idb.db.ExecContext(ctx, query, args...)
	idb.recordDuration(operation, query, start, err)
	return result, err
}

// recordDuration records query duration and logs slow queries.
func (idb *InstrumentedDB) recordDuration(operation, query string, start time.Time, err error) {
	duration := time.Since(start)

	if idb.metrics != nil {
		idb.metrics.queryDuration.WithLabelValues(operation).Observe(duration.Seconds())
	}

	if duration >= idb.verySlowQueryThreshold {
		if idb.metrics != nil {
			idb.metrics.verySlowQueries.Inc()
			idb.metrics.slowQueries.Inc()
		}
		idb.logger.Error("very slow database query",
			slog.String("operation", operation),
			slog.Duration("duration", duration),
			slog.String("threshold", idb.verySlowQueryThreshold.String()),
			slog.Bool("error", err != nil),
		)
	} else if duration >= idb.slowQueryThreshold {
		if idb.metrics != nil {
			idb.metrics.slowQueries.Inc()
		}
		idb.logger.Warn("slow database query",
			slog.String("operation", operation),
			slog.Duration("duration", duration),
			slog.String("threshold", idb.slowQueryThreshold.String()),
			slog.Bool("error", err != nil),
		)
	}
}
