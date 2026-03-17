package indexer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestIntegration_EndToEndRecordProcessing tests the complete flow from WebSocket
// message reception to record filtering and validation.
func TestIntegration_EndToEndRecordProcessing(t *testing.T) {
	// Create filter with metrics
	filterMetrics := NewFilterMetrics()
	filter := NewRecordFilter(filterMetrics)

	// Create a mock Jetstream server that sends valid AT Protocol messages
	var messagesSent int32
	var connectionsMade int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only allow one connection to avoid duplicate messages
		if atomic.AddInt32(&connectionsMade, 1) > 1 {
			http.Error(w, "Only one connection allowed", http.StatusTooManyRequests)
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send a sequence of AT Protocol messages
		messages := []struct {
			name    string
			message JetstreamMessage
		}{
			{
				name: "valid scene create",
				message: JetstreamMessage{
					DID:    "did:plc:test123",
					TimeUS: time.Now().UnixMicro(),
					Kind:   "commit",
					Commit: &AtProtoCommit{
						DID:        "did:plc:test123",
						Rev:        "rev1",
						Operation:  "create",
						Collection: CollectionScene,
						RKey:       "scene1",
						Record:     mustEncodeCBOR(t, map[string]interface{}{"name": "Test Scene"}),
					},
				},
			},
			{
				name: "valid event create",
				message: JetstreamMessage{
					DID:    "did:plc:test456",
					TimeUS: time.Now().UnixMicro(),
					Kind:   "commit",
					Commit: &AtProtoCommit{
						DID:        "did:plc:test456",
						Rev:        "rev1",
						Operation:  "create",
						Collection: CollectionEvent,
						RKey:       "event1",
						Record:     mustEncodeCBOR(t, map[string]interface{}{"name": "Test Event", "sceneId": "scene1"}),
					},
				},
			},
			{
				name: "non-matching collection (should be filtered)",
				message: JetstreamMessage{
					DID:    "did:plc:test789",
					TimeUS: time.Now().UnixMicro(),
					Kind:   "commit",
					Commit: &AtProtoCommit{
						DID:        "did:plc:test789",
						Rev:        "rev1",
						Operation:  "create",
						Collection: "app.bsky.feed.post",
						RKey:       "post1",
						Record:     mustEncodeCBOR(t, map[string]interface{}{"text": "Hello"}),
					},
				},
			},
			{
				name: "delete operation",
				message: JetstreamMessage{
					DID:    "did:plc:test123",
					TimeUS: time.Now().UnixMicro(),
					Kind:   "commit",
					Commit: &AtProtoCommit{
						DID:        "did:plc:test123",
						Rev:        "rev2",
						Operation:  "delete",
						Collection: CollectionScene,
						RKey:       "scene1",
					},
				},
			},
		}

		for _, msg := range messages {
			cborData, err := EncodeCBOR(msg.message)
			if err != nil {
				t.Logf("Failed to encode message %s: %v", msg.name, err)
				return
			}

			if err := conn.WriteMessage(websocket.BinaryMessage, cborData); err != nil {
				return
			}
			atomic.AddInt32(&messagesSent, 1)
			time.Sleep(10 * time.Millisecond)
		}

		// Keep connection alive briefly
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	// Create indexer client
	config := Config{
		URL:          "ws" + server.URL[4:], // Replace http with ws
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0,
	}

	metrics := NewMetrics()
	var processedRecords []FilterResult
	var mu sync.Mutex

	handler := func(msgType int, payload []byte) error {
		// Filter the record
		result := filter.FilterCBOR(payload)

		mu.Lock()
		processedRecords = append(processedRecords, result)
		mu.Unlock()

		metrics.IncMessagesProcessed()

		if result.Valid {
			metrics.IncUpserts()
		} else {
			metrics.IncMessagesError()
		}

		return nil
	}

	client, err := NewClientWithMetrics(config, handler, newTestLogger(), metrics)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Run client
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = client.Run(ctx)
		close(done)
	}()

	// Wait for all messages to be processed
	time.Sleep(800 * time.Millisecond)
	cancel()
	<-done

	// Verify results
	mu.Lock()
	defer mu.Unlock()

	if len(processedRecords) == 0 {
		t.Fatal("No records were processed")
	}

	// Check that we received all messages
	expectedCount := atomic.LoadInt32(&messagesSent)
	if int32(len(processedRecords)) != expectedCount {
		t.Errorf("Expected %d processed records, got %d", expectedCount, len(processedRecords))
	}

	// Verify specific records
	var validScenes, validEvents, filtered, deletes int
	for _, result := range processedRecords {
		t.Logf("Record: DID=%s, Collection=%s, Operation=%s, Valid=%v, Matched=%v",
			result.DID, result.Collection, result.Operation, result.Valid, result.Matched)

		if result.Collection == CollectionScene && result.Valid && result.Operation != "delete" {
			validScenes++
		} else if result.Collection == CollectionEvent && result.Valid {
			validEvents++
		} else if !result.Matched {
			filtered++
		}

		if result.Operation == "delete" && result.Matched {
			deletes++
		}
	}

	if validScenes < 1 {
		t.Errorf("Expected at least 1 valid scene, got %d", validScenes)
	}
	if validEvents < 1 {
		t.Errorf("Expected at least 1 valid event, got %d", validEvents)
	}
	if filtered < 1 {
		t.Errorf("Expected at least 1 filtered (non-matching) record, got %d", filtered)
	}
	if deletes < 1 {
		t.Errorf("Expected at least 1 delete operation, got %d", deletes)
	}

	// Verify metrics
	if filterMetrics.Processed() != int64(expectedCount) {
		t.Errorf("FilterMetrics.Processed() = %d, want %d", filterMetrics.Processed(), expectedCount)
	}

	// Verify client metrics
	processedCount := getCounterValue(metrics.messagesProcessed)
	if processedCount != float64(expectedCount) {
		t.Errorf("Client metrics processed = %f, want %f", processedCount, float64(expectedCount))
	}
}

// TestIntegration_RecoveryFromErrors tests that the indexer continues processing
// after encountering errors.
func TestIntegration_RecoveryFromErrors(t *testing.T) {
	filterMetrics := NewFilterMetrics()
	filter := NewRecordFilter(filterMetrics)

	var connectionsMade int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only allow one connection to avoid duplicate messages
		if atomic.AddInt32(&connectionsMade, 1) > 1 {
			http.Error(w, "Only one connection allowed", http.StatusTooManyRequests)
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send mix of valid and invalid messages
		messages := [][]byte{
			// Valid message
			mustEncodeCBORMessage(t, JetstreamMessage{
				DID:    "did:plc:valid1",
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        "did:plc:valid1",
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       "scene1",
					Record:     mustEncodeCBOR(t, map[string]interface{}{"name": "Valid Scene"}),
				},
			}),
			// Invalid CBOR
			[]byte{0xff, 0xff, 0xff},
			// Valid message after error
			mustEncodeCBORMessage(t, JetstreamMessage{
				DID:    "did:plc:valid2",
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        "did:plc:valid2",
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       "scene2",
					Record:     mustEncodeCBOR(t, map[string]interface{}{"name": "Another Scene"}),
				},
			}),
			// Missing required field
			mustEncodeCBORMessage(t, JetstreamMessage{
				DID:    "did:plc:invalid1",
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        "did:plc:invalid1",
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       "scene3",
					Record:     mustEncodeCBOR(t, map[string]interface{}{"description": "Missing name"}),
				},
			}),
		}

		for _, msg := range messages {
			if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}

		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	config := Config{
		URL:          "ws" + server.URL[4:],
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0,
	}

	var validCount, errorCount int32
	handler := func(msgType int, payload []byte) error {
		result := filter.FilterCBOR(payload)
		if result.Valid {
			atomic.AddInt32(&validCount, 1)
		} else {
			atomic.AddInt32(&errorCount, 1)
		}
		return nil
	}

	client, err := NewClient(config, handler, newTestLogger())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		_ = client.Run(ctx)
	}()

	time.Sleep(800 * time.Millisecond)
	cancel()

	// Should have processed 2 valid scenes despite errors
	if atomic.LoadInt32(&validCount) != 2 {
		t.Errorf("Expected 2 valid records, got %d", atomic.LoadInt32(&validCount))
	}

	// Should have encountered 2 errors (invalid CBOR + missing field)
	if atomic.LoadInt32(&errorCount) < 2 {
		t.Errorf("Expected at least 2 error records, got %d", atomic.LoadInt32(&errorCount))
	}
}

// TestIntegration_GracefulShutdown tests that the indexer shuts down cleanly
// and drains pending messages.
func TestIntegration_GracefulShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send messages continuously
		for i := 0; i < 100; i++ {
			msg := JetstreamMessage{
				DID:    "did:plc:test",
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        "did:plc:test",
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       fmt.Sprintf("scene%d", i),
					Record:     mustEncodeCBOR(t, map[string]interface{}{"name": "Scene"}),
				},
			}
			cborData, _ := EncodeCBOR(msg)
			if err := conn.WriteMessage(websocket.BinaryMessage, cborData); err != nil {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer server.Close()

	config := Config{
		URL:          "ws" + server.URL[4:],
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0,
	}

	var processedCount int32
	var processingDone int32
	handler := func(msgType int, payload []byte) error {
		// Simulate slow processing
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&processedCount, 1)
		return nil
	}

	client, err := NewClient(config, handler, newTestLogger())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		_ = client.Run(ctx)
		atomic.StoreInt32(&processingDone, 1)
		close(done)
	}()

	// Let some messages accumulate
	time.Sleep(200 * time.Millisecond)

	// Cancel and verify graceful shutdown
	cancel()

	// Wait for shutdown with timeout
	select {
	case <-done:
		// Success - client shut down
	case <-time.After(10 * time.Second):
		t.Fatal("Client did not shut down within timeout")
	}

	// Verify processing completed flag
	if atomic.LoadInt32(&processingDone) != 1 {
		t.Error("Processing done flag not set")
	}

	// Should have processed at least some messages
	processed := atomic.LoadInt32(&processedCount)
	if processed == 0 {
		t.Error("No messages were processed")
	}

	t.Logf("Processed %d messages before shutdown", processed)
}

// Helper functions

func mustEncodeCBOR(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := EncodeCBOR(v)
	if err != nil {
		t.Fatalf("EncodeCBOR() error = %v", err)
	}
	return data
}

func mustEncodeCBORMessage(t *testing.T, msg JetstreamMessage) []byte {
	t.Helper()
	data, err := EncodeCBOR(msg)
	if err != nil {
		t.Fatalf("EncodeCBOR() error = %v", err)
	}
	return data
}
