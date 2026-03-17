package stream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// testWSServer creates a test WebSocket server that upgrades HTTP connections.
// Returns the server and a dialer function to connect to it.
func testWSServer(t *testing.T) (*httptest.Server, func() *websocket.Conn) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		// Keep connection alive until closed
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))

	dial := func() *websocket.Conn {
		url := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial failed: %v", err)
		}
		return conn
	}

	return server, dial
}

func TestNewEventBroadcaster(t *testing.T) {
	b := NewEventBroadcaster()
	if b == nil {
		t.Fatal("expected non-nil broadcaster")
	}
	if b.connections == nil {
		t.Fatal("expected initialized connections map")
	}
}

func TestEventBroadcaster_SubscribeAndConnectionCount(t *testing.T) {
	server, dial := testWSServer(t)
	defer server.Close()

	b := NewEventBroadcaster()

	if count := b.ConnectionCount("session-1"); count != 0 {
		t.Errorf("expected 0 connections, got %d", count)
	}

	conn1 := dial()
	defer conn1.Close()
	conn2 := dial()
	defer conn2.Close()

	b.Subscribe("session-1", conn1)
	b.Subscribe("session-1", conn2)

	if count := b.ConnectionCount("session-1"); count != 2 {
		t.Errorf("expected 2 connections, got %d", count)
	}

	// Different session
	if count := b.ConnectionCount("session-2"); count != 0 {
		t.Errorf("expected 0 connections for different session, got %d", count)
	}
}

func TestEventBroadcaster_Unsubscribe(t *testing.T) {
	server, dial := testWSServer(t)
	defer server.Close()

	b := NewEventBroadcaster()

	conn1 := dial()
	defer conn1.Close()
	conn2 := dial()
	defer conn2.Close()

	b.Subscribe("session-1", conn1)
	b.Subscribe("session-1", conn2)

	b.Unsubscribe(conn1)

	if count := b.ConnectionCount("session-1"); count != 1 {
		t.Errorf("expected 1 connection after unsubscribe, got %d", count)
	}

	// Unsubscribe the last one - session entry should be cleaned up
	b.Unsubscribe(conn2)

	if count := b.ConnectionCount("session-1"); count != 0 {
		t.Errorf("expected 0 connections after full unsubscribe, got %d", count)
	}
}

func TestEventBroadcaster_Unsubscribe_NonExistent(t *testing.T) {
	server, dial := testWSServer(t)
	defer server.Close()

	b := NewEventBroadcaster()
	conn := dial()
	defer conn.Close()

	// Should not panic
	b.Unsubscribe(conn)
}

func TestEventBroadcaster_Broadcast(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	// Use a channel to safely pass the server-side connection to the test goroutine
	serverConnCh := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnCh <- conn
	}))
	defer server.Close()

	// Dial and get client connections
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer clientConn.Close()

	// Wait for server to accept and get the server-side connection safely
	var serverConn *websocket.Conn
	select {
	case serverConn = <-serverConnCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server connection")
	}

	b := NewEventBroadcaster()
	// Subscribe the server-side connection (which is what Broadcast writes to)
	b.Subscribe("session-1", serverConn)

	event := &ParticipantStateEvent{
		Type:            "participant_joined",
		StreamSessionID: "session-1",
		ParticipantID:   "user-abc",
		UserDID:         "did:plc:abc",
		Timestamp:       time.Now(),
		ActiveCount:     1,
	}

	// Broadcast should write to the server connection
	// The client reads from the other end
	b.Broadcast("session-1", event)

	// Read from client side
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var received ParticipantStateEvent
	if err := json.Unmarshal(message, &received); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if received.Type != "participant_joined" {
		t.Errorf("expected type participant_joined, got %s", received.Type)
	}
	if received.ParticipantID != "user-abc" {
		t.Errorf("expected participant user-abc, got %s", received.ParticipantID)
	}
}

func TestEventBroadcaster_Broadcast_NoSubscribers(t *testing.T) {
	b := NewEventBroadcaster()

	event := &ParticipantStateEvent{
		Type:            "participant_joined",
		StreamSessionID: "nonexistent",
		ParticipantID:   "user-abc",
	}

	// Should not panic
	b.Broadcast("nonexistent", event)
}

func TestEventBroadcaster_ConnectionCount_EmptySession(t *testing.T) {
	b := NewEventBroadcaster()

	if count := b.ConnectionCount(""); count != 0 {
		t.Errorf("expected 0 for empty session, got %d", count)
	}
	if count := b.ConnectionCount("nonexistent"); count != 0 {
		t.Errorf("expected 0 for nonexistent session, got %d", count)
	}
}
