// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Backpressure thresholds and limits.
const (
	BackpressurePauseThreshold  = 1000             // Pause consumption when pending > 1000
	BackpressureResumeThreshold = 100              // Resume when pending < 100
	MaxPauseDuration            = 30 * time.Second // Max pause before warning
	QueueBufferSize             = 2000             // Buffer size = 2x pause threshold
)

// MessageHandler is a callback function for processing incoming messages.
// The handler receives the message type and payload.
// Return an error to signal the client should disconnect.
type MessageHandler func(messageType int, payload []byte) error

// Client is a resilient WebSocket client for connecting to Jetstream.
// It automatically reconnects with exponential backoff and jitter.
// Implements backpressure handling to prevent queue explosion.
// Supports resume from last processed sequence using cursor parameter.
type Client struct {
	config          Config
	handler         MessageHandler
	logger          *slog.Logger
	metrics         *Metrics
	sequenceTracker SequenceTracker

	mu               sync.Mutex
	rng              *rand.Rand // protected by mu
	conn             *websocket.Conn
	isConnected      bool
	isPaused         bool
	pauseStart       time.Time // protected by mu
	pauseInitialized bool      // protected by mu

	// Message queue for backpressure handling
	messageQueue chan queuedMessage

	// reconnectCount tracks consecutive reconnection attempts (atomic)
	reconnectCount int64
}

// queuedMessage wraps a message with its metadata for queuing
type queuedMessage struct {
	messageType int
	payload     []byte
}

// NewClient creates a new Jetstream WebSocket client with the given configuration.
// The handler function will be called for each incoming message.
// If metrics is nil, backpressure metrics will not be recorded.
func NewClient(config Config, handler MessageHandler, logger *slog.Logger) (*Client, error) {
	return NewClientWithMetrics(config, handler, logger, nil)
}

// NewClientWithMetrics creates a new Jetstream WebSocket client with metrics support.
// The handler function will be called for each incoming message.
// If metrics is provided, backpressure events will be recorded.
// If sequenceTracker is provided, the client will resume from the last processed sequence.
func NewClientWithMetrics(config Config, handler MessageHandler, logger *slog.Logger, metrics *Metrics) (*Client, error) {
	return NewClientWithSequenceTracker(config, handler, logger, metrics, nil)
}

// NewClientWithSequenceTracker creates a new Jetstream WebSocket client with full features.
// The handler function will be called for each incoming message.
// If metrics is provided, backpressure and reconnection events will be recorded.
// If sequenceTracker is provided, the client will resume from the last processed sequence on reconnect.
func NewClientWithSequenceTracker(config Config, handler MessageHandler, logger *slog.Logger, metrics *Metrics, sequenceTracker SequenceTracker) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		config:          config,
		handler:         handler,
		logger:          logger,
		metrics:         metrics,
		sequenceTracker: sequenceTracker,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		messageQueue:    make(chan queuedMessage, QueueBufferSize),
	}, nil
}

// Run starts the WebSocket client and blocks until the context is cancelled.
// It will automatically reconnect with exponential backoff on connection failures.
// Supports resume from last processed sequence if a SequenceTracker is configured.
func (c *Client) Run(ctx context.Context) error {
	// Start message processor goroutine
	processorCtx, processorCancel := context.WithCancel(ctx)
	defer processorCancel()

	processorDone := make(chan struct{})
	go func() {
		c.processMessages(processorCtx)
		close(processorDone)
	}()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("jetstream client stopping due to context cancellation")
			c.close()
			<-processorDone // Wait for processor to finish
			return ctx.Err()
		default:
		}

		// Attempt to connect
		if err := c.connect(ctx); err != nil {
			attempt := atomic.LoadInt64(&c.reconnectCount) + 1

			// Record reconnection attempt metric
			if c.metrics != nil {
				c.metrics.IncReconnectionAttempts()
			}

			c.logger.Warn("jetstream connection failed",
				slog.String("error", err.Error()),
				slog.Int64("attempt", attempt))

			// Check if we've exceeded max retry attempts
			if c.config.MaxRetryAttempts > 0 && attempt >= c.config.MaxRetryAttempts {
				c.logger.Error("max reconnection attempts reached - alerting required",
					slog.Int64("max_attempts", c.config.MaxRetryAttempts),
					slog.Int64("current_attempt", attempt),
					slog.String("error", err.Error()))
				// Continue trying to reconnect, but log at error level to alert monitoring
			}

			// Schedule reconnect with backoff
			delay := c.computeBackoff()
			atomic.AddInt64(&c.reconnectCount, 1)

			c.logger.Info("scheduling reconnect",
				slog.Duration("delay", delay),
				slog.Int64("attempt", atomic.LoadInt64(&c.reconnectCount)))

			select {
			case <-ctx.Done():
				<-processorDone // Wait for processor to finish
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		// Reset reconnect count on successful connection
		prevCount := atomic.SwapInt64(&c.reconnectCount, 0)

		// Track successful reconnection (only if this was actually a reconnect, not initial connect)
		if prevCount > 0 && c.metrics != nil {
			c.metrics.IncReconnectionSuccess()
			c.logger.Info("reconnection successful",
				slog.Int64("previous_attempts", prevCount))
		}

		// Read messages until connection closes
		c.readLoop(ctx)
	}
}

// connect establishes a WebSocket connection to the Jetstream endpoint.
// If a SequenceTracker is configured, it will add a cursor parameter to resume from the last sequence.
func (c *Client) connect(ctx context.Context) error {
	url := c.config.URL

	// Add cursor parameter if we have a sequence tracker
	if c.sequenceTracker != nil {
		lastSeq, err := c.sequenceTracker.GetLastSequence(ctx)
		if err != nil {
			c.logger.Warn("failed to get last sequence for resume",
				slog.String("error", err.Error()))
		} else if lastSeq > 0 {
			// Add cursor query parameter to resume from last sequence
			// Jetstream uses time_us as the cursor value
			if strings.Contains(url, "?") {
				url = fmt.Sprintf("%s&cursor=%d", url, lastSeq)
			} else {
				url = fmt.Sprintf("%s?cursor=%d", url, lastSeq)
			}
			c.logger.Info("resuming from last sequence",
				slog.Int64("cursor", lastSeq))
		}
	}

	c.logger.Info("connecting to jetstream", slog.String("url", url))

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.isConnected = true
	c.mu.Unlock()

	c.logger.Info("connected to jetstream")
	return nil
}

// readLoop reads messages from the WebSocket connection until it closes.
// Implements backpressure handling by pausing consumption when queue is full.
func (c *Client) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check backpressure and pause if necessary
		queueLen := len(c.messageQueue)
		if c.metrics != nil {
			c.metrics.SetPendingMessages(queueLen)
		}

		c.mu.Lock()

		// Pause if queue exceeds threshold
		if !c.isPaused && queueLen > BackpressurePauseThreshold {
			c.isPaused = true
			c.pauseStart = time.Now()
			c.pauseInitialized = true
			if c.metrics != nil {
				c.metrics.IncBackpressurePaused()
			}
			c.logger.Warn("backpressure: pausing message consumption",
				slog.Int("pending", queueLen),
				slog.Int("threshold", BackpressurePauseThreshold))
		}

		// Resume if queue drops below threshold
		if c.isPaused && queueLen < BackpressureResumeThreshold {
			var pauseDuration time.Duration
			if c.pauseInitialized {
				pauseDuration = time.Since(c.pauseStart)
			}
			c.isPaused = false
			c.pauseInitialized = false
			if c.metrics != nil {
				c.metrics.IncBackpressureResumed()
				c.metrics.ObserveBackpressureDuration(pauseDuration.Seconds())
			}
			c.logger.Info("backpressure: resuming message consumption",
				slog.Int("pending", queueLen),
				slog.Duration("pause_duration", pauseDuration))
		}

		// Check for excessive pause duration
		if c.isPaused && c.pauseInitialized && time.Since(c.pauseStart) > MaxPauseDuration {
			c.logger.Warn("backpressure: exceeded max pause duration",
				slog.Int("pending", queueLen),
				slog.Duration("pause_duration", time.Since(c.pauseStart)),
				slog.Duration("max_pause", MaxPauseDuration))
			// Reset pause start to avoid spamming warnings
			c.pauseStart = time.Now()
		}

		isPaused := c.isPaused
		c.mu.Unlock()

		// If paused, wait a bit before checking again
		if isPaused {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		// Get connection under lock to prevent race with close()
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			// Connection was closed, exit loop
			return
		}

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			c.logger.Warn("jetstream connection closed",
				slog.String("error", err.Error()))
			c.close()
			return
		}

		// Queue message for processing (non-blocking with timeout)
		// Copy payload to avoid issues with buffer reuse
		payloadCopy := make([]byte, len(payload))
		copy(payloadCopy, payload)
		msg := queuedMessage{messageType: messageType, payload: payloadCopy}
		select {
		case c.messageQueue <- msg:
			// Message queued successfully
		case <-time.After(5 * time.Second):
			// Queue is full and blocking for too long - close connection to force reconnect
			c.logger.Error("backpressure: failed to queue message after timeout, closing connection",
				slog.Int("pending", len(c.messageQueue)))
			c.close()
			return
		case <-ctx.Done():
			return
		}
	}
}

// close cleanly closes the WebSocket connection.
func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Record pause duration if we're closing while paused
	if c.isPaused && c.pauseInitialized && c.metrics != nil {
		pauseDuration := time.Since(c.pauseStart)
		c.metrics.IncBackpressureResumed()
		c.metrics.ObserveBackpressureDuration(pauseDuration.Seconds())
		c.logger.Info("backpressure: connection closed during pause, recording duration",
			slog.Duration("pause_duration", pauseDuration))
	}

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.isConnected = false
	c.isPaused = false
	c.pauseInitialized = false
	c.pauseStart = time.Time{}
}

// processMessages processes messages from the queue.
// This runs in a separate goroutine to decouple reading from processing.
func (c *Client) processMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Drain remaining messages before exiting
			c.drainQueue()
			return
		case msg := <-c.messageQueue:
			// Process message through handler
			if c.handler != nil {
				if err := c.handler(msg.messageType, msg.payload); err != nil {
					c.logger.Error("message handler error",
						slog.String("error", err.Error()))
					// Continue processing other messages despite error
				}
			}
			// Update pending count after processing
			if c.metrics != nil {
				c.metrics.SetPendingMessages(len(c.messageQueue))
			}
		}
	}
}

// drainQueue processes any remaining messages in the queue before shutdown.
func (c *Client) drainQueue() {
	remaining := len(c.messageQueue)
	if remaining > 0 {
		c.logger.Info("draining message queue", slog.Int("remaining", remaining))
	}

	// Process remaining messages with timeout
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg := <-c.messageQueue:
			if c.handler != nil {
				if err := c.handler(msg.messageType, msg.payload); err != nil {
					c.logger.Error("message handler error during drain",
						slog.String("error", err.Error()))
				}
			}
		case <-timeout:
			remaining := len(c.messageQueue)
			if remaining > 0 {
				c.logger.Warn("queue drain timeout, messages remaining",
					slog.Int("remaining", remaining))
			}
			return
		default:
			// Queue is empty
			return
		}
	}
}

// computeBackoff calculates the next reconnection delay with exponential backoff and jitter.
func (c *Client) computeBackoff() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Exponential backoff: baseDelay * 2^attempts using bit shifting
	// Cap the shift at 30 to prevent overflow (2^30 = ~1 billion)
	reconnectCount := atomic.LoadInt64(&c.reconnectCount)
	shift := uint(reconnectCount)
	if shift > 30 {
		shift = 30
	}
	backoff := float64(c.config.BaseDelay) * float64(uint64(1)<<shift)

	// Cap at max delay
	if backoff > float64(c.config.MaxDelay) {
		backoff = float64(c.config.MaxDelay)
	}

	// Apply jitter: delay * (1 - jitter/2 + rand*jitter)
	// This creates a range of [delay*(1-jitter/2), delay*(1+jitter/2)]
	if c.config.JitterFactor > 0 {
		jitter := (c.rng.Float64() - 0.5) * c.config.JitterFactor
		backoff = backoff * (1 + jitter)
	}

	return time.Duration(backoff)
}

// IsConnected returns whether the client is currently connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

// UpdateSequence updates the last processed sequence number.
// This should be called by the message handler after successfully processing a message.
// The sequence is extracted from the TimeUS field of JetstreamMessage.
func (c *Client) UpdateSequence(ctx context.Context, sequence int64) error {
	if c.sequenceTracker == nil {
		return nil // No tracker configured, nothing to do
	}

	return c.sequenceTracker.UpdateSequence(ctx, sequence)
}
