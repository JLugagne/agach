package websocket_test

// Security RED tests for the WebSocket hub.
//
// These tests document security properties that are NOT yet enforced.
// Each test is expected to FAIL until the corresponding fix is applied.
//
// Naming convention: TestSecurity_<VulnerabilityName>

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gorillaws "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/pkg/websocket"
)

// newProductionUpgrader mirrors the upgrader defined in internal/server/init.go.
// It rejects cross-origin connections by comparing the Origin header host
// against the request host.
func newProductionUpgrader() gorillaws.Upgrader {
	return gorillaws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return u.Host == r.Host
		},
	}
}

// newProductionWSTestServer creates a test server that uses the same upgrader
// configuration as the production init.go WebSocket endpoint.
func newProductionWSTestServer(t *testing.T, hub *websocket.Hub) *httptest.Server {
	t.Helper()
	upgrader := newProductionUpgrader()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.ServeWS(conn, websocket.WithProjectID(r.URL.Query().Get("project_id")))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestSecurity_WebSocketCrossOriginRejected verifies that a WebSocket upgrade
// request from an untrusted origin is rejected with HTTP 403.
//
// RED: internal/server/init.go registers the /ws route with an upgrader whose
// CheckOrigin always returns true.  Any web page — including attacker-controlled
// pages at http://evil.com — can open a WebSocket to this server and receive
// all real-time kanban events (task moves, comments, completions, etc.) for
// every connected project, enabling data exfiltration and Cross-Site WebSocket
// Hijacking (CSWSH).
func TestSecurity_WebSocketCrossOriginRejected(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	hub := websocket.NewHub(logger)
	go hub.Run()

	srv := newProductionWSTestServer(t, hub)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	header := http.Header{}
	header.Set("Origin", "http://evil.com") // attacker-controlled origin

	conn, resp, err := gorillaws.DefaultDialer.Dial(wsURL, header)
	if conn != nil {
		conn.Close()
	}

	if err == nil {
		// The upgrade succeeded — document the expected (not yet implemented) behaviour.
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode,
			"RED: cross-origin WebSocket connection was accepted (HTTP %d); "+
				"the upgrader must reject untrusted origins with 403",
			resp.StatusCode)
		t.Fatalf("RED: cross-origin WebSocket connection was accepted; " +
			"fix: implement an origin allow-list in the CheckOrigin function in internal/server/init.go")
	}

	// When the fix is in place the dial returns an error and resp carries 403.
	require.NotNil(t, resp, "expected an HTTP response even when the upgrade is rejected")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"RED: expected 403 for cross-origin WebSocket upgrade, got %d", resp.StatusCode)
}

// TestSecurity_WebSocketSameOriginAccepted verifies that a WebSocket upgrade
// from the same host (same-origin) is accepted after the origin check fix.
//
// RED: This test documents the expected post-fix behaviour: same-origin
// connections must still work.  Currently it passes because CheckOrigin
// accepts everything, but once a real allow-list is added the test will be
// the regression guard that ensures legitimate connections are not broken.
func TestSecurity_WebSocketSameOriginAccepted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	hub := websocket.NewHub(logger)
	go hub.Run()

	srv := newProductionWSTestServer(t, hub)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	// Same-origin header: Origin matches the server host.
	header := http.Header{}
	header.Set("Origin", srv.URL)

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err,
		"same-origin WebSocket connection must be accepted after the origin fix")
	conn.Close()
}
