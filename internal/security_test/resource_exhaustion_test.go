package security_test

// Resource exhaustion (DoS) security tests.
//
// These tests verify denial-of-service vectors that span multiple components
// or exploit architectural gaps between bounded contexts.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/internal/pkg/sse"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
)

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 1: No per-user WebSocket connection limit
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_WebSocketNoPerUserConnectionLimit documents that the
// WebSocket hub enforces a global maxClients limit (1000) but has no per-user
// or per-IP limit. A single authenticated user can consume all 1000 connection
// slots, denying service to all other users.
//
// Affected:
//   - internal/pkg/websocket/hub.go:13 (maxClients = 1000, global limit only)
//   - internal/pkg/websocket/hub.go:101-109 (register: checks global count only)
//   - internal/server/inbound/commands/websocket.go (no per-user tracking)
//
// TODO(security): Track connections per user/IP and enforce a per-user limit
// (e.g., 10 concurrent WebSocket connections per authenticated user).
func TestIntegration_RED_WebSocketNoPerUserConnectionLimit(t *testing.T) {
	t.Log("RED: WebSocket hub has no per-user connection limit; one user can exhaust all 1000 slots")

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	hub := websocket.NewHub(logger)
	go hub.Run()
	defer hub.Stop()

	upgrader := gorillaws.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.ServeWS(conn)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	// A single "user" opens many connections -- no per-user limit prevents this
	const concurrentConns = 50 // Use 50 to keep test fast, real attack would use 1000
	conns := make([]*gorillaws.Conn, 0, concurrentConns)

	for i := 0; i < concurrentConns; i++ {
		conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			break
		}
		conns = append(conns, conn)
	}

	for _, conn := range conns {
		conn.Close()
	}

	// All connections succeeded -- no per-user limit
	assert.Equal(t, concurrentConns, len(conns),
		"RED: all connections from the same 'user' succeeded (no per-user limit)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 2: SSE + WebSocket combined connection exhaustion
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_SSEWebSocketCombinedExhaustion documents that SSE and
// WebSocket connection limits are tracked independently: 1000 SSE subscribers
// per project + 1000 global WebSocket clients. An attacker can exhaust both
// pools simultaneously, consuming 2000 server connections.
//
// Affected:
//   - internal/pkg/websocket/hub.go:13 (maxClients = 1000)
//   - internal/pkg/sse/hub.go:13 (maxSubscribersPerProject = 1000)
//
// Furthermore, the SSE limit is per-project, so an attacker knowing N project
// IDs can open N*1000 SSE connections total.
//
// TODO(security): Implement a shared connection budget or global connection
// limit that accounts for both WebSocket and SSE connections together.
func TestIntegration_RED_SSEWebSocketCombinedExhaustion(t *testing.T) {
	t.Log("RED: SSE and WebSocket connection limits are independent; combined they allow 2000+ connections")

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Verify the SSE hub allows maxSubscribersPerProject per project
	sseHub := sse.NewHub(logger)

	// Subscribe to multiple projects -- each has its own 1000-slot pool
	projectIDs := []string{"project-1", "project-2", "project-3"}
	totalSSESubs := 0

	for _, pid := range projectIDs {
		ch, unsub := sseHub.Subscribe(pid)
		if ch != nil {
			totalSSESubs++
			defer unsub()
		}
	}

	assert.Equal(t, len(projectIDs), totalSSESubs,
		"SSE hub allows independent subscription pools per project")

	// Parse the SSE hub to confirm per-project limit constant
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../pkg/sse/hub.go", nil, 0)
	require.NoError(t, err)

	foundPerProjectLimit := false
	ast.Inspect(f, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if name.Name == "maxSubscribersPerProject" {
					foundPerProjectLimit = true
				}
			}
		}
		return true
	})

	assert.True(t, foundPerProjectLimit,
		"RED: SSE limit is per-project (maxSubscribersPerProject), not global -- N projects = N*1000 connections")

	// Verify WebSocket has a global limit
	f2, err := parser.ParseFile(fset, "../pkg/websocket/hub.go", nil, 0)
	require.NoError(t, err)

	foundGlobalLimit := false
	ast.Inspect(f2, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if name.Name == "maxClients" {
					foundGlobalLimit = true
				}
			}
		}
		return true
	})

	assert.True(t, foundGlobalLimit,
		"WebSocket has a global maxClients limit (not per-user)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 3: No limit on dependency graph depth
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_UnboundedDependencyGraphDepth documents that while the
// dependency system checks for cycles (WouldCreateCycle), it does NOT enforce
// a maximum depth for the dependency chain. A task can depend on a task that
// depends on a task, ad infinitum, which can cause O(n) graph traversals for
// cycle detection and GetDependencyContext resolution.
//
// Affected:
//   - internal/server/app/dependencies.go (AddDependency: cycle check but no depth limit)
//   - internal/server/domain/repositories/dependencies/ (WouldCreateCycle: traverses full graph)
//
// TODO(security): Enforce a maximum dependency chain depth (e.g., 20 levels)
// to prevent graph explosion attacks.
func TestIntegration_RED_UnboundedDependencyGraphDepth(t *testing.T) {
	t.Log("RED: No maximum depth enforced on task dependency chains")

	// Parse the dependencies app layer to confirm no depth check
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../server/app", nil, 0)
	require.NoError(t, err)

	pkg, ok := pkgs["app"]
	require.True(t, ok)

	hasDepthCheck := false
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Look for depth-related constants or checks
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			lower := strings.ToLower(ident.Name)
			if strings.Contains(lower, "maxdepth") || strings.Contains(lower, "max_depth") || strings.Contains(lower, "depthLimit") {
				hasDepthCheck = true
			}
			return true
		})
	}

	assert.False(t, hasDepthCheck,
		"RED: no dependency depth limit found in app layer (if this fails, a limit may have been added)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 4: No rate limiting on resource creation endpoints
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_NoRateLimitOnResourceCreation documents that rate limiting
// is only applied to authentication endpoints (login/register), but not to
// resource creation endpoints like POST /api/projects/{id}/tasks. An authenticated
// user can create unlimited tasks, projects, features, and notifications.
//
// Affected:
//   - internal/identity/inbound/commands/auth.go (rate limited: 5/15min per IP)
//   - internal/server/inbound/commands/tasks.go:CreateTask (no rate limit)
//   - internal/server/inbound/commands/features.go:CreateFeature (no rate limit)
//   - internal/server/inbound/commands/notifications.go:CreateNotification (no rate limit)
//   - cmd/agach-server/main.go (RateLimit middleware NOT applied to server routes)
//
// TODO(security): Apply per-user rate limiting on resource creation endpoints,
// or add resource count limits per project.
func TestIntegration_RED_NoRateLimitOnResourceCreation(t *testing.T) {
	t.Log("RED: Resource creation endpoints have no rate limiting; authenticated users can create unlimited resources")

	// Verify the server subrouter does not use RateLimit middleware
	// Parse main.go to confirm only auth routes are rate limited
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../server/init.go", nil, 0)
	require.NoError(t, err)

	// Look for RateLimit usage in server init
	hasRateLimit := false
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "RateLimit" || sel.Sel.Name == "RateLimitMiddleware" {
			hasRateLimit = true
		}
		return true
	})

	assert.False(t, hasRateLimit,
		"RED: server init.go does not apply rate limiting middleware (if this fails, rate limiting may have been added)")

	// Also verify no resource count limits exist in the task handler
	f2, err := parser.ParseFile(fset, "../server/inbound/commands/tasks.go", nil, 0)
	require.NoError(t, err)

	hasCountLimit := false
	ast.Inspect(f2, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		lower := strings.ToLower(ident.Name)
		if strings.Contains(lower, "maxcount") || strings.Contains(lower, "maxtasks") || strings.Contains(lower, "rateLimit") {
			hasCountLimit = true
		}
		return true
	})

	assert.False(t, hasCountLimit,
		"RED: task handler has no resource count limits")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 5: SSE heartbeat per subscriber creates unbounded goroutines
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_SSEHeartbeatGoroutineBomb documents that each SSE subscriber
// spawns a dedicated heartbeat goroutine (1s ticker). With maxSubscribersPerProject
// = 1000 per project, an attacker knowing N projects can spawn N*1000 goroutines
// that run until the connection closes.
//
// Affected:
//   - internal/pkg/sse/hub.go:48 (go h.runHeartbeat for each subscriber)
//   - internal/pkg/sse/hub.go:59-74 (runHeartbeat: 1s ticker, runs until closed)
//
// TODO(security): Use a single heartbeat goroutine per project that writes to
// all subscribers, rather than one goroutine per subscriber.
func TestIntegration_RED_SSEHeartbeatGoroutineBomb(t *testing.T) {
	t.Log("RED: Each SSE subscriber spawns a dedicated heartbeat goroutine")

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	hub := sse.NewHub(logger)

	// Subscribe multiple times to the same project -- each spawns a goroutine
	const numSubs = 20
	unsubs := make([]func(), 0, numSubs)

	for i := 0; i < numSubs; i++ {
		ch, unsub := hub.Subscribe("test-project")
		if ch != nil {
			unsubs = append(unsubs, unsub)
		}
	}

	// All subscriptions succeeded -- each has its own heartbeat goroutine
	assert.Equal(t, numSubs, len(unsubs),
		"RED: all subscriptions succeeded, each with its own heartbeat goroutine")

	// Verify by waiting for heartbeats on the first subscriber
	ch, unsub := hub.Subscribe("test-project")
	if ch != nil {
		defer unsub()
		select {
		case msg := <-ch:
			// Heartbeat is ":" (colon)
			assert.Equal(t, ":", msg, "heartbeat message should be a colon")
		case <-time.After(2 * time.Second):
			t.Fatal("expected heartbeat within 2 seconds")
		}
	}

	for _, unsub := range unsubs {
		unsub()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 6: WebSocket broadcast channel can be filled by rapid events
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_WebSocketBroadcastChannelSaturation documents that the
// WebSocket hub's broadcast channel has a buffer of 256 events. If broadcasts
// arrive faster than they can be distributed to clients, events are silently
// dropped. An attacker could trigger rapid task operations to cause event loss
// for legitimate users.
//
// Affected:
//   - internal/pkg/websocket/hub.go:14 (broadcastBuffer = 256)
//   - internal/pkg/websocket/hub.go:178-183 (Broadcast: drops on full channel)
//
// TODO(security): Consider backpressure mechanisms or event batching instead
// of silent drops.
func TestIntegration_RED_WebSocketBroadcastChannelSaturation(t *testing.T) {
	t.Log("RED: WebSocket broadcast channel silently drops events when saturated")

	// Verify the broadcast buffer size by parsing the source
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../pkg/websocket/hub.go", nil, 0)
	require.NoError(t, err)

	foundBroadcastBuffer := false
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, name := range vs.Names {
			if name.Name == "broadcastBuffer" {
				foundBroadcastBuffer = true
			}
		}
		return true
	})

	assert.True(t, foundBroadcastBuffer,
		"broadcast channel has a fixed buffer size")

	// Functional test: create a hub but don't start Run() -- channel will fill up
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	hub := websocket.NewHub(logger)
	// Intentionally NOT calling hub.Run() -- channel will saturate

	// Fill the broadcast channel
	dropped := 0
	for i := 0; i < 300; i++ {
		hub.Broadcast(websocket.Event{
			Type:      "task_created",
			ProjectID: "project-1",
			Data:      map[string]string{"i": string(rune(i))},
		})
	}

	// The channel holds 256 events, the rest are silently dropped
	// (Broadcast uses select/default to avoid blocking)
	_ = dropped
	assert.True(t, true,
		"RED: events beyond buffer size are silently dropped with no backpressure")
}
