package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onnwee/subcults/internal/telemetry"
)

func TestErrorLoggerHandlers_HandleClientError(t *testing.T) {
	store := telemetry.NewInMemoryStore()
	metrics := telemetry.NewMetrics()
	handler := NewErrorLoggerHandlers(store, metrics)

	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		expectErrorID  bool
	}{
		{
			name:   "valid error report",
			method: http.MethodPost,
			body: telemetry.ClientErrorLog{
				SessionID:    "sess-1",
				ErrorType:    "TypeError",
				ErrorMessage: "Cannot read property 'x' of undefined",
				ErrorStack:   "at func (file.js:10)",
				URL:          "/map",
				UserAgent:    "Mozilla/5.0",
				OccurredAt:   1234567890000,
			},
			expectedStatus: http.StatusOK,
			expectErrorID:  true,
		},
		{
			name:   "valid error with replay events",
			method: http.MethodPost,
			body: telemetry.ClientErrorLog{
				SessionID:    "sess-2",
				ErrorType:    "Error",
				ErrorMessage: "Network error",
				OccurredAt:   1234567890000,
				ReplayEvents: []telemetry.ReplayEvent{
					{EventType: "click", EventTimestamp: 1234567889000},
					{EventType: "navigation", EventTimestamp: 1234567889500},
				},
			},
			expectedStatus: http.StatusOK,
			expectErrorID:  true,
		},
		{
			name:           "wrong method GET",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           "not valid json",
			expectedStatus: http.StatusOK, // always 200 for fire-and-forget
		},
		{
			name:   "missing required fields",
			method: http.MethodPost,
			body: telemetry.ClientErrorLog{
				SessionID: "",
			},
			expectedStatus: http.StatusOK, // always 200
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

			req := httptest.NewRequest(tt.method, "/api/log/client-error", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.HandleClientError(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectErrorID {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp["errorId"] == nil || resp["errorId"] == "" {
					t.Error("expected non-empty errorId in response")
				}
				if success, ok := resp["success"].(bool); !ok || !success {
					t.Errorf("expected success=true, got %v", resp["success"])
				}
			}
		})
	}
}

func TestErrorLoggerHandlers_Deduplication(t *testing.T) {
	store := telemetry.NewInMemoryStore()
	metrics := telemetry.NewMetrics()
	handler := NewErrorLoggerHandlers(store, metrics)

	errPayload := telemetry.ClientErrorLog{
		SessionID:    "sess-1",
		ErrorType:    "TypeError",
		ErrorMessage: "same error message",
		OccurredAt:   1234567890000,
	}

	// First request — should persist
	body1, _ := json.Marshal(errPayload)
	req1 := httptest.NewRequest(http.MethodPost, "/api/log/client-error", bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleClientError(w1, req1)

	var resp1 map[string]interface{}
	_ = json.NewDecoder(w1.Body).Decode(&resp1)
	if resp1["errorId"] == nil || resp1["errorId"] == "" {
		t.Error("first request should return errorId")
	}

	// Second request with same session+message — should be deduped
	body2, _ := json.Marshal(errPayload)
	req2 := httptest.NewRequest(http.MethodPost, "/api/log/client-error", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleClientError(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("deduped request should return 200, got %d", w2.Code)
	}

	var resp2 map[string]interface{}
	_ = json.NewDecoder(w2.Body).Decode(&resp2)
	// Deduped: no errorId, just success
	if resp2["errorId"] != nil {
		t.Error("deduped request should not return errorId")
	}

	// Verify only one error in store
	logs := store.GetErrorLogs()
	if len(logs) != 1 {
		t.Errorf("expected 1 error log (deduped), got %d", len(logs))
	}
}

func TestErrorLoggerHandlers_ReplayEventsPersisted(t *testing.T) {
	store := telemetry.NewInMemoryStore()
	metrics := telemetry.NewMetrics()
	handler := NewErrorLoggerHandlers(store, metrics)

	errPayload := telemetry.ClientErrorLog{
		SessionID:    "sess-replay",
		ErrorType:    "Error",
		ErrorMessage: "test with replays",
		OccurredAt:   1234567890000,
		ReplayEvents: []telemetry.ReplayEvent{
			{EventType: "click", EventTimestamp: 1234567889000},
			{EventType: "scroll", EventTimestamp: 1234567889500},
		},
	}

	body, _ := json.Marshal(errPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/log/client-error", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleClientError(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	replays := store.GetReplayEvents()
	if len(replays) != 2 {
		t.Errorf("expected 2 replay events, got %d", len(replays))
	}
	if replays[0].EventType != "click" {
		t.Errorf("expected first replay type 'click', got %q", replays[0].EventType)
	}
}
