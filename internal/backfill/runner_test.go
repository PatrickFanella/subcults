package backfill

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/onnwee/subcults/internal/indexer"
)

func TestRunner_JetstreamCreatesCheckpoint(t *testing.T) {
	store := newInMemoryCheckpointStore()
	repo := newTestRepo()
	filter := newTestFilter()
	cfg := Config{
		Source:    "jetstream",
		StartTS:  1000000,
		EndTS:    2000000,
		BatchSize: 100,
		Logger:   newTestLogger(),
	}
	runner := NewRunner(cfg, repo, filter, store)
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	cp, err := store.GetLatest(context.Background(), "jetstream")
	if err != nil {
		t.Fatalf("unexpected error getting checkpoint: %v", err)
	}
	if cp == nil {
		t.Fatal("expected checkpoint to be created")
	}
	if cp.Status != "completed" {
		t.Errorf("expected status completed, got %s", cp.Status)
	}
}

func TestRunner_CARCreatesCheckpoint(t *testing.T) {
	// Create a temporary CAR file
	tmpFile, err := os.CreateTemp("", "test-*.car")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	carData := buildTestCARv1(nil)
	if _, err := tmpFile.Write(carData); err != nil {
		t.Fatalf("failed to write CAR data: %v", err)
	}
	tmpFile.Close()

	store := newInMemoryCheckpointStore()
	repo := newTestRepo()
	filter := newTestFilter()
	cfg := Config{
		Source:    "car",
		CARPath:  tmpFile.Name(),
		BatchSize: 100,
		Logger:   newTestLogger(),
	}
	runner := NewRunner(cfg, repo, filter, store)
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	cp, err := store.GetLatest(context.Background(), "car")
	if err != nil {
		t.Fatalf("unexpected error getting checkpoint: %v", err)
	}
	if cp == nil {
		t.Fatal("expected checkpoint to be created")
	}
	if cp.Status != "completed" {
		t.Errorf("expected status completed, got %s", cp.Status)
	}
}

func TestRunner_InvalidSource(t *testing.T) {
	store := newInMemoryCheckpointStore()
	repo := newTestRepo()
	filter := newTestFilter()
	cfg := Config{
		Source: "unknown",
		Logger: newTestLogger(),
	}
	runner := NewRunner(cfg, repo, filter, store)
	_, err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
}

func TestRunner_DefaultBatchSize(t *testing.T) {
	store := newInMemoryCheckpointStore()
	repo := newTestRepo()
	filter := newTestFilter()
	cfg := Config{
		Source: "jetstream",
		Logger: newTestLogger(),
	}
	runner := NewRunner(cfg, repo, filter, store)
	if runner.config.BatchSize != 1000 {
		t.Errorf("expected default batch size 1000, got %d", runner.config.BatchSize)
	}
}

func TestRunner_ProcessRecord_MatchingCollection(t *testing.T) {
	store := newInMemoryCheckpointStore()
	repo := newTestRepo()
	filter := newTestFilter()
	cfg := Config{
		Source: "jetstream",
		Logger: newTestLogger(),
	}
	runner := NewRunner(cfg, repo, filter, store)
	payload, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Scene",
		"description": "A test scene",
	})
	err := runner.ProcessRecord(context.Background(), indexer.CollectionScene, payload, "did:plc:test", "abc123", "rev1")
	if err != nil {
		t.Fatalf("unexpected error processing record: %v", err)
	}
}

func TestRunner_ProcessRecord_NonMatchingCollection(t *testing.T) {
	store := newInMemoryCheckpointStore()
	repo := newTestRepo()
	filter := newTestFilter()
	cfg := Config{
		Source: "jetstream",
		Logger: newTestLogger(),
	}
	runner := NewRunner(cfg, repo, filter, store)
	err := runner.ProcessRecord(context.Background(), "app.other.thing", []byte(`{}`), "did:plc:test", "abc123", "rev1")
	if err != nil {
		t.Fatalf("non-matching collection should not error: %v", err)
	}
}

func TestCheckpointStore_CreateAndGet(t *testing.T) {
	store := newInMemoryCheckpointStore()
	ctx := context.Background()
	id, err := store.Create(ctx, "jetstream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero checkpoint ID")
	}
	cp, err := store.GetLatest(ctx, "jetstream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp == nil {
		t.Fatal("expected checkpoint")
	}
	if cp.Source != "jetstream" {
		t.Errorf("expected source jetstream, got %s", cp.Source)
	}
	if cp.Status != "running" {
		t.Errorf("expected status running, got %s", cp.Status)
	}
}

func TestCheckpointStore_CompleteAndFail(t *testing.T) {
	store := newInMemoryCheckpointStore()
	ctx := context.Background()
	id1, _ := store.Create(ctx, "jetstream")
	err := store.Complete(ctx, id1, 100, 5, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cp := store.get(id1)
	if cp.Status != "completed" {
		t.Errorf("expected status completed, got %s", cp.Status)
	}
	if cp.RecordsProcessed != 100 {
		t.Errorf("expected 100 processed, got %d", cp.RecordsProcessed)
	}
	id2, _ := store.Create(ctx, "car")
	err = store.Fail(ctx, id2, 50, 3, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cp2 := store.get(id2)
	if cp2.Status != "failed" {
		t.Errorf("expected status failed, got %s", cp2.Status)
	}
}

func TestCheckpointStore_GetLatest_NoCheckpoints(t *testing.T) {
	store := newInMemoryCheckpointStore()
	cp, err := store.GetLatest(context.Background(), "jetstream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp != nil {
		t.Error("expected nil checkpoint for empty store")
	}
}

func TestRunner_DryRun(t *testing.T) {
	store := newInMemoryCheckpointStore()
	repo := newTestRepo()
	filter := newTestFilter()
	cfg := Config{
		Source: "jetstream",
		DryRun: true,
		Logger: newTestLogger(),
	}
	runner := NewRunner(cfg, repo, filter, store)
	payload, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Scene",
		"description": "A test scene",
	})
	err := runner.ProcessRecord(context.Background(), indexer.CollectionScene, payload, "did:plc:test", "abc123", "rev1")
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
}

// buildTestCARv1 constructs a minimal CAR v1 file for testing.
func buildTestCARv1(blocks [][]byte) []byte {
	var buf bytes.Buffer
	header := struct {
		Version int      `cbor:"version"`
		Roots   [][]byte `cbor:"roots"`
	}{Version: 1, Roots: [][]byte{}}
	headerBytes, _ := cbor.Marshal(header)
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, uint64(len(headerBytes)))
	buf.Write(b[:n])
	buf.Write(headerBytes)
	for _, blockData := range blocks {
		cid := buildTestCIDv1()
		section := append(cid, blockData...)
		n := binary.PutUvarint(b, uint64(len(section)))
		buf.Write(b[:n])
		buf.Write(section)
	}
	return buf.Bytes()
}

func buildTestCIDv1() []byte {
	var cid bytes.Buffer
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, 1)
	cid.Write(b[:n])
	n = binary.PutUvarint(b, 0x71)
	cid.Write(b[:n])
	n = binary.PutUvarint(b, 0x12)
	cid.Write(b[:n])
	n = binary.PutUvarint(b, 32)
	cid.Write(b[:n])
	cid.Write(make([]byte, 32))
	return cid.Bytes()
}
