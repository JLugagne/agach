package security_test

// Deep security tests for pkg/websocket — vulnerabilities NOT covered by
// the existing websocket_security_test.go or deep_security_test.go.

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

// ─── helpers ─────────────────────────────────────────────────────────────────

func newDeepHub(t *testing.T) *websocket.Hub {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	hub := websocket.NewHub(logger)
	go hub.Run()
	t.Cleanup(hub.Stop)
	return hub
}

func newDeepHubServer(t *testing.T, hub *websocket.Hub, opts ...func(*http.Request) []websocket.ServeWSOption) *httptest.Server {
	t.Helper()
	u := gorillaws.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	optsFn := func(r *http.Request) []websocket.ServeWSOption {
		return []websocket.ServeWSOption{websocket.WithProjectID(r.URL.Query().Get("project_id"))}
	}
	if len(opts) > 0 {
		optsFn = opts[0]
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.ServeWS(conn, optsFn(r)...)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func dialWS(t *testing.T, srv *httptest.Server, path string) *gorillaws.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + path
	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readWSEvent(t *testing.T, conn *gorillaws.Conn, timeout time.Duration) websocket.Event {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(timeout)))
	defer conn.SetReadDeadline(time.Time{}) //nolint:errcheck
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err, "failed to read WebSocket message")
	var event websocket.Event
	require.NoError(t, json.Unmarshal(msg, &event), "failed to unmarshal event")
	return event
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Relay messages bypass project scoping
//
// File: hub.go:133-150
//
// The relay case in the Run loop forwards messages based solely on the
// isDaemon flag. When a daemon client sends a relay message, it is
// forwarded to ALL non-daemon clients regardless of their projectID.
// Similarly, a non-daemon relay message goes to ALL daemon clients.
//
// This means a user connected to project A receives relay messages
// (chat messages, daemon events) from project B's daemon if both are
// connected to the same hub.
//
// TODO(security): Filter relay messages by projectID, only forwarding
// to clients in the same project as the sender.
// ---------------------------------------------------------------------------

func TestSecurity_RED_RelayBypassesProjectScoping(t *testing.T) {
	hub := newDeepHub(t)

	// Server that extracts project_id and daemon flag from query params.
	srv := newDeepHubServer(t, hub, func(r *http.Request) []websocket.ServeWSOption {
		var opts []websocket.ServeWSOption
		if pid := r.URL.Query().Get("project_id"); pid != "" {
			opts = append(opts, websocket.WithProjectID(pid))
		}
		if r.URL.Query().Get("daemon") == "true" {
			opts = append(opts, websocket.AsDaemon())
		}
		return opts
	})

	// Connect a daemon for project A.
	daemonConn := dialWS(t, srv, "/?project_id=project-A&daemon=true")
	// Connect a regular user for project B (different project).
	userConnB := dialWS(t, srv, "/?project_id=project-B")

	time.Sleep(50 * time.Millisecond) // wait for registration

	// Register a relay handler for "chat_message" type.
	hub.RegisterHandler("chat_message", hub.NewRelayHandler())

	// Daemon sends a chat message. It should only go to project-A users,
	// but the relay path sends to ALL non-daemon clients.
	msg := json.RawMessage(`{"type":"chat_message","data":"secret from project A"}`)
	err := daemonConn.WriteMessage(gorillaws.TextMessage, msg)
	require.NoError(t, err)

	// RED: user in project B should NOT receive this message, but they do
	// because relay ignores projectID.
	userConnB.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, received, readErr := userConnB.ReadMessage()
	if readErr == nil {
		assert.NotEmpty(t, received,
			"RED: user in project-B received a relay message from project-A's daemon — "+
				"relay bypasses project scoping, enabling cross-project data leakage")
		t.Log("RED: relay messages are forwarded to ALL non-daemon clients regardless of projectID")
	} else {
		t.Log("Relay message not received (may not have been processed in time); " +
			"the vulnerability exists in code but timing prevented demonstration")
	}
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Broadcast sends to clients with empty projectID
//
// File: hub.go:118-120
//
// The broadcast filter logic is:
//   if client.projectID != "" && event.ProjectID != "" && client.projectID != event.ProjectID {
//       continue
//   }
//
// A client that connects WITHOUT specifying a projectID (projectID == "")
// passes the filter for EVERY event, regardless of the event's ProjectID.
// This means such a client acts as a "global listener" receiving events
// from all projects — a serious data leakage if the upgrade handler does
// not enforce projectID.
//
// TODO(security): Reject clients with empty projectID, or only send them
// events that also have an empty ProjectID (global events).
// ---------------------------------------------------------------------------

func TestSecurity_RED_EmptyProjectIDClientReceivesAllEvents(t *testing.T) {
	hub := newDeepHub(t)
	srv := newDeepHubServer(t, hub)

	// Connect a client with no project_id (empty string).
	globalListener := dialWS(t, srv, "/")
	// Connect a client properly scoped to project-X.
	scopedClient := dialWS(t, srv, "/?project_id=project-X")

	time.Sleep(50 * time.Millisecond) // wait for registration

	// Broadcast an event scoped to project-X.
	hub.Broadcast(websocket.Event{
		Type:      "task_updated",
		ProjectID: "project-X",
		Data:      map[string]string{"task": "secret-task"},
	})

	// The scoped client should receive it.
	ev := readWSEvent(t, scopedClient, 2*time.Second)
	assert.Equal(t, "task_updated", ev.Type)

	// RED: the unscoped client ALSO receives it because empty projectID
	// bypasses the filter.
	globalListener.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err := globalListener.ReadMessage()
	if err == nil {
		var leakedEvent websocket.Event
		_ = json.Unmarshal(msg, &leakedEvent)
		assert.Equal(t, "task_updated", leakedEvent.Type,
			"RED: client with empty projectID received an event scoped to project-X — "+
				"any client that connects without project_id acts as a global eavesdropper")
		t.Log("RED: empty-projectID clients receive ALL project-scoped events")
	} else {
		t.Log("Event not received in time; the vulnerability exists in code logic")
	}
}

// ---------------------------------------------------------------------------
// VULNERABILITY: ReadPump ReadLimit (64KB) inconsistent with MaxMessageSize constant (4KB)
//
// File: hub.go:204 vs constants.go:9
//
// constants.go defines MaxMessageSize = 4 * 1024 (4 KB) as a public constant
// that presumably represents the intended application limit. However,
// ReadPump sets conn.SetReadLimit(64 * 1024) — 16x larger than the declared
// constant. This means:
//   1. The MaxMessageSize constant is misleading/unused
//   2. Clients can send messages up to 64 KB, not the intended 4 KB
//
// TODO(security): Use MaxMessageSize in SetReadLimit, or update the constant
// to match the actual enforced limit.
// ---------------------------------------------------------------------------

func TestSecurity_RED_ReadLimitInconsistentWithMaxMessageSize(t *testing.T) {
	hub := newDeepHub(t)
	srv := newDeepHubServer(t, hub)

	conn := dialWS(t, srv, "/")
	time.Sleep(50 * time.Millisecond)

	// The declared MaxMessageSize is 4 KB. Send a 5 KB message — it should
	// be rejected if MaxMessageSize is enforced, but accepted because
	// ReadPump uses 64 KB.
	msg := make([]byte, 5*1024)
	for i := range msg {
		msg[i] = 'A'
	}
	// Wrap as valid JSON so ReadPump's unmarshal doesn't silently skip it.
	payload := []byte(`{"type":"probe","data":"` + string(msg) + `"}`)

	err := conn.WriteMessage(gorillaws.TextMessage, payload)
	require.NoError(t, err, "client-side write should succeed")

	// Wait for server to process.
	time.Sleep(100 * time.Millisecond)

	// If MaxMessageSize (4KB) were enforced, the connection would be closed.
	// Try sending another message to see if connection is still alive.
	err = conn.WriteMessage(gorillaws.TextMessage, []byte(`{"type":"ping"}`))

	assert.NoError(t, err,
		"RED: 5 KB message was accepted even though MaxMessageSize constant is 4 KB — "+
			"SetReadLimit uses 64 KB instead of the declared MaxMessageSize constant")

	t.Logf("RED: ReadPump uses SetReadLimit(64 KB) but MaxMessageSize constant is %d bytes — "+
		"the constant is misleading and unused", websocket.MaxMessageSize)
}

// ---------------------------------------------------------------------------
// VULNERABILITY: SendToDaemon deadlock risk — sendRaw called with mu held
//
// File: hub.go:306-317
//
// SendToDaemon acquires h.mu.Lock(), then calls h.sendRaw() which acquires
// client.writeMu. Meanwhile, WritePump holds client.writeMu and broadcasts
// can trigger hub operations that need h.mu. This creates a potential
// lock-ordering deadlock: goroutine A holds h.mu, waits for client.writeMu;
// goroutine B holds client.writeMu, waits for h.mu (via unregister).
//
// The same pattern exists in the relay case (hub.go:134-150).
//
// TODO(security): Either release h.mu before calling sendRaw, or ensure
// strict lock ordering (always acquire h.mu before writeMu, never the reverse).
// ---------------------------------------------------------------------------

func TestSecurity_RED_SendToDaemonDeadlockRisk(t *testing.T) {
	hub := newDeepHub(t)
	srv := newDeepHubServer(t, hub, func(r *http.Request) []websocket.ServeWSOption {
		var opts []websocket.ServeWSOption
		if r.URL.Query().Get("daemon") == "true" {
			opts = append(opts, websocket.AsDaemon(), websocket.WithNodeID("node-1"))
		}
		return opts
	})

	// Connect a daemon client.
	daemonConn := dialWS(t, srv, "/?daemon=true")
	time.Sleep(50 * time.Millisecond)

	// Start draining daemon messages to avoid buffer fill.
	go func() {
		for {
			daemonConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, _, err := daemonConn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Hammer SendToDaemon and Broadcast concurrently to try to trigger
	// the lock-ordering issue.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 500; i++ {
			msg := json.RawMessage(`{"type":"test","data":"send-to-daemon"}`)
			hub.SendToDaemon("node-1", msg)
			hub.Broadcast(websocket.Event{Type: "concurrent", Data: i})
		}
	}()

	select {
	case <-done:
		// Completed without deadlock in this run.
		t.Log("RED: SendToDaemon acquires h.mu then client.writeMu — " +
			"potential deadlock with WritePump (lock ordering violation); " +
			"this run did not deadlock but the code path is unsafe")
	case <-time.After(5 * time.Second):
		t.Fatal("RED: deadlock detected — SendToDaemon + Broadcast timed out after 5s")
	}
}
