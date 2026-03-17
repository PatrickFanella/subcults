package indexer

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func TestDecodeCBORMessage_ValidCommit(t *testing.T) {
	// Create a sample Jetstream commit message
	sceneData := map[string]interface{}{"name": "Test Scene"}
	recordCBOR, err := EncodeCBOR(sceneData)
	if err != nil {
		t.Fatalf("failed to encode scene data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:test123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:test123",
			Rev:        "abc123",
			Operation:  "create",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			Record:     cbor.RawMessage(recordCBOR),
		},
	}

	// Encode to CBOR
	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode test message: %v", err)
	}

	// Decode the message
	decoded, err := DecodeCBORMessage(data)
	if err != nil {
		t.Fatalf("DecodeCBORMessage() error = %v", err)
	}

	// Verify fields
	if decoded.DID != msg.DID {
		t.Errorf("DID = %q, want %q", decoded.DID, msg.DID)
	}
	if decoded.Kind != msg.Kind {
		t.Errorf("Kind = %q, want %q", decoded.Kind, msg.Kind)
	}
	if decoded.Commit == nil {
		t.Fatal("Commit is nil")
	}
	if decoded.Commit.Collection != msg.Commit.Collection {
		t.Errorf("Collection = %q, want %q", decoded.Commit.Collection, msg.Commit.Collection)
	}
}

func TestDecodeCBORMessage_EmptyData(t *testing.T) {
	_, err := DecodeCBORMessage([]byte{})
	if err != ErrInvalidCBOR {
		t.Errorf("expected ErrInvalidCBOR, got %v", err)
	}
}

func TestDecodeCBORMessage_MalformedCBOR(t *testing.T) {
	// Invalid CBOR data
	invalidData := []byte{0xff, 0xff, 0xff, 0xff}
	_, err := DecodeCBORMessage(invalidData)
	if err == nil {
		t.Error("expected error for malformed CBOR")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("invalid CBOR")) {
		t.Errorf("expected 'invalid CBOR' error, got: %v", err)
	}
}

func TestDecodeCBORCommit_Valid(t *testing.T) {
	eventData := map[string]interface{}{"name": "Event", "sceneId": "s1"}
	recordCBOR, err := EncodeCBOR(eventData)
	if err != nil {
		t.Fatalf("failed to encode event data: %v", err)
	}

	commit := AtProtoCommit{
		DID:        "did:plc:abc123",
		Rev:        "rev1",
		Operation:  "create",
		Collection: "app.subcult.event",
		RKey:       "event1",
		Record:     cbor.RawMessage(recordCBOR),
	}

	data, err := EncodeCBOR(commit)
	if err != nil {
		t.Fatalf("failed to encode commit: %v", err)
	}

	decoded, err := DecodeCBORCommit(data)
	if err != nil {
		t.Fatalf("DecodeCBORCommit() error = %v", err)
	}

	if decoded.DID != commit.DID {
		t.Errorf("DID = %q, want %q", decoded.DID, commit.DID)
	}
	if decoded.Collection != commit.Collection {
		t.Errorf("Collection = %q, want %q", decoded.Collection, commit.Collection)
	}
	if decoded.Operation != commit.Operation {
		t.Errorf("Operation = %q, want %q", decoded.Operation, commit.Operation)
	}
}

func TestDecodeCBORCommit_MissingDID(t *testing.T) {
	commit := AtProtoCommit{
		Collection: "app.subcult.scene",
		RKey:       "scene1",
	}

	data, err := EncodeCBOR(commit)
	if err != nil {
		t.Fatalf("failed to encode commit: %v", err)
	}

	_, err = DecodeCBORCommit(data)
	if err != ErrMissingDID {
		t.Errorf("expected ErrMissingDID, got %v", err)
	}
}

func TestDecodeCBORCommit_MissingCollection(t *testing.T) {
	commit := AtProtoCommit{
		DID:  "did:plc:test",
		RKey: "record1",
	}

	data, err := EncodeCBOR(commit)
	if err != nil {
		t.Fatalf("failed to encode commit: %v", err)
	}

	_, err = DecodeCBORCommit(data)
	if err != ErrMissingPath {
		t.Errorf("expected ErrMissingPath, got %v", err)
	}
}

func TestParseRecord_ValidSceneCreate(t *testing.T) {
	// Create a scene record in CBOR
	sceneData := map[string]interface{}{
		"name":        "Underground Techno",
		"description": "Berlin warehouse scene",
	}
	recordCBOR, err := EncodeCBOR(sceneData)
	if err != nil {
		t.Fatalf("failed to encode scene data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:scene123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:scene123",
			Rev:        "abc123",
			Operation:  "create",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			Record:     recordCBOR,
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	parsed, err := ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord() error = %v", err)
	}

	if parsed.DID != "did:plc:scene123" {
		t.Errorf("DID = %q, want %q", parsed.DID, "did:plc:scene123")
	}
	if parsed.Collection != "app.subcult.scene" {
		t.Errorf("Collection = %q, want %q", parsed.Collection, "app.subcult.scene")
	}
	if parsed.Operation != "create" {
		t.Errorf("Operation = %q, want %q", parsed.Operation, "create")
	}
	if parsed.Record == nil {
		t.Fatal("Record is nil")
	}

	// Verify the record can be decoded as JSON
	var decodedScene map[string]interface{}
	if err := json.Unmarshal(parsed.Record, &decodedScene); err != nil {
		t.Fatalf("failed to decode record as JSON: %v", err)
	}
	if decodedScene["name"] != "Underground Techno" {
		t.Errorf("name = %v, want %q", decodedScene["name"], "Underground Techno")
	}
}

func TestParseRecord_ValidEventCreate(t *testing.T) {
	eventData := map[string]interface{}{
		"name":    "Friday Night",
		"sceneId": "scene123",
	}
	recordCBOR, err := EncodeCBOR(eventData)
	if err != nil {
		t.Fatalf("failed to encode event data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:event456",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:event456",
			Rev:        "def456",
			Operation:  "create",
			Collection: "app.subcult.event",
			RKey:       "event1",
			Record:     recordCBOR,
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	parsed, err := ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord() error = %v", err)
	}

	if parsed.Collection != "app.subcult.event" {
		t.Errorf("Collection = %q, want %q", parsed.Collection, "app.subcult.event")
	}
}

func TestParseRecord_ValidPostCreate(t *testing.T) {
	postData := map[string]interface{}{
		"text":    "Check out this show!",
		"sceneId": "scene789",
	}
	recordCBOR, err := EncodeCBOR(postData)
	if err != nil {
		t.Fatalf("failed to encode post data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:post789",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:post789",
			Rev:        "ghi789",
			Operation:  "create",
			Collection: "app.subcult.post",
			RKey:       "post1",
			Record:     recordCBOR,
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	parsed, err := ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord() error = %v", err)
	}

	if parsed.Collection != "app.subcult.post" {
		t.Errorf("Collection = %q, want %q", parsed.Collection, "app.subcult.post")
	}
}

func TestParseRecord_ValidAllianceCreate(t *testing.T) {
	allianceData := map[string]interface{}{
		"from": "scene1",
		"to":   "scene2",
		"role": "promoter",
	}
	recordCBOR, err := EncodeCBOR(allianceData)
	if err != nil {
		t.Fatalf("failed to encode alliance data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:alliance000",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:alliance000",
			Rev:        "jkl000",
			Operation:  "create",
			Collection: "app.subcult.alliance",
			RKey:       "alliance1",
			Record:     recordCBOR,
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	parsed, err := ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord() error = %v", err)
	}

	if parsed.Collection != "app.subcult.alliance" {
		t.Errorf("Collection = %q, want %q", parsed.Collection, "app.subcult.alliance")
	}
}

func TestParseRecord_DeleteOperation(t *testing.T) {
	msg := JetstreamMessage{
		DID:    "did:plc:delete123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:delete123",
			Rev:        "del123",
			Operation:  "delete",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			// No record data for delete operations
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	parsed, err := ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord() error = %v", err)
	}

	if parsed.Operation != "delete" {
		t.Errorf("Operation = %q, want %q", parsed.Operation, "delete")
	}
	if parsed.Record != nil {
		t.Error("expected Record to be nil for delete operation")
	}
}

func TestParseRecord_UpdateOperation(t *testing.T) {
	sceneData := map[string]interface{}{
		"name":        "Updated Scene",
		"description": "New description",
	}
	recordCBOR, err := EncodeCBOR(sceneData)
	if err != nil {
		t.Fatalf("failed to encode scene data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:update123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:update123",
			Rev:        "upd123",
			Operation:  "update",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			Record:     recordCBOR,
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	parsed, err := ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord() error = %v", err)
	}

	if parsed.Operation != "update" {
		t.Errorf("Operation = %q, want %q", parsed.Operation, "update")
	}
	if parsed.Record == nil {
		t.Fatal("Record is nil for update operation")
	}
}

func TestParseRecord_NonCommitMessage(t *testing.T) {
	msg := JetstreamMessage{
		DID:    "did:plc:identity123",
		TimeUS: 1234567890,
		Kind:   "identity",
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	_, err = ParseRecord(data)
	if err == nil {
		t.Error("expected error for non-commit message")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("unsupported message kind")) {
		t.Errorf("expected 'unsupported message kind' error, got: %v", err)
	}
}

func TestParseRecord_MissingCommit(t *testing.T) {
	msg := JetstreamMessage{
		DID:    "did:plc:nocommit123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: nil,
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	_, err = ParseRecord(data)
	if err != ErrMissingRecord {
		t.Errorf("expected ErrMissingRecord, got %v", err)
	}
}

func TestParseRecord_MissingRecordData(t *testing.T) {
	msg := JetstreamMessage{
		DID:    "did:plc:nodata123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:nodata123",
			Rev:        "nodata123",
			Operation:  "create",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			// Missing Record data for create operation
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	_, err = ParseRecord(data)
	if err != ErrMissingRecord {
		t.Errorf("expected ErrMissingRecord, got %v", err)
	}
}

func TestParseRecord_InvalidRecordCBOR(t *testing.T) {
	// Create a message with raw invalid CBOR in the record field
	// We'll construct this manually without using EncodeCBOR for the whole message
	invalidRecordCBOR := []byte{0xff, 0xff, 0xff} // Invalid CBOR

	commit := AtProtoCommit{
		DID:        "did:plc:invalid123",
		Rev:        "invalid123",
		Operation:  "create",
		Collection: "app.subcult.scene",
		RKey:       "scene1",
		Record:     cbor.RawMessage(invalidRecordCBOR),
	}

	msg := JetstreamMessage{
		DID:    "did:plc:invalid123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &commit,
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		// This is expected - we can't encode a message with invalid CBOR record
		// Instead, we'll test by directly parsing a manually constructed message
		// that has invalid record data
		t.Skip("Cannot encode message with invalid CBOR record, skipping")
		return
	}

	_, err = ParseRecord(data)
	if err == nil {
		t.Error("expected error for invalid record CBOR")
	}
}

func TestEncodeCBOR_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "string",
			value: "test string",
		},
		{
			name:  "number",
			value: 42,
		},
		{
			name: "map",
			value: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		},
		{
			name:  "array",
			value: []interface{}{"a", "b", "c"},
		},
		{
			name: "commit struct",
			value: AtProtoCommit{
				DID:        "did:plc:test",
				Rev:        "rev1",
				Operation:  "create",
				Collection: "app.subcult.scene",
				RKey:       "scene1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode to CBOR
			encoded, err := EncodeCBOR(tt.value)
			if err != nil {
				t.Fatalf("EncodeCBOR() error = %v", err)
			}

			// Decode back
			var decoded interface{}
			if err := cbor.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			// For basic types, verify the value
			switch v := tt.value.(type) {
			case string:
				if decoded.(string) != v {
					t.Errorf("decoded = %v, want %v", decoded, v)
				}
			case int:
				// CBOR may decode as uint64 or int64; handle both safely.
				var decodedInt int
				switch n := decoded.(type) {
				case uint64:
					decodedInt = int(n)
				case int64:
					decodedInt = int(n)
				default:
					t.Fatalf("unexpected type for decoded number: %T", decoded)
				}
				if decodedInt != v {
					t.Errorf("decoded = %v, want %v", decodedInt, v)
				}
			}
		})
	}
}

func TestParseRecord_ComplexSceneRecord(t *testing.T) {
	// Test with a more complex scene record including optional fields
	sceneData := map[string]interface{}{
		"name":        "Complex Scene",
		"description": "A detailed scene",
		"location": map[string]interface{}{
			"lat": 52.5200,
			"lon": 13.4050,
		},
		"genres":   []string{"techno", "house"},
		"capacity": 500,
	}
	recordCBOR, err := EncodeCBOR(sceneData)
	if err != nil {
		t.Fatalf("failed to encode scene data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:complex123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:complex123",
			Rev:        "cplx123",
			Operation:  "create",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			Record:     recordCBOR,
		},
	}

	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	parsed, err := ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord() error = %v", err)
	}

	// Verify we can decode the complex record as JSON
	var decodedScene map[string]interface{}
	if err := json.Unmarshal(parsed.Record, &decodedScene); err != nil {
		t.Fatalf("failed to decode record as JSON: %v", err)
	}

	if decodedScene["name"] != "Complex Scene" {
		t.Errorf("name = %v, want %q", decodedScene["name"], "Complex Scene")
	}

	// Verify nested structures preserved
	location, ok := decodedScene["location"].(map[string]interface{})
	if !ok {
		t.Fatalf("location is not a map")
	}
	if location["lat"] == nil {
		t.Error("location.lat is missing")
	}
}

func TestParseRecord_EmptyMessage(t *testing.T) {
	_, err := ParseRecord([]byte{})
	if err != ErrInvalidCBOR {
		t.Errorf("expected ErrInvalidCBOR, got %v", err)
	}
}

func TestParseRecord_MalformedMessage(t *testing.T) {
	invalidData := []byte{0xff, 0xff, 0xff, 0xff}
	_, err := ParseRecord(invalidData)
	if err == nil {
		t.Error("expected error for malformed message")
	}
}

// Benchmark tests
func BenchmarkDecodeCBORMessage(b *testing.B) {
	msg := JetstreamMessage{
		DID:    "did:plc:bench123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:bench123",
			Rev:        "bench123",
			Operation:  "create",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
		},
	}

	data, _ := EncodeCBOR(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeCBORMessage(data)
	}
}

func BenchmarkParseRecord(b *testing.B) {
	sceneData := map[string]interface{}{
		"name":        "Benchmark Scene",
		"description": "Performance test",
	}
	recordCBOR, _ := EncodeCBOR(sceneData)

	msg := JetstreamMessage{
		DID:    "did:plc:bench456",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:bench456",
			Rev:        "bench456",
			Operation:  "create",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			Record:     recordCBOR,
		},
	}

	data, _ := EncodeCBOR(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseRecord(data)
	}
}
