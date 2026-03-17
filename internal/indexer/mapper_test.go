package indexer

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/scene"
)

func TestMapSceneRecord_ValidMinimal(t *testing.T) {
	// Minimal valid scene record
	sceneJSON := `{"name":"Underground Techno"}`

	record := &FilterResult{
		DID:        "did:plc:scene123",
		Collection: CollectionScene,
		RKey:       "scene1",
		Record:     json.RawMessage(sceneJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapSceneRecord(record)
	if err != nil {
		t.Fatalf("MapSceneRecord() error = %v", err)
	}

	// Verify required fields
	if result.Name != "Underground Techno" {
		t.Errorf("Name = %q, want %q", result.Name, "Underground Techno")
	}
	if result.OwnerDID != "did:plc:scene123" {
		t.Errorf("OwnerDID = %q, want %q", result.OwnerDID, "did:plc:scene123")
	}
	if result.RecordDID == nil || *result.RecordDID != "did:plc:scene123" {
		t.Errorf("RecordDID not set correctly")
	}
	if result.RecordRKey == nil || *result.RecordRKey != "scene1" {
		t.Errorf("RecordRKey not set correctly")
	}

	// Verify defaults
	if result.Visibility != scene.VisibilityPublic {
		t.Errorf("Visibility = %q, want %q", result.Visibility, scene.VisibilityPublic)
	}
	if result.CoarseGeohash != "u4pruyd" {
		t.Errorf("CoarseGeohash = %q, want default", result.CoarseGeohash)
	}
	if result.AllowPrecise {
		t.Error("AllowPrecise should default to false")
	}
}

func TestMapSceneRecord_ValidComplete(t *testing.T) {
	// Complete scene record with all fields
	sceneJSON := `{
		"name": "Underground Techno",
		"description": "Berlin warehouse scene",
		"location": {
			"lat": 52.52,
			"lng": 13.405,
			"allowPrecise": true
		},
		"tags": ["techno", "underground", "berlin"],
		"visibility": "public",
		"palette": {
			"primary": "#FF0000",
			"secondary": "#00FF00",
			"accent": "#0000FF",
			"background": "#FFFFFF",
			"text": "#000000"
		}
	}`

	record := &FilterResult{
		DID:        "did:plc:scene123",
		Collection: CollectionScene,
		RKey:       "scene1",
		Record:     json.RawMessage(sceneJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapSceneRecord(record)
	if err != nil {
		t.Fatalf("MapSceneRecord() error = %v", err)
	}

	// Verify all fields
	if result.Name != "Underground Techno" {
		t.Errorf("Name = %q, want %q", result.Name, "Underground Techno")
	}
	if result.Description != "Berlin warehouse scene" {
		t.Errorf("Description = %q, want %q", result.Description, "Berlin warehouse scene")
	}
	if !result.AllowPrecise {
		t.Error("AllowPrecise should be true")
	}
	if result.PrecisePoint == nil {
		t.Fatal("PrecisePoint should not be nil")
	}
	if result.PrecisePoint.Lat != 52.52 {
		t.Errorf("Lat = %f, want 52.52", result.PrecisePoint.Lat)
	}
	if result.PrecisePoint.Lng != 13.405 {
		t.Errorf("Lng = %f, want 13.405", result.PrecisePoint.Lng)
	}
	if len(result.Tags) != 3 {
		t.Errorf("Tags length = %d, want 3", len(result.Tags))
	}
	if result.Visibility != "public" {
		t.Errorf("Visibility = %q, want public", result.Visibility)
	}
	if result.Palette == nil {
		t.Fatal("Palette should not be nil")
	}
	if result.Palette.Primary != "#FF0000" {
		t.Errorf("Primary color = %q, want #FF0000", result.Palette.Primary)
	}
}

func TestMapSceneRecord_LocationConsentEnforced(t *testing.T) {
	// Scene with location but allowPrecise=false
	sceneJSON := `{
		"name": "Private Scene",
		"location": {
			"lat": 52.52,
			"lng": 13.405,
			"allowPrecise": false
		}
	}`

	record := &FilterResult{
		DID:        "did:plc:scene456",
		Collection: CollectionScene,
		RKey:       "scene2",
		Record:     json.RawMessage(sceneJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapSceneRecord(record)
	if err != nil {
		t.Fatalf("MapSceneRecord() error = %v", err)
	}

	// Verify location consent is enforced
	if result.AllowPrecise {
		t.Error("AllowPrecise should be false")
	}
	if result.PrecisePoint != nil {
		t.Error("PrecisePoint should be cleared when AllowPrecise is false")
	}
	if result.CoarseGeohash == "" {
		t.Error("CoarseGeohash should still be set")
	}
}

func TestMapSceneRecord_MissingName(t *testing.T) {
	sceneJSON := `{"description":"No name"}`

	record := &FilterResult{
		DID:        "did:plc:scene789",
		Collection: CollectionScene,
		RKey:       "scene3",
		Record:     json.RawMessage(sceneJSON),
		Valid:      true,
		Matched:    true,
	}

	_, err := MapSceneRecord(record)
	if err == nil {
		t.Fatal("Expected error for missing name")
	}
}

func TestMapSceneRecord_NilRecord(t *testing.T) {
	_, err := MapSceneRecord(nil)
	if err != ErrMissingRequiredField {
		t.Errorf("Expected ErrMissingRequiredField, got %v", err)
	}
}

func TestMapSceneRecord_InvalidJSON(t *testing.T) {
	record := &FilterResult{
		DID:        "did:plc:scene999",
		Collection: CollectionScene,
		RKey:       "scene4",
		Record:     json.RawMessage(`{invalid json`),
		Valid:      true,
		Matched:    true,
	}

	_, err := MapSceneRecord(record)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestMapEventRecord_ValidMinimal(t *testing.T) {
	eventJSON := `{
		"name": "Friday Night",
		"sceneId": "scene123",
		"startsAt": "2024-06-15T20:00:00Z"
	}`

	record := &FilterResult{
		DID:        "did:plc:event456",
		Collection: CollectionEvent,
		RKey:       "event1",
		Record:     json.RawMessage(eventJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapEventRecord(record)
	if err != nil {
		t.Fatalf("MapEventRecord() error = %v", err)
	}

	// Verify required fields
	if result.Title != "Friday Night" {
		t.Errorf("Title = %q, want %q", result.Title, "Friday Night")
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2024-06-15T20:00:00Z")
	if !result.StartsAt.Equal(expectedTime) {
		t.Errorf("StartsAt = %v, want %v", result.StartsAt, expectedTime)
	}
	if result.RecordDID == nil || *result.RecordDID != "did:plc:event456" {
		t.Error("RecordDID not set correctly")
	}

	// Verify defaults
	if result.Status != "scheduled" {
		t.Errorf("Status = %q, want scheduled", result.Status)
	}
}

func TestMapEventRecord_ValidComplete(t *testing.T) {
	eventJSON := `{
		"name": "Saturday Rave",
		"sceneId": "scene456",
		"description": "All night long",
		"startsAt": "2024-06-15T22:00:00Z",
		"endsAt": "2024-06-16T06:00:00Z",
		"location": {
			"lat": 52.52,
			"lng": 13.405,
			"allowPrecise": true
		},
		"tags": ["rave", "techno"],
		"status": "live"
	}`

	record := &FilterResult{
		DID:        "did:plc:event789",
		Collection: CollectionEvent,
		RKey:       "event2",
		Record:     json.RawMessage(eventJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapEventRecord(record)
	if err != nil {
		t.Fatalf("MapEventRecord() error = %v", err)
	}

	// Verify all fields
	if result.Title != "Saturday Rave" {
		t.Errorf("Title = %q", result.Title)
	}
	if result.Description != "All night long" {
		t.Errorf("Description = %q", result.Description)
	}
	if result.EndsAt == nil {
		t.Fatal("EndsAt should not be nil")
	}
	if len(result.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(result.Tags))
	}
	if result.Status != "live" {
		t.Errorf("Status = %q, want live", result.Status)
	}
	if !result.AllowPrecise {
		t.Error("AllowPrecise should be true")
	}
}

func TestMapEventRecord_MissingRequired(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"missing name", `{"sceneId":"s1","startsAt":"2024-01-01T00:00:00Z"}`},
		{"missing sceneId", `{"name":"Event","startsAt":"2024-01-01T00:00:00Z"}`},
		{"missing startsAt", `{"name":"Event","sceneId":"s1"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &FilterResult{
				DID:        "did:plc:test",
				Collection: CollectionEvent,
				RKey:       "event",
				Record:     json.RawMessage(tt.json),
				Valid:      true,
				Matched:    true,
			}

			_, err := MapEventRecord(record)
			if err == nil {
				t.Fatalf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestMapEventRecord_InvalidTimestamp(t *testing.T) {
	eventJSON := `{
		"name": "Bad Event",
		"sceneId": "scene1",
		"startsAt": "not-a-timestamp"
	}`

	record := &FilterResult{
		DID:        "did:plc:event",
		Collection: CollectionEvent,
		RKey:       "event",
		Record:     json.RawMessage(eventJSON),
		Valid:      true,
		Matched:    true,
	}

	_, err := MapEventRecord(record)
	if err == nil {
		t.Fatal("Expected error for invalid timestamp")
	}
}

func TestMapPostRecord_ValidMinimal(t *testing.T) {
	postJSON := `{
		"text": "Great night!",
		"sceneId": "scene123"
	}`

	record := &FilterResult{
		DID:        "did:plc:user789",
		Collection: CollectionPost,
		RKey:       "post1",
		Record:     json.RawMessage(postJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapPostRecord(record)
	if err != nil {
		t.Fatalf("MapPostRecord() error = %v", err)
	}

	// Verify required fields
	if result.Text != "Great night!" {
		t.Errorf("Text = %q", result.Text)
	}
	if result.AuthorDID != "did:plc:user789" {
		t.Errorf("AuthorDID = %q", result.AuthorDID)
	}
	if result.RecordDID == nil || *result.RecordDID != "did:plc:user789" {
		t.Error("RecordDID not set correctly")
	}
	if result.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if result.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestMapPostRecord_ValidComplete(t *testing.T) {
	postJSON := `{
		"text": "Amazing show!",
		"sceneId": "scene456",
		"eventId": "event789",
		"attachments": [
			{
				"url": "https://example.com/photo.jpg",
				"type": "image/jpeg",
				"sizeBytes": 12345,
				"width": 1920,
				"height": 1080
			}
		],
		"labels": ["featured", "photo"]
	}`

	record := &FilterResult{
		DID:        "did:plc:user999",
		Collection: CollectionPost,
		RKey:       "post2",
		Record:     json.RawMessage(postJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapPostRecord(record)
	if err != nil {
		t.Fatalf("MapPostRecord() error = %v", err)
	}

	// Verify all fields
	if result.Text != "Amazing show!" {
		t.Errorf("Text = %q", result.Text)
	}
	if len(result.Attachments) != 1 {
		t.Fatalf("Attachments length = %d, want 1", len(result.Attachments))
	}
	att := result.Attachments[0]
	if att.URL != "https://example.com/photo.jpg" {
		t.Errorf("Attachment URL = %q", att.URL)
	}
	if att.Type != "image/jpeg" {
		t.Errorf("Attachment Type = %q", att.Type)
	}
	if att.SizeBytes != 12345 {
		t.Errorf("Attachment SizeBytes = %d", att.SizeBytes)
	}
	if att.Width == nil || *att.Width != 1920 {
		t.Error("Attachment Width not set correctly")
	}
	if len(result.Labels) != 2 {
		t.Errorf("Labels length = %d, want 2", len(result.Labels))
	}
}

func TestMapPostRecord_MissingText(t *testing.T) {
	postJSON := `{"sceneId":"scene1"}`

	record := &FilterResult{
		DID:        "did:plc:user",
		Collection: CollectionPost,
		RKey:       "post",
		Record:     json.RawMessage(postJSON),
		Valid:      true,
		Matched:    true,
	}

	_, err := MapPostRecord(record)
	if err == nil {
		t.Fatal("Expected error for missing text")
	}
}

func TestMapPostRecord_MissingSceneAndEvent(t *testing.T) {
	postJSON := `{"text":"Hello"}`

	record := &FilterResult{
		DID:        "did:plc:user",
		Collection: CollectionPost,
		RKey:       "post",
		Record:     json.RawMessage(postJSON),
		Valid:      true,
		Matched:    true,
	}

	_, err := MapPostRecord(record)
	if err == nil {
		t.Fatal("Expected error for missing sceneId and eventId")
	}
}

func TestMapAllianceRecord_ValidMinimal(t *testing.T) {
	allianceJSON := `{
		"fromSceneId": "scene123",
		"toSceneId": "scene456"
	}`

	record := &FilterResult{
		DID:        "did:plc:scene123",
		Collection: "app.subcult.alliance",
		RKey:       "alliance1",
		Record:     json.RawMessage(allianceJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapAllianceRecord(record)
	if err != nil {
		t.Fatalf("MapAllianceRecord() error = %v", err)
	}

	// Verify required fields
	if result.RecordDID == nil || *result.RecordDID != "did:plc:scene123" {
		t.Error("RecordDID not set correctly")
	}
	if result.RecordRKey == nil || *result.RecordRKey != "alliance1" {
		t.Error("RecordRKey not set correctly")
	}

	// Verify defaults
	if result.Weight != 1.0 {
		t.Errorf("Weight = %f, want 1.0", result.Weight)
	}
	if result.Status != "active" {
		t.Errorf("Status = %q, want active", result.Status)
	}
	if result.Since.IsZero() {
		t.Error("Since should be set to current time")
	}
}

func TestMapAllianceRecord_ValidComplete(t *testing.T) {
	allianceJSON := `{
		"fromSceneId": "scene123",
		"toSceneId": "scene456",
		"weight": 2.5,
		"status": "pending",
		"reason": "Collaboration request",
		"since": "2024-01-01T00:00:00Z"
	}`

	record := &FilterResult{
		DID:        "did:plc:scene123",
		Collection: "app.subcult.alliance",
		RKey:       "alliance2",
		Record:     json.RawMessage(allianceJSON),
		Valid:      true,
		Matched:    true,
	}

	result, err := MapAllianceRecord(record)
	if err != nil {
		t.Fatalf("MapAllianceRecord() error = %v", err)
	}

	// Verify all fields
	if result.Weight != 2.5 {
		t.Errorf("Weight = %f, want 2.5", result.Weight)
	}
	if result.Status != "pending" {
		t.Errorf("Status = %q, want pending", result.Status)
	}
	if result.Reason == nil || *result.Reason != "Collaboration request" {
		t.Error("Reason not set correctly")
	}
	expectedSince, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	if !result.Since.Equal(expectedSince) {
		t.Errorf("Since = %v, want %v", result.Since, expectedSince)
	}
}

func TestMapAllianceRecord_MissingRequired(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"missing fromSceneId", `{"toSceneId":"scene2"}`},
		{"missing toSceneId", `{"fromSceneId":"scene1"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &FilterResult{
				DID:        "did:plc:test",
				Collection: "app.subcult.alliance",
				RKey:       "alliance",
				Record:     json.RawMessage(tt.json),
				Valid:      true,
				Matched:    true,
			}

			_, err := MapAllianceRecord(record)
			if err == nil {
				t.Fatalf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestMapAllianceRecord_InvalidTimestamp(t *testing.T) {
	allianceJSON := `{
		"fromSceneId": "scene1",
		"toSceneId": "scene2",
		"since": "invalid-timestamp"
	}`

	record := &FilterResult{
		DID:        "did:plc:scene",
		Collection: "app.subcult.alliance",
		RKey:       "alliance",
		Record:     json.RawMessage(allianceJSON),
		Valid:      true,
		Matched:    true,
	}

	_, err := MapAllianceRecord(record)
	if err == nil {
		t.Fatal("Expected error for invalid timestamp")
	}
}
