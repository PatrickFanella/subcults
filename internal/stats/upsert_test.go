package stats

import (
	"bytes"
	"log/slog"
	"sync"
	"testing"
)

func TestUpsertStats_RecordInsert(t *testing.T) {
	stats := NewUpsertStats()

	stats.RecordInsert()
	if stats.Inserted() != 1 {
		t.Errorf("Expected 1 insert, got %d", stats.Inserted())
	}

	stats.RecordInsert()
	if stats.Inserted() != 2 {
		t.Errorf("Expected 2 inserts, got %d", stats.Inserted())
	}
}

func TestUpsertStats_RecordUpdate(t *testing.T) {
	stats := NewUpsertStats()

	stats.RecordUpdate()
	if stats.Updated() != 1 {
		t.Errorf("Expected 1 update, got %d", stats.Updated())
	}

	stats.RecordUpdate()
	if stats.Updated() != 2 {
		t.Errorf("Expected 2 updates, got %d", stats.Updated())
	}
}

func TestUpsertStats_Total(t *testing.T) {
	stats := NewUpsertStats()

	stats.RecordInsert()
	stats.RecordInsert()
	stats.RecordUpdate()

	if stats.Total() != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total())
	}
}

func TestUpsertStats_Reset(t *testing.T) {
	stats := NewUpsertStats()

	stats.RecordInsert()
	stats.RecordUpdate()
	stats.Reset()

	if stats.Inserted() != 0 {
		t.Errorf("Expected 0 inserts after reset, got %d", stats.Inserted())
	}

	if stats.Updated() != 0 {
		t.Errorf("Expected 0 updates after reset, got %d", stats.Updated())
	}
}

func TestUpsertStats_String(t *testing.T) {
	stats := NewUpsertStats()

	stats.RecordInsert()
	stats.RecordInsert()
	stats.RecordUpdate()

	expected := "inserted=2 updated=1 total=3"
	if stats.String() != expected {
		t.Errorf("Expected %q, got %q", expected, stats.String())
	}
}

func TestUpsertStats_Concurrent(t *testing.T) {
	stats := NewUpsertStats()
	var wg sync.WaitGroup

	// Simulate concurrent inserts and updates
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			stats.RecordInsert()
		}()
		go func() {
			defer wg.Done()
			stats.RecordUpdate()
		}()
	}

	wg.Wait()

	if stats.Inserted() != 100 {
		t.Errorf("Expected 100 inserts, got %d", stats.Inserted())
	}

	if stats.Updated() != 100 {
		t.Errorf("Expected 100 updates, got %d", stats.Updated())
	}

	if stats.Total() != 200 {
		t.Errorf("Expected total 200, got %d", stats.Total())
	}
}

func TestUpsertStats_LogSummary(t *testing.T) {
	stats := NewUpsertStats()
	stats.RecordInsert()
	stats.RecordInsert()
	stats.RecordUpdate()

	// Create a logger that writes to a buffer
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))

	stats.LogSummary(logger, "test_entity")

	output := buf.String()
	if output == "" {
		t.Error("Expected log output, got empty string")
	}

	// Check that key fields are present in the log
	expectedFields := []string{"entity", "test_entity", "inserted", "updated", "total"}
	for _, field := range expectedFields {
		if !bytes.Contains(buf.Bytes(), []byte(field)) {
			t.Errorf("Expected log to contain %q", field)
		}
	}
}
