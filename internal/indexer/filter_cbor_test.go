package indexer

import (
	"testing"
)

func TestRecordFilter_FilterCBOR_ValidSceneCreate(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Create a valid scene record
	sceneData := map[string]interface{}{
		"name":        "Test Scene",
		"description": "A test scene",
	}
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
			Record:     recordCBOR,
		},
	}

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if !result.Matched {
		t.Error("expected Matched = true")
	}
	if !result.Valid {
		t.Errorf("expected Valid = true, got error: %v", result.Error)
	}
	if result.DID != "did:plc:test123" {
		t.Errorf("DID = %q, want %q", result.DID, "did:plc:test123")
	}
	if result.Collection != "app.subcult.scene" {
		t.Errorf("Collection = %q, want %q", result.Collection, "app.subcult.scene")
	}
	if result.RKey != "scene1" {
		t.Errorf("RKey = %q, want %q", result.RKey, "scene1")
	}
	if result.Operation != "create" {
		t.Errorf("Operation = %q, want %q", result.Operation, "create")
	}
	if result.Record == nil {
		t.Error("Record is nil")
	}
}

func TestRecordFilter_FilterCBOR_ValidEventCreate(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

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

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if !result.Matched || !result.Valid {
		t.Errorf("expected valid event, got Matched=%v Valid=%v error=%v", result.Matched, result.Valid, result.Error)
	}
	if result.Collection != "app.subcult.event" {
		t.Errorf("Collection = %q, want %q", result.Collection, "app.subcult.event")
	}
}

func TestRecordFilter_FilterCBOR_ValidPostCreate(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

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

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if !result.Matched || !result.Valid {
		t.Errorf("expected valid post, got Matched=%v Valid=%v error=%v", result.Matched, result.Valid, result.Error)
	}
	if result.Collection != "app.subcult.post" {
		t.Errorf("Collection = %q, want %q", result.Collection, "app.subcult.post")
	}
}

func TestRecordFilter_FilterCBOR_ValidAllianceCreate(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	allianceData := map[string]interface{}{
		"fromSceneId": "scene1",
		"toSceneId":   "scene2",
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

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	// Alliance collection doesn't have specific validation, so it should match with valid JSON
	if !result.Matched || !result.Valid {
		t.Errorf("expected valid alliance, got Matched=%v Valid=%v error=%v", result.Matched, result.Valid, result.Error)
	}
	if result.Collection != "app.subcult.alliance" {
		t.Errorf("Collection = %q, want %q", result.Collection, "app.subcult.alliance")
	}
}

func TestRecordFilter_FilterCBOR_DeleteOperation(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

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
		},
	}

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if !result.Matched {
		t.Error("expected Matched = true")
	}
	if !result.Valid {
		t.Errorf("expected Valid = true for delete operation, got error: %v", result.Error)
	}
	if result.Operation != "delete" {
		t.Errorf("Operation = %q, want %q", result.Operation, "delete")
	}
	if result.Record != nil {
		t.Error("expected Record to be nil for delete operation")
	}
}

func TestRecordFilter_FilterCBOR_NonMatchingCollection(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Create a bsky post (non-subcult collection)
	postData := map[string]interface{}{
		"text": "Hello Bluesky",
	}
	recordCBOR, err := EncodeCBOR(postData)
	if err != nil {
		t.Fatalf("failed to encode post data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:bsky123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:bsky123",
			Rev:        "bsky123",
			Operation:  "create",
			Collection: "app.bsky.feed.post",
			RKey:       "post1",
			Record:     recordCBOR,
		},
	}

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if result.Matched {
		t.Error("expected Matched = false for bsky collection")
	}
	if result.Valid {
		t.Error("expected Valid = false for non-matching collection")
	}
	if result.Error != ErrNonMatchingLexicon {
		t.Errorf("expected error %v, got %v", ErrNonMatchingLexicon, result.Error)
	}
}

func TestRecordFilter_FilterCBOR_InvalidSceneRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Scene missing required "name" field
	sceneData := map[string]interface{}{
		"description": "Missing name field",
	}
	recordCBOR, err := EncodeCBOR(sceneData)
	if err != nil {
		t.Fatalf("failed to encode scene data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:invalid123",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:invalid123",
			Rev:        "inv123",
			Operation:  "create",
			Collection: "app.subcult.scene",
			RKey:       "scene1",
			Record:     recordCBOR,
		},
	}

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if !result.Matched {
		t.Error("expected Matched = true for matching collection")
	}
	if result.Valid {
		t.Error("expected Valid = false for invalid scene record")
	}
	if result.Error != ErrMissingField {
		t.Errorf("expected error %v, got %v", ErrMissingField, result.Error)
	}
}

func TestRecordFilter_FilterCBOR_InvalidEventRecord(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Event missing required "sceneId" field
	eventData := map[string]interface{}{
		"name": "Event without sceneId",
	}
	recordCBOR, err := EncodeCBOR(eventData)
	if err != nil {
		t.Fatalf("failed to encode event data: %v", err)
	}

	msg := JetstreamMessage{
		DID:    "did:plc:invalid456",
		TimeUS: 1234567890,
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:invalid456",
			Rev:        "inv456",
			Operation:  "create",
			Collection: "app.subcult.event",
			RKey:       "event1",
			Record:     recordCBOR,
		},
	}

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if !result.Matched {
		t.Error("expected Matched = true for matching collection")
	}
	if result.Valid {
		t.Error("expected Valid = false for invalid event record")
	}
	if result.Error != ErrMissingField {
		t.Errorf("expected error %v, got %v", ErrMissingField, result.Error)
	}
}

func TestRecordFilter_FilterCBOR_MalformedCBOR(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Invalid CBOR data
	invalidCBOR := []byte{0xff, 0xff, 0xff, 0xff}

	result := filter.FilterCBOR(invalidCBOR)

	if result.Matched {
		t.Error("expected Matched = false for malformed CBOR")
	}
	if result.Valid {
		t.Error("expected Valid = false for malformed CBOR")
	}
	if result.Error == nil {
		t.Error("expected error for malformed CBOR")
	}
}

func TestRecordFilter_FilterCBOR_NonCommitMessage(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	msg := JetstreamMessage{
		DID:    "did:plc:identity123",
		TimeUS: 1234567890,
		Kind:   "identity",
	}

	cborData, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	result := filter.FilterCBOR(cborData)

	if result.Matched {
		t.Error("expected Matched = false for non-commit message")
	}
	if result.Valid {
		t.Error("expected Valid = false for non-commit message")
	}
	if result.Error == nil {
		t.Error("expected error for non-commit message")
	}
}

func TestRecordFilter_FilterCBOR_MetricsTracking(t *testing.T) {
	metrics := NewFilterMetrics()
	filter := NewRecordFilter(metrics)

	// Process a mix of records
	tests := []struct {
		name       string
		collection string
		data       map[string]interface{}
		wantValid  bool
	}{
		{
			name:       "valid scene",
			collection: "app.subcult.scene",
			data:       map[string]interface{}{"name": "Test"},
			wantValid:  true,
		},
		{
			name:       "invalid scene",
			collection: "app.subcult.scene",
			data:       map[string]interface{}{"description": "No name"},
			wantValid:  false,
		},
		{
			name:       "valid event",
			collection: "app.subcult.event",
			data:       map[string]interface{}{"name": "Event", "sceneId": "s1"},
			wantValid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recordCBOR, err := EncodeCBOR(tt.data)
			if err != nil {
				t.Fatalf("failed to encode record data: %v", err)
			}
			msg := JetstreamMessage{
				DID:    "did:plc:metrics123",
				TimeUS: 1234567890,
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        "did:plc:metrics123",
					Rev:        "met123",
					Operation:  "create",
					Collection: tt.collection,
					RKey:       "record1",
					Record:     recordCBOR,
				},
			}
			cborData, err := EncodeCBOR(msg)
			if err != nil {
				t.Fatalf("failed to encode message: %v", err)
			}
			result := filter.FilterCBOR(cborData)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}

	// Verify metrics
	if metrics.Processed() != 3 {
		t.Errorf("Processed() = %d, want 3", metrics.Processed())
	}
	if metrics.Matched() != 3 {
		t.Errorf("Matched() = %d, want 3", metrics.Matched())
	}
	if metrics.Discarded() != 1 {
		t.Errorf("Discarded() = %d, want 1", metrics.Discarded())
	}
}
