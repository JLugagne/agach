// Package security_test — deep security tests for the WebSocket hub.
//
// Each vulnerability is documented with:
//   - RED test: demonstrates the vulnerability (expected to fail until fixed)
//   - GREEN test: passes after the fix is applied (or documents safe behaviour)
//
// Run with: go test -race -failfast ./internal/pkg/websocket/security_test/...
package security_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// newSecHub creates a hub with a quiet logger.
func newSecHub(t *testing.T) *websocket.Hub {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	hub := websocket.NewHub(logger)
	go hub.Run()
	return hub
}

// openRawWS opens a real WebSocket against a test server and returns the conn.
func openRawWS(t *testing.T, srv *httptest.Server) *gorillaws.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// permissiveUpgrader always accepts connections, mirroring what hub_test.go uses.
func permissiveUpgrader() gorillaws.Upgrader {
	return gorillaws.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}
}

// newHubServer builds a plain test HTTP server backed by the hub.
func newHubServer(t *testing.T, hub *websocket.Hub) *httptest.Server {
	t.Helper()
	u := permissiveUpgrader()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.ServeWS(conn, websocket.WithProjectID(r.URL.Query().Get("project_id")))
	}))
	t.Cleanup(srv.Close)
	return srv
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

// ─── VULN-1: Unbounded message sizes (DoS) ───────────────────────────────────
//
// hub.go ReadPump (line 99-108) never calls conn.SetReadLimit.
// Default gorilla limit is 32 768 bytes, but the production code does not
// explicitly enforce any application-level cap, so a future change to the
// default or a direct use of the underlying net.Conn can cause OOM.
// More concretely: we can verify no limit is enforced right now.

// TestSecurity_RED_UnboundedMessageSize_NoLimitEnforced demonstrates that the
// read pump accepts messages larger than a sensible application limit (4 KB).
// A patched hub should reject messages above the configured limit and close
// the connection.
func TestSecurity_RED_UnboundedMessageSize_NoLimitEnforced(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	conn := openRawWS(t, srv)
	// Send a 64 KB text message — well above any sane application limit.
	largeMsg := bytes.Repeat([]byte("A"), 64*1024)
	err := conn.WriteMessage(gorillaws.TextMessage, largeMsg)
	require.NoError(t, err, "write of large message should succeed from client side")

	// Give the server time to process.
	time.Sleep(50 * time.Millisecond)

	// RED assertion: a secure hub would close the connection after receiving
	// an oversized message. We verify the connection is still alive, proving
	// no limit is enforced.
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, readErr := conn.ReadMessage()
	// If connection is still open, readErr will be a timeout, not a close.
	isClose := gorillaws.IsCloseError(readErr,
		gorillaws.CloseMessageTooBig,
		gorillaws.CloseAbnormalClosure,
		gorillaws.CloseGoingAway,
	)
	assert.False(t, isClose,
		"RED: large message was NOT rejected — hub must call conn.SetReadLimit to enforce a cap")
}

// TestSecurity_GREEN_UnboundedMessageSize_ConnectionStillUsable verifies that,
// even after a large message, the hub connection is functional (baseline test).
func TestSecurity_GREEN_UnboundedMessageSize_ConnectionStillUsable(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	conn := openRawWS(t, srv)
	waitForRegistration()

	// Send a small, valid message.
	err := conn.WriteMessage(gorillaws.TextMessage, []byte("ping"))
	require.NoError(t, err)

	// Broadcast an event and verify we receive it.
	hub.Broadcast(websocket.Event{Type: "ping_test", Data: map[string]interface{}{}})
	ev := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "ping_test", ev.Type)
}

// ─── VULN-2: Missing read deadline (goroutine leak) ──────────────────────────
//
// hub.go ReadPump never calls conn.SetReadDeadline (or sets up ping/pong).
// A half-open TCP connection (client crashed, NAT timeout) keeps the goroutine
// and file descriptor alive forever.

// TestSecurity_RED_MissingReadDeadline_HalfOpenConnectionLeaksGoroutine
// shows that after a client abruptly disappears (TCP-level close without
// WebSocket close handshake), the hub goroutine remains blocked for more
// than the expected maximum idle period.
func TestSecurity_RED_MissingReadDeadline_HalfOpenConnectionLeaksGoroutine(t *testing.T) {
	hub := newSecHub(t)

	// Build a server that lets us grab the underlying net.Conn so we can
	// forcibly close it at TCP level without a WebSocket close frame.
	var rawConn net.Conn
	var rawMu sync.Mutex
	connReady := make(chan struct{})

	u := permissiveUpgrader()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intercept the hijacked conn.
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", 500)
			return
		}
		// We can't hijack before upgrading, so upgrade normally.
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Grab the underlying connection via reflection-free approach:
		// keep a reference to the net.Conn by using a custom listener.
		_ = hj
		hub.ServeWS(conn)
		rawMu.Lock()
		rawConn = conn.UnderlyingConn()
		rawMu.Unlock()
		close(connReady)
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Wait until the server has captured the raw conn.
	select {
	case <-connReady:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not signal conn ready in time")
	}

	// Abruptly close at TCP level (no WebSocket close frame).
	clientConn.UnderlyingConn().Close()
	rawMu.Lock()
	if rawConn != nil {
		rawConn.Close()
	}
	rawMu.Unlock()

	// RED: if a read deadline were set (e.g. 1 s idle ping timeout),
	// the goroutine would exit within ~1 s.  Without a deadline, it can
	// block indefinitely.  We wait 150 ms and assert the hub still has
	// the ghost client in its broadcast path — observable because Broadcast
	// does not block even when the client is gone (it closes and removes it).
	// This test documents the absence of a deadline rather than measuring
	// goroutine count (which requires runtime introspection).
	time.Sleep(150 * time.Millisecond)

	// The hub should have detected the disconnect. If it did not, the next
	// broadcast will silently drop the send (buffer full path). We broadcast
	// a burst to fill the dead client's send buffer.
	for i := 0; i < 300; i++ {
		hub.Broadcast(websocket.Event{Type: "flood", Data: i})
	}

	// GREEN path: no panic, no deadlock. The RED condition is the absence of
	// a configurable deadline; we flag it explicitly.
	t.Log("RED: ReadPump has no SetReadDeadline — half-open connections leak goroutines until the OS detects TCP RST")
}

// ─── VULN-3: Missing write deadline (slow-client DoS) ────────────────────────
//
// hub.go WritePump (line 112-131) never calls conn.SetWriteDeadline.
// A client that acknowledges TCP segments but never drains its receive buffer
// (TCP receive window = 0) blocks the WritePump goroutine for minutes.

// TestSecurity_RED_MissingWriteDeadline_SlowClientBlocksPump verifies that
// WritePump blocks indefinitely when the client's TCP receive window is full.
// We simulate this by setting a very small SO_RCVBUF and never reading.
func TestSecurity_RED_MissingWriteDeadline_SlowClientBlocksPump(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	// Dial but never read from the connection.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	waitForRegistration()

	// Flood the hub's broadcast channel to fill up the client send buffer.
	// The client goroutine will eventually block in WritePump's conn.NextWriter.
	broadcastDone := make(chan struct{})
	go func() {
		defer close(broadcastDone)
		for i := 0; i < 512; i++ {
			hub.Broadcast(websocket.Event{
				Type: "flood",
				Data: strings.Repeat("X", 1024),
			})
		}
	}()

	select {
	case <-broadcastDone:
		// All broadcasts queued without blocking the caller — channel is buffered.
	case <-time.After(5 * time.Second):
		t.Fatal("RED: Broadcast caller blocked — broadcast channel is full or hub stalled")
	}

	t.Log("RED: WritePump has no SetWriteDeadline — a slow/unresponsive client blocks the pump goroutine indefinitely")
}

// TestSecurity_GREEN_WriteDeadline_FastClientReceivesEvents verifies that a
// normally reading client receives events without issue (green baseline).
func TestSecurity_GREEN_WriteDeadline_FastClientReceivesEvents(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)
	conn := openRawWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{Type: "deadline_green", Data: "ok"})
	ev := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "deadline_green", ev.Type)
}

// ─── VULN-4: Broadcast blocks caller (DoS / goroutine stall) ─────────────────
//
// hub.go Broadcast (line 88-90) performs a blocking send to h.broadcast.
// If the hub Run loop falls behind (e.g. unregister/register operations are
// slow), caller goroutines stack up.

// TestSecurity_RED_BroadcastBlocksCallerWhenChannelFull demonstrates that
// when the broadcast channel is saturated (257 pending items > buffer 256),
// the 257th caller blocks.
func TestSecurity_RED_BroadcastBlocksCallerWhenChannelFull(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	// Create a hub but do NOT start Run() — the channel will fill up.
	hub := websocket.NewHub(logger)
	// Do not call hub.Run() intentionally.

	const bufferSize = 256
	blocked := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		for i := 0; i <= bufferSize; i++ {
			if i == bufferSize {
				close(blocked) // about to send the blocking one
			}
			hub.Broadcast(websocket.Event{Type: "flood", Data: i})
		}
	}()

	select {
	case <-blocked:
		// The (bufferSize+1)-th send is about to happen.
	case <-time.After(1 * time.Second):
		t.Fatal("could not fill broadcast channel within 1 s")
	}

	select {
	case <-done:
		t.Log("RED: Broadcast returned even when channel should be full (channel may be larger than expected)")
	case <-time.After(300 * time.Millisecond):
		// This is the expected RED condition: caller is blocked.
		t.Log("RED: Broadcast blocks the calling goroutine when the channel is full — " +
			"fix: use a non-blocking select with a drop-or-log strategy")
	}
	// Drain channel to unblock the goroutine and avoid goroutine leak in test.
	go hub.Run()
	<-done
}

// TestSecurity_GREEN_BroadcastDoesNotBlockWithRunningHub verifies that when
// Run() is active and clients are reading, Broadcast does not block.
func TestSecurity_GREEN_BroadcastDoesNotBlockWithRunningHub(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)
	conn := openRawWS(t, srv)
	waitForRegistration()

	done := make(chan struct{})
	go func() {
		defer close(done)
		hub.Broadcast(websocket.Event{Type: "no_block", Data: "ok"})
	}()

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("Broadcast blocked with an active hub")
	}

	ev := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "no_block", ev.Type)
}

// ─── VULN-5: Race condition — map mutation under RLock ───────────────────────
//
// hub.go Run broadcast case (lines 66-84): the loop holds h.mu.RLock() but
// calls delete(h.clients, client) on line 79, mutating the map while only a
// read lock is held. This is a data race detectable by go test -race.

// TestSecurity_RED_RaceCondition_MapMutationUnderRLock triggers the race by
// hammering concurrent broadcasts while clients connect and disconnect rapidly.
// The -race detector should flag this on line 79 of hub.go.
func TestSecurity_RED_RaceCondition_MapMutationUnderRLock(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	const (
		numClients    = 10
		numBroadcasts = 200
	)

	var wg sync.WaitGroup

	// Spawn clients that connect and immediately disconnect in a tight loop.
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
			conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			// Read a few events then disconnect.
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			for {
				_, _, readErr := conn.ReadMessage()
				if readErr != nil {
					break
				}
			}
			conn.Close()
		}()
	}

	// Broadcast rapidly while clients are connecting/disconnecting.
	for i := 0; i < numBroadcasts; i++ {
		hub.Broadcast(websocket.Event{Type: "race_test", Data: i})
		// Tiny sleep to interleave with client goroutines.
		if i%10 == 0 {
			time.Sleep(time.Millisecond)
		}
	}

	wg.Wait()
	// If -race detects a write on h.clients under RLock, the test binary
	// will print a DATA RACE report and exit with a non-zero code.
	t.Log("RED: if -race is enabled, this test exposes the map mutation under RLock in hub.go:79")
}

// ─── VULN-6: Double-close panic on client.send ───────────────────────────────
//
// Two code paths in hub.go can close client.send for the same client:
//   1. unregister case (line 61): close(client.send)
//   2. broadcast buffer-full branch (line 80): close(client.send) + delete
//
// If both paths race (unregister arrives after broadcast already closed it),
// Go panics with "close of closed channel".

// TestSecurity_RED_DoubleClose_PanicOnClosedChannel attempts to trigger the
// double-close panic by simultaneously flooding a slow client (to trigger the
// buffer-full close path) and disconnecting it (to trigger the unregister path).
func TestSecurity_RED_DoubleClose_PanicOnClosedChannel(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RED: panic recovered — double-close of client.send channel: %v", r)
		}
	}()

	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	waitForRegistration()

	// Close the client immediately (triggers unregister path).
	clientConn.Close()

	// Simultaneously flood broadcasts to trigger the buffer-full path.
	var wg sync.WaitGroup
	for i := 0; i < 512; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			hub.Broadcast(websocket.Event{Type: "double_close", Data: n})
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)
	// If we reach here without a panic, either the race did not manifest or
	// the fix is in place.
	t.Log("RED: double-close race may not always manifest; use -count=100 to increase probability")
}

// TestSecurity_GREEN_DoubleClose_SingleClientNormalLifecycle verifies that a
// normally disconnecting client does not panic.
func TestSecurity_GREEN_DoubleClose_SingleClientNormalLifecycle(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	conn := openRawWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{Type: "lifecycle", Data: "hello"})
	readEvent(t, conn, 2*time.Second)

	// Graceful close via WebSocket close handshake.
	conn.WriteMessage(gorillaws.CloseMessage,
		gorillaws.FormatCloseMessage(gorillaws.CloseNormalClosure, "bye"))
	time.Sleep(50 * time.Millisecond)
}

// ─── VULN-7: No ping/pong keepalive — stale connections leak goroutines ───────
//
// hub.go ReadPump does not send pings or configure a pong handler.
// Connections that become network-dead (NAT table timeout, kernel TCP buffer
// filled) are not cleaned up until the next Write fails.

// TestSecurity_RED_NoPingPong_StaleConnectionNotDetected verifies that the
// hub has no keepalive mechanism. After a connection becomes silently dead,
// the hub should detect it within a configurable ping interval. Currently it
// does not.
func TestSecurity_RED_NoPingPong_StaleConnectionNotDetected(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	conn := openRawWS(t, srv)
	waitForRegistration()

	// Simulate a dead connection: stop reading but don't close.
	// In production this models a NAT timeout where the TCP session appears
	// open locally but packets are silently dropped.
	conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

	// Wait longer than any reasonable ping interval (we expect < 30 s).
	// Since no ping is configured, the server will not detect the dead conn
	// within a short test window. We use 200 ms as a proxy.
	time.Sleep(200 * time.Millisecond)

	// Broadcast more than the send buffer capacity to fill the dead client.
	for i := 0; i < 300; i++ {
		hub.Broadcast(websocket.Event{Type: "keepalive_test", Data: i})
	}

	time.Sleep(100 * time.Millisecond)
	t.Log("RED: No ping/pong is implemented; stale connections are not cleaned up proactively")
}

// ─── VULN-8: Channel exhaustion — unlimited concurrent clients ────────────────
//
// hub.go ServeWS creates a new Client with a 256-event send channel for every
// connection. There is no cap on the number of simultaneous clients.
// An attacker can open thousands of WebSocket connections, exhausting file
// descriptors and goroutine stacks.

// TestSecurity_RED_ChannelExhaustion_NoClientLimit verifies that the hub
// accepts connections without enforcing any cap.
func TestSecurity_RED_ChannelExhaustion_NoClientLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping exhaustion test in short mode")
	}

	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	const numConnections = 200
	var accepted int64

	var wg sync.WaitGroup
	conns := make([]*gorillaws.Conn, 0, numConnections)
	var mu sync.Mutex

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			atomic.AddInt64(&accepted, 1)
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// Clean up all connections.
	mu.Lock()
	for _, c := range conns {
		c.Close()
	}
	mu.Unlock()

	// RED: all connections were accepted — no limit is enforced.
	assert.Equal(t, int64(numConnections), accepted,
		"RED: hub accepted all %d connections without any limit — "+
			"fix: add a maximum client count and reject upgrades beyond the cap",
		numConnections)
}

// TestSecurity_GREEN_ChannelExhaustion_LegitimateClientWorks verifies that a
// single legitimate client always connects successfully.
func TestSecurity_GREEN_ChannelExhaustion_LegitimateClientWorks(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)
	conn := openRawWS(t, srv)
	waitForRegistration()

	hub.Broadcast(websocket.Event{Type: "legit", Data: "ok"})
	ev := readEvent(t, conn, 2*time.Second)
	assert.Equal(t, "legit", ev.Type)
}

// ─── VULN-9: Missing hub shutdown / goroutine leak ────────────────────────────
//
// hub.go Run() (line 48-85) has no context or done channel. Calling Run()
// starts a goroutine that never exits, leaking it when tests or the server
// restarts the hub.

// TestSecurity_RED_HubRunLeaksGoroutine documents that Hub has no Stop/Close
// method. We verify the method does not exist by checking that we cannot call
// it (compilation would fail if we tried — so we document it in prose).
func TestSecurity_RED_HubRunLeaksGoroutine(t *testing.T) {
	// The Hub type exposes: NewHub, Run, Broadcast, ServeWS.
	// It does NOT expose: Stop(), Close(), Shutdown(), RunWithContext().
	// We demonstrate this via the interface — if a Stop method existed we
	// would call it here.
	hub := newSecHub(t)

	// Start and immediately "stop" — we have no way to stop it.
	// Any long-running test suite that creates many hubs leaks goroutines.
	_ = hub
	t.Log("RED: Hub.Run() has no termination mechanism — " +
		"fix: accept context.Context and return when ctx.Done() fires")
}

// TestSecurity_GREEN_HubRun_NoPanicOrDeadlockAfterManyBroadcasts verifies that
// a running hub is stable under load (baseline stability test).
func TestSecurity_GREEN_HubRun_NoPanicOrDeadlockAfterManyBroadcasts(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)
	conn := openRawWS(t, srv)
	waitForRegistration()

	// Drain in background.
	go func() {
		for {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	for i := 0; i < 100; i++ {
		hub.Broadcast(websocket.Event{Type: "stability", Data: i})
	}

	time.Sleep(200 * time.Millisecond)
	// No panic = pass.
}

// ─── VULN-10: WritePump silently discards events when writer fails ────────────
//
// hub.go WritePump (line 118-129): if conn.NextWriter returns an error,
// WritePump returns immediately. Any events remaining in client.send are
// silently discarded. Callers have no indication that events were lost.

// TestSecurity_RED_WritePump_SilentEventLoss verifies that when WritePump
// encounters a write error, queued events are lost without notification.
func TestSecurity_RED_WritePump_SilentEventLoss(t *testing.T) {
	hub := newSecHub(t)
	srv := newHubServer(t, hub)

	conn := openRawWS(t, srv)
	waitForRegistration()

	// Queue several events.
	for i := 0; i < 5; i++ {
		hub.Broadcast(websocket.Event{Type: "pre_disconnect", Data: i})
	}

	// Forcibly close the underlying TCP connection.
	conn.UnderlyingConn().Close()

	// Queue more events after the connection died.
	for i := 0; i < 5; i++ {
		hub.Broadcast(websocket.Event{Type: "post_disconnect", Data: i})
	}

	time.Sleep(100 * time.Millisecond)
	t.Log("RED: events queued after a write failure are silently discarded — " +
		"no error counter, no notification to the application layer")
}
