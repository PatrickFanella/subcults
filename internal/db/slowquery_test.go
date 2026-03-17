package db

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewSlowQueryMetrics_Register(t *testing.T) {
	m := NewSlowQueryMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Should fail on double registration.
	if err := m.Register(reg); err == nil {
		t.Error("expected error on double registration")
	}
}

func TestInstrumentedDB_RecordDuration_Normal(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	m := NewSlowQueryMetrics()
	reg := prometheus.NewRegistry()
	if err := m.Register(reg); err != nil {
		t.Fatal(err)
	}

	idb := NewInstrumentedDB(InstrumentedDBConfig{
		Logger:  logger,
		Metrics: m,
	})

	// Simulate a fast query.
	idb.recordDuration("select", "SELECT 1", time.Now(), nil)

	// No slow query log expected.
	if bytes.Contains(buf.Bytes(), []byte("slow")) {
		t.Error("unexpected slow query log for fast query")
	}

	// Duration histogram should have 1 observation.
	count := testutil.CollectAndCount(m.queryDuration)
	if count == 0 {
		t.Error("expected non-zero histogram observation")
	}

	// Slow counters should be zero.
	if v := testutil.ToFloat64(m.slowQueries); v != 0 {
		t.Errorf("expected 0 slow queries, got %v", v)
	}
	if v := testutil.ToFloat64(m.verySlowQueries); v != 0 {
		t.Errorf("expected 0 very slow queries, got %v", v)
	}
}

func TestInstrumentedDB_RecordDuration_Slow(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	m := NewSlowQueryMetrics()

	idb := NewInstrumentedDB(InstrumentedDBConfig{
		Logger:                 logger,
		Metrics:                m,
		SlowQueryThreshold:     10 * time.Millisecond,
		VerySlowQueryThreshold: 100 * time.Millisecond,
	})

	// Simulate a slow query (past start time makes duration > threshold).
	past := time.Now().Add(-50 * time.Millisecond)
	idb.recordDuration("scene_list", "SELECT * FROM scenes", past, nil)

	if !bytes.Contains(buf.Bytes(), []byte("slow database query")) {
		t.Error("expected slow query warning in log")
	}
	if bytes.Contains(buf.Bytes(), []byte("very slow")) {
		t.Error("should not be very slow query")
	}

	if v := testutil.ToFloat64(m.slowQueries); v != 1 {
		t.Errorf("expected 1 slow query, got %v", v)
	}
	if v := testutil.ToFloat64(m.verySlowQueries); v != 0 {
		t.Errorf("expected 0 very slow queries, got %v", v)
	}
}

func TestInstrumentedDB_RecordDuration_VerySlow(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	m := NewSlowQueryMetrics()

	idb := NewInstrumentedDB(InstrumentedDBConfig{
		Logger:                 logger,
		Metrics:                m,
		SlowQueryThreshold:     10 * time.Millisecond,
		VerySlowQueryThreshold: 50 * time.Millisecond,
	})

	past := time.Now().Add(-200 * time.Millisecond)
	idb.recordDuration("event_search", "SELECT * FROM events WHERE ...", past, nil)

	if !bytes.Contains(buf.Bytes(), []byte("very slow database query")) {
		t.Error("expected very slow query error in log")
	}

	// Both counters should increment for very slow queries.
	if v := testutil.ToFloat64(m.slowQueries); v != 1 {
		t.Errorf("expected 1 slow query, got %v", v)
	}
	if v := testutil.ToFloat64(m.verySlowQueries); v != 1 {
		t.Errorf("expected 1 very slow query, got %v", v)
	}
}

func TestInstrumentedDB_DefaultThresholds(t *testing.T) {
	idb := NewInstrumentedDB(InstrumentedDBConfig{
		Logger: slog.Default(),
	})

	if idb.slowQueryThreshold != DefaultSlowQueryThreshold {
		t.Errorf("expected default slow threshold %v, got %v", DefaultSlowQueryThreshold, idb.slowQueryThreshold)
	}
	if idb.verySlowQueryThreshold != DefaultVerySlowQueryThreshold {
		t.Errorf("expected default very slow threshold %v, got %v", DefaultVerySlowQueryThreshold, idb.verySlowQueryThreshold)
	}
}

func TestInstrumentedDB_NilMetrics(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Should not panic with nil metrics.
	idb := NewInstrumentedDB(InstrumentedDBConfig{
		Logger:                 logger,
		SlowQueryThreshold:     1 * time.Millisecond,
		VerySlowQueryThreshold: 2 * time.Millisecond,
	})

	past := time.Now().Add(-10 * time.Millisecond)
	idb.recordDuration("test", "SELECT 1", past, nil)

	if !bytes.Contains(buf.Bytes(), []byte("very slow database query")) {
		t.Error("expected log even without metrics")
	}
}

func TestSlowQueryMetrics_Collectors(t *testing.T) {
	m := NewSlowQueryMetrics()
	collectors := m.Collectors()
	if len(collectors) != 3 {
		t.Errorf("expected 3 collectors, got %d", len(collectors))
	}
}

func TestInstrumentedDB_DB(t *testing.T) {
	idb := NewInstrumentedDB(InstrumentedDBConfig{
		Logger: slog.Default(),
	})
	// DB() should return nil since we didn't provide one.
	if idb.DB() != nil {
		t.Error("expected nil DB")
	}
}

func TestInstrumentedDB_RecordDuration_WithError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	m := NewSlowQueryMetrics()

	idb := NewInstrumentedDB(InstrumentedDBConfig{
		Logger:             logger,
		Metrics:            m,
		SlowQueryThreshold: 1 * time.Millisecond,
	})

	past := time.Now().Add(-10 * time.Millisecond)
	idb.recordDuration("insert", "INSERT INTO scenes ...", past, context.DeadlineExceeded)

	// Log should include error=true.
	if !bytes.Contains(buf.Bytes(), []byte("error=true")) {
		t.Error("expected error=true in slow query log")
	}
}
