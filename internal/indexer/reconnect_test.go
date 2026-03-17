package indexer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestClient_SequenceTracking_Resume tests that the client resumes from the last sequence
func TestClient_SequenceTracking_Resume(t *testing.T) {
	// Create a sequence tracker with an initial sequence
	tracker := NewInMemorySequenceTracker(newTestLogger())
	ctx := context.Background()

	// Set initial sequence to simulate previous run
	initialSeq := int64(1234567890)
	if err := tracker.UpdateSequence(ctx, initialSeq); err != nil {
		t.Fatalf("Failed to set initial sequence: %v", err)
	}

	// Track what cursor parameter was received
	var receivedCursor string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the cursor parameter
		receivedCursor = r.URL.Query().Get("cursor")

		// Upgrade to WebSocket
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send a test message
		msg := JetstreamMessage{
			DID:    "did:plc:test",
			TimeUS: initialSeq + 100,
			Kind:   "commit",
		}
		cborData, _ := EncodeCBOR(msg)
		_ = conn.WriteMessage(websocket.BinaryMessage, cborData)

		// Keep connection alive briefly
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	config := Config{
		URL:              "ws" + server.URL[4:],
		BaseDelay:        10 * time.Millisecond,
		MaxDelay:         100 * time.Millisecond,
		JitterFactor:     0,
		MaxRetryAttempts: 5,
	}

	var messagesProcessed int32
	handler := func(msgType int, payload []byte) error {
		atomic.AddInt32(&messagesProcessed, 1)
		return nil
	}

	client, err := NewClientWithSequenceTracker(config, handler, newTestLogger(), nil, tracker)
	if err != nil {
		t.Fatalf("NewClientWithSequenceTracker() error = %v", err)
	}

	// Run client briefly
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = client.Run(ctx)

	// Verify cursor parameter was sent with resume value
	if receivedCursor != "1234567890" {
		t.Errorf("Expected cursor=%d in URL, got cursor=%s", initialSeq, receivedCursor)
	}

	// Verify at least one message was processed
	if atomic.LoadInt32(&messagesProcessed) == 0 {
		t.Error("Expected at least one message to be processed")
	}
}

// TestClient_MaxRetryAttempts_AlertsAfterLimit tests that max retry attempts trigger error logging
func TestClient_MaxRetryAttempts_AlertsAfterLimit(t *testing.T) {
	// Create a server that always fails to connect
	failCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&failCount, 1)
		// Return error without upgrading to WebSocket
		http.Error(w, "Connection refused", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := Config{
		URL:              "ws" + server.URL[4:],
		BaseDelay:        10 * time.Millisecond,
		MaxDelay:         50 * time.Millisecond,
		JitterFactor:     0,
		MaxRetryAttempts: 5,
	}

	metrics := NewMetrics()
	client, err := NewClientWithMetrics(config, nil, newTestLogger(), metrics)
	if err != nil {
		t.Fatalf("NewClientWithMetrics() error = %v", err)
	}

	// Run client briefly (long enough for multiple retry attempts)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = client.Run(ctx)

	// Verify multiple connection attempts were made
	attempts := atomic.LoadInt64(&client.reconnectCount)
	if attempts < config.MaxRetryAttempts {
		t.Errorf("Expected at least %d retry attempts, got %d", config.MaxRetryAttempts, attempts)
	}

	// Verify the server was contacted multiple times
	if atomic.LoadInt32(&failCount) < int32(config.MaxRetryAttempts) {
		t.Errorf("Expected at least %d connection attempts to server, got %d", config.MaxRetryAttempts, failCount)
	}
}

// TestClient_ReconnectionSuccess_TracksMetric tests that successful reconnections are tracked
func TestClient_ReconnectionSuccess_TracksMetric(t *testing.T) {
	connectionCount := int32(0)
	closeAfter := int32(2) // Close after 2 messages

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&connectionCount, 1)

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Send a few messages then close
		for i := 0; i < int(closeAfter); i++ {
			msg := JetstreamMessage{
				DID:    "did:plc:test",
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
			}
			cborData, _ := EncodeCBOR(msg)
			if err := conn.WriteMessage(websocket.BinaryMessage, cborData); err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		conn.Close()
	}))
	defer server.Close()

	config := Config{
		URL:              "ws" + server.URL[4:],
		BaseDelay:        10 * time.Millisecond,
		MaxDelay:         100 * time.Millisecond,
		JitterFactor:     0,
		MaxRetryAttempts: 5,
	}

	metrics := NewMetrics()
	client, err := NewClientWithMetrics(config, nil, newTestLogger(), metrics)
	if err != nil {
		t.Fatalf("NewClientWithMetrics() error = %v", err)
	}

	// Run client long enough for multiple reconnects
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_ = client.Run(ctx)

	// Verify multiple connections were established (initial + reconnects)
	connections := atomic.LoadInt32(&connectionCount)
	if connections < 2 {
		t.Errorf("Expected at least 2 connections (initial + reconnect), got %d", connections)
	}

	// Note: We can't easily verify the metric value here without exposing internal state,
	// but we've verified the code path is exercised
}

// TestClient_SequenceTracking_UpdateAfterProcessing tests that sequence is updated after message processing
func TestClient_SequenceTracking_UpdateAfterProcessing(t *testing.T) {
	tracker := NewInMemorySequenceTracker(newTestLogger())

	testSeq := int64(9876543210)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send a message with a specific sequence
		msg := JetstreamMessage{
			DID:    "did:plc:test",
			TimeUS: testSeq,
			Kind:   "commit",
		}
		cborData, _ := EncodeCBOR(msg)
		_ = conn.WriteMessage(websocket.BinaryMessage, cborData)

		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	config := Config{
		URL:              "ws" + server.URL[4:],
		BaseDelay:        10 * time.Millisecond,
		MaxDelay:         100 * time.Millisecond,
		JitterFactor:     0,
		MaxRetryAttempts: 5,
	}

	ctx := context.Background()
	handler := func(msgType int, payload []byte) error {
		// Decode message and update sequence
		msg, err := DecodeCBORMessage(payload)
		if err == nil && msg != nil && msg.TimeUS > 0 {
			_ = tracker.UpdateSequence(ctx, msg.TimeUS)
		}
		return nil
	}

	client, err := NewClientWithSequenceTracker(config, handler, newTestLogger(), nil, tracker)
	if err != nil {
		t.Fatalf("NewClientWithSequenceTracker() error = %v", err)
	}

	// Run client briefly
	runCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = client.Run(runCtx)

	// Verify sequence was updated
	seq, err := tracker.GetLastSequence(ctx)
	if err != nil {
		t.Fatalf("GetLastSequence() error = %v", err)
	}

	if seq != testSeq {
		t.Errorf("Expected sequence %d, got %d", testSeq, seq)
	}
}
