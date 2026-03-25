package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
)

// newTestHub creates a hub for testing and starts its Run loop in a goroutine.
func newTestHub(t *testing.T) *websocket.Hub {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // quiet during tests
	hub := websocket.NewHub(logger)
	go hub.Run()
	return hub
}

// newTestWSServer creates an HTTP test server that upgrades connections and
// delegates to the hub's ServeWS method.
func newTestWSServer(t *testing.T, hub *websocket.Hub) *httptest.Server {
	t.Helper()

	upgrader := gorillaws.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "upgrade failed", http.StatusInternalServerError)
			return
		}
		hub.ServeWS(conn)
	}))

	t.Cleanup(srv.Close)
	return srv
}

// dialWS dials the test WebSocket server and returns the connection.
func dialWS(t *testing.T, srv *httptest.Server) *gorillaws.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := gorillaws.DefaultDialer.Dial(url, nil)
	require.NoError(t, err, "failed to dial WebSocket server")

	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

// readEvent reads one JSON-encoded Event from the WebSocket connection with a timeout.
func readEvent(t *testing.T, conn *gorillaws.Conn, timeout time.Duration) websocket.Event {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(timeout)))
	defer conn.SetReadDeadline(time.Time{}) //nolint:errcheck

	_, msg, err := conn.ReadMessage()
	require.NoError(t, err, "failed to read WebSocket message")

	var event websocket.Event
	require.NoError(t, json.Unmarshal(msg, &event), "failed to unmarshal event")
	return event
}

// waitForRegistration adds a small sleep so the hub's register goroutine has
// time to process the client before we broadcast.
func waitForRegistration() {
	time.Sleep(20 * time.Millisecond)
}

// TestHub_BroadcastToSingleClient verifies that a single connected client
// receives a broadcast event with the correct type and data.
func TestHub_BroadcastToSingleClient(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn := dialWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{
		Type: "project_created",
		Data: map[string]interface{}{"id": "proj-1", "name": "My Project"},
	})

	event := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "project_created", event.Type)

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok, "event.Data should be a map")
	assert.Equal(t, "proj-1", data["id"])
	assert.Equal(t, "My Project", data["name"])
}

// TestHub_BroadcastToMultipleClients verifies that all connected clients
// receive the same broadcast event.
func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn1 := dialWS(t, srv)
	conn2 := dialWS(t, srv)
	conn3 := dialWS(t, srv)
	waitForRegistration()

	expectedType := "task_moved"
	hub.Broadcast(websocket.Event{
		Type:      expectedType,
		ProjectID: "proj-42",
		Data:      map[string]interface{}{"task_id": "task-7", "target_column": "in_progress"},
	})

	for i, conn := range []*gorillaws.Conn{conn1, conn2, conn3} {
		event := readEvent(t, conn, 2*time.Second)
		assert.Equal(t, expectedType, event.Type, "client %d should receive the event", i+1)
		assert.Equal(t, "proj-42", event.ProjectID, "client %d should see project_id", i+1)
	}
}

// TestHub_BroadcastProjectCreatedEvent verifies the project_created event
// format broadcast by the hub.
func TestHub_BroadcastProjectCreatedEvent(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn := dialWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{
		Type: "project_created",
		Data: map[string]interface{}{
			"id":        "project-123",
			"name":      "Integration Test Project",
			"parent_id": nil,
		},
	})

	event := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "project_created", event.Type)
	assert.Empty(t, event.ProjectID, "project_created event should have no project_id field")

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "project-123", data["id"])
	assert.Equal(t, "Integration Test Project", data["name"])
}

// TestHub_BroadcastTaskMovedEvent verifies the task_moved event includes
// the project_id and correct task data.
func TestHub_BroadcastTaskMovedEvent(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn := dialWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{
		Type:      "task_moved",
		ProjectID: "project-abc",
		Data: map[string]interface{}{
			"task_id":       "task-xyz",
			"target_column": "in_progress",
		},
	})

	event := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "task_moved", event.Type)
	assert.Equal(t, "project-abc", event.ProjectID)

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "task-xyz", data["task_id"])
	assert.Equal(t, "in_progress", data["target_column"])
}

// TestHub_BroadcastTaskCreatedEvent verifies the task_created event format.
func TestHub_BroadcastTaskCreatedEvent(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn := dialWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{
		Type:      "task_created",
		ProjectID: "project-abc",
		Data:      map[string]interface{}{"id": "task-1", "title": "Do something"},
	})

	event := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "task_created", event.Type)
	assert.Equal(t, "project-abc", event.ProjectID)
}

// TestHub_BroadcastTaskBlockedEvent verifies the task_blocked event format.
func TestHub_BroadcastTaskBlockedEvent(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn := dialWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{
		Type:      "task_blocked",
		ProjectID: "project-abc",
		Data: map[string]interface{}{
			"task_id":          "task-1",
			"blocked_reason":   "Cannot proceed",
			"blocked_by_agent": "agent-1",
		},
	})

	event := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "task_blocked", event.Type)
	assert.Equal(t, "project-abc", event.ProjectID)

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "task-1", data["task_id"])
	assert.Equal(t, "Cannot proceed", data["blocked_reason"])
}

// TestHub_BroadcastMultipleEvents verifies that consecutive events are
// delivered to a client in order.
func TestHub_BroadcastMultipleEvents(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn := dialWS(t, srv)
	waitForRegistration()

	events := []websocket.Event{
		{Type: "project_created", Data: map[string]interface{}{"id": "p1"}},
		{Type: "task_created", ProjectID: "p1", Data: map[string]interface{}{"id": "t1"}},
		{Type: "task_moved", ProjectID: "p1", Data: map[string]interface{}{"task_id": "t1", "target_column": "in_progress"}},
	}

	for _, e := range events {
		hub.Broadcast(e)
	}

	for i, expected := range events {
		event := readEvent(t, conn, 2*time.Second)
		assert.Equal(t, expected.Type, event.Type, "event %d type mismatch", i)
	}
}

// TestHub_ClientDisconnect verifies that the hub handles a disconnected
// client gracefully and continues broadcasting to remaining clients.
func TestHub_ClientDisconnect(t *testing.T) {
	hub := newTestHub(t)
	srv := newTestWSServer(t, hub)

	conn1 := dialWS(t, srv)
	conn2 := dialWS(t, srv)
	waitForRegistration()

	// Disconnect conn1
	conn1.Close()

	// Give the hub time to detect the disconnect via the read pump
	time.Sleep(50 * time.Millisecond)

	// Broadcast should still reach conn2 without panicking
	hub.Broadcast(websocket.Event{
		Type: "task_created",
		Data: map[string]interface{}{"id": "task-after-disconnect"},
	})

	event := readEvent(t, conn2, 2*time.Second)
	assert.Equal(t, "task_created", event.Type)
}

// TestHub_NoClientsConnected verifies that broadcasting when no clients
// are connected does not block or panic.
func TestHub_NoClientsConnected(t *testing.T) {
	hub := newTestHub(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		hub.Broadcast(websocket.Event{
			Type: "project_created",
			Data: map[string]interface{}{"id": "p1"},
		})
	}()

	select {
	case <-done:
		// success — did not block
	case <-time.After(1 * time.Second):
		t.Fatal("Broadcast blocked when no clients were connected")
	}
}
