package security_test

// Resource exhaustion (DoS) security tests.
//
// These tests verify denial-of-service vectors that span multiple components
// or exploit architectural gaps between bounded contexts.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gorillaws "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if !hub.CanAcceptIP(ip) {
			http.Error(w, "too many connections", http.StatusTooManyRequests)
			return
		}
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

	// RED: Today this assertion fails because the hub has no per-user connection limit.
	// Security fix required: the hub must enforce a per-user/per-IP limit and reject excess connections.
	assert.Less(t, len(conns), concurrentConns,
		"hub must reject excess connections from a single user/IP (per-user connection limit required)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 2: No limit on dependency graph depth
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

	// RED: Today this assertion fails because no depth limit exists in the app layer.
	// Security fix required: enforce a maximum dependency chain depth (e.g., maxDepth constant).
	assert.True(t, hasDepthCheck,
		"app layer must enforce a maximum dependency chain depth (maxDepth/max_depth/depthLimit constant required)")
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

	// RED: Today this assertion fails because server init.go does not apply rate limiting.
	// Security fix required: resource creation endpoints must be rate limited.
	assert.True(t, hasRateLimit,
		"server init.go must apply RateLimit/RateLimitMiddleware to resource creation endpoints")

	// Also verify resource count limits exist in the task handler
	f2, err := parser.ParseFile(fset, "../server/inbound/commands/tasks.go", nil, 0)
	require.NoError(t, err)

	hasCountLimit := false
	ast.Inspect(f2, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		lower := strings.ToLower(ident.Name)
		if strings.Contains(lower, "maxcount") || strings.Contains(lower, "maxtasks") || strings.Contains(lower, "ratelimit") {
			hasCountLimit = true
		}
		return true
	})

	// RED: Today this also fails because no resource count limits exist in the task handler.
	assert.True(t, hasCountLimit,
		"task handler must have resource count limits (maxCount/maxTasks/rateLimit reference required)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 5: WebSocket broadcast channel can be filled by rapid events
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

	// Fill the broadcast channel beyond its buffer size
	dropped := 0
	const totalEvents = 300
	for i := 0; i < totalEvents; i++ {
		hub.Broadcast(websocket.Event{
			Type:      "task_created",
			ProjectID: "project-1",
			Data:      map[string]string{"i": string(rune(i))},
		})
	}

	// RED: Today Broadcast() silently drops events beyond the buffer size (select/default).
	// Security fix required: Broadcast must signal backpressure — either return an error,
	// increment a counter, or apply flow control. Silent drops are unacceptable.
	//
	// Verify by parsing the source for a backpressure mechanism.
	fset2 := token.NewFileSet()
	f2, err := parser.ParseFile(fset2, "../pkg/websocket/hub.go", nil, 0)
	require.NoError(t, err)

	hasBackpressure := false
	ast.Inspect(f2, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		lower := strings.ToLower(ident.Name)
		if strings.Contains(lower, "dropped") || strings.Contains(lower, "backpressure") || strings.Contains(lower, "overflow") {
			hasBackpressure = true
		}
		return true
	})

	_ = dropped
	assert.True(t, hasBackpressure,
		"WebSocket hub must implement backpressure on broadcast channel saturation (dropped/backpressure/overflow counter or error return required)")
}
