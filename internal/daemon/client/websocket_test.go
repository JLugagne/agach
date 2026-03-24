package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func newTestLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func TestWSClient_Connect_Success(t *testing.T) {
	var receivedToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.URL.Query().Get("token")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewWSClient(wsURL+"/ws", "my-jwt-token", newTestLogger(), nil)

	err := c.Connect(context.Background())
	require.NoError(t, err)
	assert.True(t, c.IsConnected())
	assert.Equal(t, "my-jwt-token", receivedToken)

	c.Disconnect()
	assert.False(t, c.IsConnected())
}

func TestWSClient_Connect_InvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewWSClient(wsURL+"/ws", "bad-token", newTestLogger(), nil)

	err := c.Connect(context.Background())
	require.Error(t, err)
	assert.False(t, c.IsConnected())
}

func TestWSClient_ReceivesEvents(t *testing.T) {
	event := WSEvent{
		Type:      "task_updated",
		ProjectID: "proj-1",
		Data:      json.RawMessage(`{"id":"t1"}`),
	}
	eventBytes, _ := json.Marshal(event)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.TextMessage, eventBytes)
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	received := make(chan WSEvent, 1)
	handler := func(e WSEvent) {
		received <- e
	}

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewWSClient(wsURL+"/ws", "token", newTestLogger(), handler)

	err := c.Connect(context.Background())
	require.NoError(t, err)

	select {
	case e := <-received:
		assert.Equal(t, "task_updated", e.Type)
		assert.Equal(t, "proj-1", e.ProjectID)
		assert.Equal(t, `{"id":"t1"}`, string(e.Data))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	c.Disconnect()
}

func TestWSClient_Reconnects(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewWSClient(wsURL+"/ws", "token", newTestLogger(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	_ = c.RunWithReconnect(ctx)

	assert.GreaterOrEqual(t, int(attempts.Load()), 2, "expected multiple connection attempts")
}
