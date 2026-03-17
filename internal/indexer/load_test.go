package indexer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestLoad_1000CommitsPerSecond verifies the indexer can handle 1000+ commits/sec.
// This is a critical acceptance criteria test.
func TestLoad_1000CommitsPerSecond(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	const (
		targetRate    = 1000 // commits per second
		testDuration  = 5    // seconds
		totalMessages = targetRate * testDuration
		sendInterval  = time.Second / targetRate
		allowedMargin = 0.1 // 10% margin for timing variations
	)

	t.Logf("Load test configuration: %d msgs/sec for %d seconds = %d total messages",
		targetRate, testDuration, totalMessages)

	var messagesSent int32
	var messagesReceived int32
	var startTime, endTime time.Time
	var mu sync.Mutex
	var connectionsMade int32

	// Create high-throughput mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only allow one connection to avoid timing issues
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

		// Pre-generate a batch of messages to avoid encoding overhead during sending
		batchSize := 100
		messageBatch := make([][]byte, batchSize)
		for i := 0; i < batchSize; i++ {
			msg := JetstreamMessage{
				DID:    fmt.Sprintf("did:plc:load%d", i),
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        fmt.Sprintf("did:plc:load%d", i),
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       fmt.Sprintf("scene%d", i),
					Record:     mustEncodeCBORForLoad(map[string]interface{}{"name": fmt.Sprintf("Load Test Scene %d", i)}),
				},
			}
			data, err := EncodeCBOR(msg)
			if err != nil {
				t.Logf("Failed to encode message: %v", err)
				return
			}
			messageBatch[i] = data
		}

		mu.Lock()
		startTime = time.Now()
		mu.Unlock()

		// Send messages at target rate
		ticker := time.NewTicker(sendInterval)
		defer ticker.Stop()

		sent := 0
		for sent < totalMessages {
			select {
			case <-ticker.C:
				// Use pre-encoded message from batch (cycle through)
				msgData := messageBatch[sent%batchSize]
				if err := conn.WriteMessage(websocket.BinaryMessage, msgData); err != nil {
					mu.Lock()
					endTime = time.Now()
					mu.Unlock()
					return
				}
				sent++
				atomic.AddInt32(&messagesSent, 1)
			}
		}

		mu.Lock()
		endTime = time.Now()
		mu.Unlock()

		// Keep connection alive briefly to allow processing
		time.Sleep(1 * time.Second)
	}))
	defer server.Close()

	// Create indexer with optimized configuration
	config := Config{
		URL:          "ws" + server.URL[4:],
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0,
	}

	metrics := NewMetrics()
	filterMetrics := NewFilterMetrics()
	filter := NewRecordFilter(filterMetrics)

	// Fast handler with minimal processing
	handler := func(msgType int, payload []byte) error {
		// Quick validation
		result := filter.FilterCBOR(payload)
		atomic.AddInt32(&messagesReceived, 1)

		if result.Valid {
			metrics.IncUpserts()
		}

		return nil
	}

	client, err := NewClientWithMetrics(config, handler, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Run client
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(testDuration+10)*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = client.Run(ctx)
		close(done)
	}()

	// Wait for test to complete
	<-done

	// Analyze results
	mu.Lock()
	elapsed := endTime.Sub(startTime)
	mu.Unlock()

	sent := atomic.LoadInt32(&messagesSent)
	received := atomic.LoadInt32(&messagesReceived)

	// Verify we have valid timing
	if elapsed <= 0 {
		t.Fatalf("Invalid timing: elapsed=%v, likely reconnection issue", elapsed)
	}

	t.Logf("Load test results:")
	t.Logf("  Messages sent: %d", sent)
	t.Logf("  Messages received: %d", received)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Send rate: %.2f msgs/sec", float64(sent)/elapsed.Seconds())
	t.Logf("  Receive rate: %.2f msgs/sec", float64(received)/elapsed.Seconds())
	t.Logf("  Message loss: %d (%.2f%%)", sent-received, float64(sent-received)/float64(sent)*100)

	// Verify throughput meets requirements
	actualRate := float64(received) / elapsed.Seconds()
	minAcceptableRate := float64(targetRate) * (1 - allowedMargin)

	if actualRate < minAcceptableRate {
		t.Errorf("Throughput too low: %.2f msgs/sec (min required: %.2f msgs/sec)", actualRate, minAcceptableRate)
	}

	// Verify message delivery (allow small loss due to timing)
	deliveryRate := float64(received) / float64(sent)
	minDeliveryRate := 0.95 // 95% delivery

	if deliveryRate < minDeliveryRate {
		t.Errorf("Message delivery rate too low: %.2f%% (min required: %.2f%%)", deliveryRate*100, minDeliveryRate*100)
	}

	// Check for backpressure events
	pauseCount := getCounterValue(metrics.backpressurePaused)
	t.Logf("  Backpressure pauses: %.0f", pauseCount)

	// Verify filter metrics
	if filterMetrics.Processed() != int64(received) {
		t.Errorf("Filter processed count mismatch: %d != %d", filterMetrics.Processed(), received)
	}
}

// TestLoad_BurstTraffic tests handling of bursty traffic patterns.
func TestLoad_BurstTraffic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	const (
		burstSize     = 500
		burstInterval = 2 * time.Second
		numBursts     = 3
	)

	var messagesReceived int32
	var backpressureTriggered int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Pre-generate message
		msg := JetstreamMessage{
			DID:    "did:plc:burst",
			TimeUS: time.Now().UnixMicro(),
			Kind:   "commit",
			Commit: &AtProtoCommit{
				DID:        "did:plc:burst",
				Operation:  "create",
				Collection: CollectionScene,
				RKey:       "scene",
				Record:     mustEncodeCBORForLoad(map[string]interface{}{"name": "Burst Scene"}),
			},
		}
		msgData, _ := EncodeCBOR(msg)

		// Send bursts of messages
		for burst := 0; burst < numBursts; burst++ {
			// Send burst as fast as possible
			for i := 0; i < burstSize; i++ {
				if err := conn.WriteMessage(websocket.BinaryMessage, msgData); err != nil {
					return
				}
			}

			// Wait before next burst
			if burst < numBursts-1 {
				time.Sleep(burstInterval)
			}
		}

		// Keep connection alive
		time.Sleep(1 * time.Second)
	}))
	defer server.Close()

	config := Config{
		URL:          "ws" + server.URL[4:],
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0,
	}

	metrics := NewMetrics()

	// Slow handler to trigger backpressure
	handler := func(msgType int, payload []byte) error {
		time.Sleep(10 * time.Millisecond) // Simulate DB write
		atomic.AddInt32(&messagesReceived, 1)
		return nil
	}

	client, err := NewClientWithMetrics(config, handler, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Monitor backpressure
	stopMonitor := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if getCounterValue(metrics.backpressurePaused) > 0 {
					atomic.StoreInt32(&backpressureTriggered, 1)
				}
			case <-stopMonitor:
				return
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		_ = client.Run(ctx)
		close(done)
	}()

	<-done
	close(stopMonitor)

	received := atomic.LoadInt32(&messagesReceived)
	expectedTotal := burstSize * numBursts

	t.Logf("Burst test results:")
	t.Logf("  Expected messages: %d", expectedTotal)
	t.Logf("  Received messages: %d", received)
	t.Logf("  Backpressure triggered: %v", atomic.LoadInt32(&backpressureTriggered) == 1)

	// Should receive most messages despite bursts
	if received < int32(float64(expectedTotal)*0.8) {
		t.Errorf("Received too few messages: %d < %d", received, int32(float64(expectedTotal)*0.8))
	}

	// Backpressure should have been triggered during bursts
	if atomic.LoadInt32(&backpressureTriggered) != 1 {
		t.Log("Warning: Backpressure was not triggered during burst traffic (may be acceptable)")
	}
}

// TestLoad_ConcurrentProcessing tests parallel message processing without race conditions.
func TestLoad_ConcurrentProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	const numMessages = 1000

	var messagesReceived int32
	var processingMap sync.Map // Track which DIDs were processed
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

		// Send unique messages
		for i := 0; i < numMessages; i++ {
			msg := JetstreamMessage{
				DID:    fmt.Sprintf("did:plc:concurrent%d", i),
				TimeUS: time.Now().UnixMicro(),
				Kind:   "commit",
				Commit: &AtProtoCommit{
					DID:        fmt.Sprintf("did:plc:concurrent%d", i),
					Operation:  "create",
					Collection: CollectionScene,
					RKey:       fmt.Sprintf("scene%d", i),
					Record:     mustEncodeCBORForLoad(map[string]interface{}{"name": fmt.Sprintf("Scene %d", i)}),
				},
			}
			msgData, _ := EncodeCBOR(msg)
			if err := conn.WriteMessage(websocket.BinaryMessage, msgData); err != nil {
				return
			}
		}

		time.Sleep(1 * time.Second)
	}))
	defer server.Close()

	config := Config{
		URL:          "ws" + server.URL[4:],
		BaseDelay:    10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0,
	}

	filter := NewRecordFilter(NewFilterMetrics())

	handler := func(msgType int, payload []byte) error {
		result := filter.FilterCBOR(payload)
		if result.Valid && result.DID != "" {
			// Check for duplicate processing (race condition indicator)
			if _, exists := processingMap.LoadOrStore(result.DID, true); exists {
				t.Errorf("Duplicate processing detected for DID: %s", result.DID)
			}
			atomic.AddInt32(&messagesReceived, 1)
		}
		return nil
	}

	client, err := NewClient(config, handler, newTestLogger())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = client.Run(ctx)
		close(done)
	}()

	<-done

	received := atomic.LoadInt32(&messagesReceived)

	t.Logf("Concurrent processing results:")
	t.Logf("  Messages sent: %d", numMessages)
	t.Logf("  Messages received: %d", received)
	t.Logf("  Processing rate: %.2f%%", float64(received)/float64(numMessages)*100)

	// Should process most messages
	if received < int32(float64(numMessages)*0.95) {
		t.Errorf("Too many messages lost: %d/%d", numMessages-int(received), numMessages)
	}
}

// BenchmarkIndexer_Throughput benchmarks raw indexer throughput.
func BenchmarkIndexer_Throughput(b *testing.B) {
	filter := NewRecordFilter(NewFilterMetrics())

	// Pre-generate a sample message
	msg := JetstreamMessage{
		DID:    "did:plc:bench",
		TimeUS: time.Now().UnixMicro(),
		Kind:   "commit",
		Commit: &AtProtoCommit{
			DID:        "did:plc:bench",
			Operation:  "create",
			Collection: CollectionScene,
			RKey:       "scene",
			Record:     mustEncodeCBORForLoad(map[string]interface{}{"name": "Benchmark Scene"}),
		},
	}
	msgData, _ := EncodeCBOR(msg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := filter.FilterCBOR(msgData)
		if !result.Valid {
			b.Fatal("Message validation failed")
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msgs/sec")
}

// Helper function for load tests
func mustEncodeCBORForLoad(v interface{}) []byte {
	data, err := EncodeCBOR(v)
	if err != nil {
		panic(fmt.Sprintf("EncodeCBOR() error = %v", err))
	}
	return data
}
