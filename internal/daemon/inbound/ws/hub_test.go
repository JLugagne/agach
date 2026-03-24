package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/inbound/ws"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestHub(t *testing.T) *ws.Hub {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return ws.NewHub(logger)
}

// startHubServer starts an httptest server that upgrades connections and passes them to hub.HandleConnection.
func startHubServer(t *testing.T, hub *ws.Hub) (*httptest.Server, func() *gorillaWS.Conn) {
	t.Helper()
	upgrader := gorillaWS.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.HandleConnection(conn)
	}))
	t.Cleanup(srv.Close)

	dial := func() *gorillaWS.Conn {
		url := "ws" + strings.TrimPrefix(srv.URL, "http")
		conn, _, err := gorillaWS.DefaultDialer.Dial(url, nil)
		require.NoError(t, err)
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	return srv, dial
}

func TestHub_RequestResponse(t *testing.T) {
	hub := newTestHub(t)
	hub.RegisterHandler(daemonws.TypeDockerList, func(ctx context.Context, msg daemonws.Message) (daemonws.Message, error) {
		return daemonws.Message{
			Type:      daemonws.TypeDockerList,
			RequestID: msg.RequestID,
			Payload:   json.RawMessage(`{"images":[]}`),
		}, nil
	})
	go hub.Run()

	_, dial := startHubServer(t, hub)
	conn := dial()

	// Send request
	req := daemonws.Message{
		Type:      daemonws.TypeDockerList,
		RequestID: "abc123",
	}
	err := conn.WriteJSON(req)
	require.NoError(t, err)

	// Read response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp daemonws.Message
	err = conn.ReadJSON(&resp)
	require.NoError(t, err)
	require.Equal(t, "abc123", resp.RequestID, "response must have same request_id")
	require.Equal(t, daemonws.TypeDockerList, resp.Type)
}

func TestHub_EventPush(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	_, dial := startHubServer(t, hub)
	conn := dial()

	// Give connection time to register
	time.Sleep(50 * time.Millisecond)

	// Push event
	event := daemonws.Message{
		Type:    daemonws.TypeBuildEvent,
		Payload: json.RawMessage(`{"status":"started"}`),
	}
	hub.SendEvent(event)

	// Read event
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var received daemonws.Message
	err := conn.ReadJSON(&received)
	require.NoError(t, err)
	require.Equal(t, daemonws.TypeBuildEvent, received.Type)
}

func TestHub_MultipleConnections(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	_, dial := startHubServer(t, hub)
	conn1 := dial()
	conn2 := dial()

	time.Sleep(50 * time.Millisecond)

	event := daemonws.Message{
		Type:    daemonws.TypeBuildEvent,
		Payload: json.RawMessage(`{"status":"completed"}`),
	}
	hub.SendEvent(event)

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))

	var msg1, msg2 daemonws.Message
	require.NoError(t, conn1.ReadJSON(&msg1))
	require.NoError(t, conn2.ReadJSON(&msg2))
	require.Equal(t, daemonws.TypeBuildEvent, msg1.Type)
	require.Equal(t, daemonws.TypeBuildEvent, msg2.Type)
}

func TestHub_ConnectionCleanup(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	_, dial := startHubServer(t, hub)
	conn := dial()

	time.Sleep(50 * time.Millisecond)

	// Close connection
	conn.Close()
	time.Sleep(50 * time.Millisecond)

	// SendEvent should not panic
	require.NotPanics(t, func() {
		hub.SendEvent(daemonws.Message{Type: "test"})
	})
}
