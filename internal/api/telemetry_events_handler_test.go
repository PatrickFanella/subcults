package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onnwee/subcults/internal/telemetry"
)

func TestTelemetryHandlers_PostEvents(t *testing.T) {
	store := telemetry.NewInMemoryStore()
	metrics := telemetry.NewMetrics()
	handler := NewTelemetryHandlers(store, metrics)

	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:   "valid event batch",
			method: http.MethodPost,
			body: TelemetryEventsRequest{
				Events: []telemetry.TelemetryEvent{
					{SessionID: "sess-1", Name: "page_view", Timestamp: 1000},
					{SessionID: "sess-1", Name: "click", Timestamp: 2000},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:   "single event",
			method: http.MethodPost,
			body: TelemetryEventsRequest{
				Events: []telemetry.TelemetryEvent{
					{SessionID: "sess-1", Name: "search", Timestamp: 3000},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "wrong method",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "empty events array",
			method: http.MethodPost,
			body: TelemetryEventsRequest{
				Events: []telemetry.TelemetryEvent{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "events missing required fields are skipped",
			method: http.MethodPost,
			body: TelemetryEventsRequest{
				Events: []telemetry.TelemetryEvent{
					{Name: "", SessionID: "sess-1", Timestamp: 1000},   // missing name
					{Name: "ok", SessionID: "", Timestamp: 1000},       // missing session
					{Name: "ok", SessionID: "sess-1", Timestamp: 0},    // missing timestamp
				},
			},
			expectedStatus: http.StatusBadRequest, // all invalid → no valid events
		},
		{
			name:   "mix of valid and invalid events",
			method: http.MethodPost,
			body: TelemetryEventsRequest{
				Events: []telemetry.TelemetryEvent{
					{Name: "valid", SessionID: "sess-1", Timestamp: 1000},
					{Name: "", SessionID: "sess-1", Timestamp: 1000}, // invalid
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					reqBody = []byte(v)
				default:
					reqBody, err = json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("failed to marshal: %v", err)
					}
				}
			}

			req := httptest.NewRequest(tt.method, "/api/telemetry", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.PostEvents(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectSuccess {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if success, ok := resp["success"].(bool); !ok || !success {
					t.Errorf("expected success=true, got %v", resp["success"])
				}
			}
		})
	}
}

func TestTelemetryHandlers_PostEvents_MaxBatchSize(t *testing.T) {
	store := telemetry.NewInMemoryStore()
	metrics := telemetry.NewMetrics()
	handler := NewTelemetryHandlers(store, metrics)

	// Create 21 events (exceeds maxTelemetryBatchSize of 20)
	events := make([]telemetry.TelemetryEvent, 21)
	for i := range events {
		events[i] = telemetry.TelemetryEvent{
			SessionID: "sess-1",
			Name:      "event",
			Timestamp: int64(i + 1),
		}
	}

	reqBody, _ := json.Marshal(TelemetryEventsRequest{Events: events})
	req := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostEvents(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oversized batch, got %d", w.Code)
	}
}

func TestTelemetryHandlers_PostEvents_PersistsToStore(t *testing.T) {
	store := telemetry.NewInMemoryStore()
	metrics := telemetry.NewMetrics()
	handler := NewTelemetryHandlers(store, metrics)

	reqBody, _ := json.Marshal(TelemetryEventsRequest{
		Events: []telemetry.TelemetryEvent{
			{SessionID: "sess-1", Name: "page_view", Timestamp: 1000},
			{SessionID: "sess-1", Name: "click", Timestamp: 2000},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	events := store.GetEvents()
	if len(events) != 2 {
		t.Errorf("expected 2 persisted events, got %d", len(events))
	}
}
