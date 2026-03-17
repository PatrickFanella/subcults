// Package stream provides WebSocket event broadcasting for real-time participant updates.
package stream

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// connWrapper wraps a WebSocket connection with a write mutex for safe concurrent writes.
type connWrapper struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// EventBroadcaster manages WebSocket connections and broadcasts participant events.
type EventBroadcaster struct {
	mu          sync.RWMutex
	connections map[string]map[*connWrapper]bool // streamSessionID -> connections
}

// NewEventBroadcaster creates a new event broadcaster.
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		connections: make(map[string]map[*connWrapper]bool),
	}
}

// Subscribe registers a WebSocket connection for a stream session.
func (b *EventBroadcaster) Subscribe(streamSessionID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	wrapper := &connWrapper{conn: conn}
	if b.connections[streamSessionID] == nil {
		b.connections[streamSessionID] = make(map[*connWrapper]bool)
	}
	b.connections[streamSessionID][wrapper] = true
}

// Unsubscribe removes a WebSocket connection from all stream sessions.
func (b *EventBroadcaster) Unsubscribe(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for streamID, conns := range b.connections {
		for wrapper := range conns {
			if wrapper.conn == conn {
				delete(conns, wrapper)
			}
		}
		if len(conns) == 0 {
			delete(b.connections, streamID)
		}
	}
}

// Broadcast sends a participant event to all subscribers of a stream.
func (b *EventBroadcaster) Broadcast(streamSessionID string, event *ParticipantStateEvent) {
	// Serialize event once before acquiring locks
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal participant event", "error", err)
		return
	}

	// Get snapshot of connections under read lock
	b.mu.RLock()
	conns, exists := b.connections[streamSessionID]
	if !exists || len(conns) == 0 {
		b.mu.RUnlock()
		return
	}

	// Create a snapshot to avoid holding the lock during I/O
	snapshot := make([]*connWrapper, 0, len(conns))
	for wrapper := range conns {
		snapshot = append(snapshot, wrapper)
	}
	b.mu.RUnlock()

	// Broadcast to all connections (with per-connection write mutex)
	deadConns := make([]*connWrapper, 0)
	for _, wrapper := range snapshot {
		wrapper.mu.Lock()
		err := wrapper.conn.WriteMessage(websocket.TextMessage, data)
		wrapper.mu.Unlock()

		if err != nil {
			slog.Warn("failed to send message to websocket client",
				"error", err,
				"stream_session_id", streamSessionID,
			)
			// Track dead connections for cleanup
			deadConns = append(deadConns, wrapper)
		}
	}

	// Clean up dead connections
	if len(deadConns) > 0 {
		b.mu.Lock()
		for _, wrapper := range deadConns {
			if conns, exists := b.connections[streamSessionID]; exists {
				delete(conns, wrapper)
				if len(conns) == 0 {
					delete(b.connections, streamSessionID)
				}
			}
		}
		b.mu.Unlock()
	}
}

// ConnectionCount returns the number of active WebSocket connections for a stream.
func (b *EventBroadcaster) ConnectionCount(streamSessionID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if conns, exists := b.connections[streamSessionID]; exists {
		return len(conns)
	}
	return 0
}
