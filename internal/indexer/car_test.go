package indexer

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func testCARLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// buildCARv1 constructs a minimal CAR v1 file with the given blocks.
func buildCARv1(blocks [][]byte) []byte {
	var buf bytes.Buffer

	// Encode header
	header := CARHeader{Version: 1, Roots: [][]byte{}}
	headerBytes, _ := cbor.Marshal(header)
	writeUvarint(&buf, uint64(len(headerBytes)))
	buf.Write(headerBytes)

	// Encode blocks
	for _, blockData := range blocks {
		// Build a CIDv1 for each block: version=1, codec=0x71 (dag-cbor), sha2-256 hash
		cid := buildTestCID()
		section := append(cid, blockData...)
		writeUvarint(&buf, uint64(len(section)))
		buf.Write(section)
	}
	return buf.Bytes()
}

func writeUvarint(buf *bytes.Buffer, v uint64) {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, v)
	buf.Write(b[:n])
}

// buildTestCID creates a minimal CIDv1 (version=1, codec=0x71 dag-cbor, sha256 placeholder).
func buildTestCID() []byte {
	var cid bytes.Buffer
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, 1) // CID version 1
	cid.Write(b[:n])
	n = binary.PutUvarint(b, 0x71) // dag-cbor codec
	cid.Write(b[:n])
	n = binary.PutUvarint(b, 0x12) // sha2-256
	cid.Write(b[:n])
	n = binary.PutUvarint(b, 32) // 32-byte digest
	cid.Write(b[:n])
	cid.Write(make([]byte, 32)) // zero hash
	return cid.Bytes()
}

func TestCARReader_ValidHeader(t *testing.T) {
	data := buildCARv1(nil)
	reader, err := NewCARReader(bytes.NewReader(data), testCARLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader.Header().Version != 1 {
		t.Errorf("expected version 1, got %d", reader.Header().Version)
	}
}

func TestCARReader_InvalidHeader(t *testing.T) {
	_, err := NewCARReader(bytes.NewReader([]byte{0xFF, 0xFF}), testCARLogger())
	if err == nil {
		t.Fatal("expected error for invalid header")
	}
}

func TestCARReader_UnsupportedVersion(t *testing.T) {
	header := CARHeader{Version: 2, Roots: [][]byte{}}
	headerBytes, _ := cbor.Marshal(header)
	var buf bytes.Buffer
	writeUvarint(&buf, uint64(len(headerBytes)))
	buf.Write(headerBytes)

	_, err := NewCARReader(bytes.NewReader(buf.Bytes()), testCARLogger())
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestCARReader_ReadBlocks(t *testing.T) {
	block1, _ := cbor.Marshal(map[string]string{"key": "value1"})
	block2, _ := cbor.Marshal(map[string]string{"key": "value2"})
	data := buildCARv1([][]byte{block1, block2})

	reader, err := NewCARReader(bytes.NewReader(data), testCARLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := 0
	for {
		_, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error reading block: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 blocks, got %d", count)
	}
}

func TestCARReader_EmptyFile(t *testing.T) {
	data := buildCARv1(nil)
	reader, err := NewCARReader(bytes.NewReader(data), testCARLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = reader.Next()
	if err != io.EOF {
		t.Errorf("expected EOF for empty CAR, got %v", err)
	}
}

func TestCARImporter_DryRun(t *testing.T) {
	repo := NewInMemoryRecordRepository(testCARLogger())
	filter := NewRecordFilter(NewFilterMetrics())
	importer := NewCARImporter(repo, filter, testCARLogger())

	// Create a block with an AT Protocol record
	record := map[string]interface{}{
		"$type":       CollectionScene,
		"name":        "Test Scene",
		"description": "Test description",
	}
	blockData, _ := cbor.Marshal(record)
	data := buildCARv1([][]byte{blockData})
	reader, _ := NewCARReader(bytes.NewReader(data), testCARLogger())

	result, err := importer.Import(context.Background(), reader, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BlocksRead != 1 {
		t.Errorf("expected 1 block read, got %d", result.BlocksRead)
	}
}

func TestCARImporter_SkipsNonMatchingRecords(t *testing.T) {
	repo := NewInMemoryRecordRepository(testCARLogger())
	filter := NewRecordFilter(NewFilterMetrics())
	importer := NewCARImporter(repo, filter, testCARLogger())

	record := map[string]interface{}{
		"$type": "app.other.thing",
		"name":  "Not a subcult record",
	}
	blockData, _ := cbor.Marshal(record)
	data := buildCARv1([][]byte{blockData})
	reader, _ := NewCARReader(bytes.NewReader(data), testCARLogger())

	result, err := importer.Import(context.Background(), reader, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RecordsSkipped != 1 {
		t.Errorf("expected 1 record skipped, got %d", result.RecordsSkipped)
	}
}

func TestCARImporter_CancelsOnContext(t *testing.T) {
	repo := NewInMemoryRecordRepository(testCARLogger())
	filter := NewRecordFilter(NewFilterMetrics())
	importer := NewCARImporter(repo, filter, testCARLogger())

	block, _ := cbor.Marshal(map[string]string{"key": "value"})
	data := buildCARv1([][]byte{block})
	reader, _ := NewCARReader(bytes.NewReader(data), testCARLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := importer.Import(ctx, reader, false)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
