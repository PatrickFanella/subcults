package audit

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExportLogs_CSV_ByUser(t *testing.T) {
	repo := NewInMemoryRepository()

	// Add test data
	now := time.Now().UTC()
	entries := []LogEntry{
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "scene_create", Outcome: OutcomeSuccess},
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "scene_update", Outcome: OutcomeSuccess},
		{UserDID: "user2", EntityType: "event", EntityID: "event-1", Action: "event_create", Outcome: OutcomeSuccess},
	}

	for _, entry := range entries {
		_, err := repo.LogAccess(entry)
		if err != nil {
			t.Fatalf("LogAccess() error = %v", err)
		}
	}

	// Export logs for user1
	opts := ExportOptions{
		Format:  ExportFormatCSV,
		UserDID: "user1",
		From:    now.Add(-1 * time.Hour),
		To:      now.Add(1 * time.Hour),
	}

	data, err := ExportLogs(repo, opts)
	if err != nil {
		t.Fatalf("ExportLogs() error = %v", err)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + 2 data rows
	if len(records) != 3 {
		t.Errorf("Expected 3 CSV rows (header + 2 data), got %d", len(records))
	}

	// Verify header
	expectedHeader := []string{"ID", "Timestamp (UTC)", "User DID", "Entity Type", "Entity ID", "Action", "Outcome", "Request ID", "IP Address", "User Agent", "Previous Hash"}
	header := records[0]
	if len(header) != len(expectedHeader) {
		t.Errorf("CSV header has %d columns, want %d", len(header), len(expectedHeader))
	}

	// Verify data rows contain user1
	for i := 1; i < len(records); i++ {
		if records[i][2] != "user1" {
			t.Errorf("Row %d User DID = %q, want user1", i, records[i][2])
		}
	}
}

func TestExportLogs_JSON_ByUser(t *testing.T) {
	repo := NewInMemoryRepository()

	// Add test data
	now := time.Now().UTC()
	entries := []LogEntry{
		{UserDID: "user1", EntityType: "scene", EntityID: "scene-1", Action: "scene_create", Outcome: OutcomeSuccess},
		{UserDID: "user1", EntityType: "payment", EntityID: "pay-1", Action: "payment_create", Outcome: OutcomeSuccess},
		{UserDID: "user2", EntityType: "event", EntityID: "event-1", Action: "event_create", Outcome: OutcomeSuccess},
	}

	for _, entry := range entries {
		_, err := repo.LogAccess(entry)
		if err != nil {
			t.Fatalf("LogAccess() error = %v", err)
		}
	}

	// Export logs for user1
	opts := ExportOptions{
		Format:  ExportFormatJSON,
		UserDID: "user1",
		From:    now.Add(-1 * time.Hour),
		To:      now.Add(1 * time.Hour),
	}

	data, err := ExportLogs(repo, opts)
	if err != nil {
		t.Fatalf("ExportLogs() error = %v", err)
	}

	// Parse JSON
	var logs []map[string]interface{}
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should have 2 logs for user1
	if len(logs) != 2 {
		t.Errorf("Expected 2 JSON logs, got %d", len(logs))
	}

	// Verify all logs are for user1
	for i, log := range logs {
		userDID, ok := log["user_did"].(string)
		if !ok {
			t.Fatalf("Log %d missing user_did field", i)
		}
		if userDID != "user1" {
			t.Errorf("Log %d user_did = %q, want user1", i, userDID)
		}

		// Verify required fields exist
		requiredFields := []string{"id", "timestamp", "entity_type", "entity_id", "action", "outcome"}
		for _, field := range requiredFields {
			if _, ok := log[field]; !ok {
				t.Errorf("Log %d missing required field: %s", i, field)
			}
		}
	}
}

func TestExportLogs_TimeRangeFilter(t *testing.T) {
	repo := NewInMemoryRepository()

	// Add test data at different times
	now := time.Now().UTC()
	
	// Old entry (should be filtered out)
	entry1 := LogEntry{
		UserDID:    "user1",
		EntityType: "scene",
		EntityID:   "scene-1",
		Action:     "scene_create",
		Outcome:    OutcomeSuccess,
	}
	repo.LogAccess(entry1)
	time.Sleep(10 * time.Millisecond)

	fromTime := time.Now().UTC()
	time.Sleep(10 * time.Millisecond)

	// Recent entries (should be included)
	entry2 := LogEntry{
		UserDID:    "user1",
		EntityType: "scene",
		EntityID:   "scene-1",
		Action:     "scene_update",
		Outcome:    OutcomeSuccess,
	}
	repo.LogAccess(entry2)

	entry3 := LogEntry{
		UserDID:    "user1",
		EntityType: "event",
		EntityID:   "event-1",
		Action:     "event_create",
		Outcome:    OutcomeSuccess,
	}
	repo.LogAccess(entry3)

	// Export with time range
	opts := ExportOptions{
		Format:  ExportFormatJSON,
		UserDID: "user1",
		From:    fromTime,
		To:      now.Add(1 * time.Hour),
	}

	data, err := ExportLogs(repo, opts)
	if err != nil {
		t.Fatalf("ExportLogs() error = %v", err)
	}

	var logs []map[string]interface{}
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should only have 2 recent logs
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs after time filter, got %d", len(logs))
	}
}

func TestExportLogs_InvalidFormat(t *testing.T) {
	repo := NewInMemoryRepository()

	opts := ExportOptions{
		Format:  "invalid",
		UserDID: "user1",
	}

	_, err := ExportLogs(repo, opts)
	if err == nil {
		t.Error("ExportLogs() with invalid format should return error")
	}
}

func TestExportLogs_NoUserDIDFilter(t *testing.T) {
	repo := NewInMemoryRepository()

	opts := ExportOptions{
		Format: ExportFormatJSON,
		// No UserDID filter
	}

	_, err := ExportLogs(repo, opts)
	if err == nil {
		t.Error("ExportLogs() without UserDID filter should return error (not yet implemented)")
	}
}

func TestExportLogs_EmptyResults(t *testing.T) {
	repo := NewInMemoryRepository()

	opts := ExportOptions{
		Format:  ExportFormatJSON,
		UserDID: "nonexistent",
	}

	data, err := ExportLogs(repo, opts)
	if err != nil {
		t.Fatalf("ExportLogs() error = %v", err)
	}

	var logs []map[string]interface{}
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs for nonexistent user, got %d", len(logs))
	}
}

func TestExportLogs_WithLimit(t *testing.T) {
	repo := NewInMemoryRepository()

	// Add 5 entries
	for i := 0; i < 5; i++ {
		entry := LogEntry{
			UserDID:    "user1",
			EntityType: "scene",
			EntityID:   "scene-1",
			Action:     "scene_update",
			Outcome:    OutcomeSuccess,
		}
		repo.LogAccess(entry)
	}

	// Export with limit
	opts := ExportOptions{
		Format:  ExportFormatJSON,
		UserDID: "user1",
		Limit:   3,
	}

	data, err := ExportLogs(repo, opts)
	if err != nil {
		t.Fatalf("ExportLogs() error = %v", err)
	}

	var logs []map[string]interface{}
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs with limit, got %d", len(logs))
	}
}

func TestExportToCSV_SpecialCharacters(t *testing.T) {
	repo := NewInMemoryRepository()

	// Entry with special characters that need CSV escaping
	entry := LogEntry{
		UserDID:    "user1",
		EntityType: "scene",
		EntityID:   "scene-1",
		Action:     "scene_create",
		Outcome:    OutcomeSuccess,
		UserAgent:  "Mozilla/5.0 (Test, with \"quotes\" and commas)",
	}
	_, err := repo.LogAccess(entry)
	if err != nil {
		t.Fatalf("LogAccess() error = %v", err)
	}

	opts := ExportOptions{
		Format:  ExportFormatCSV,
		UserDID: "user1",
	}

	data, err := ExportLogs(repo, opts)
	if err != nil {
		t.Fatalf("ExportLogs() error = %v", err)
	}

	// Should be valid CSV
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV with special characters: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 CSV rows, got %d", len(records))
	}
}
