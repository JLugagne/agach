package security_test

// Cross-project data isolation security tests.
//
// These tests verify that data from one project cannot leak to users of another
// project through WebSocket broadcasts, direct resource access,
// or notification queries.

import (
	"encoding/json"
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

	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
)

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 1: WebSocket broadcast leaks to clients without projectID
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_WebSocketBroadcastLeaksToUnfilteredClients documents that
// WebSocket clients connected without a projectID receive ALL broadcast events,
// including those scoped to projects they should not have access to.
//
// Affected:
//   - internal/pkg/websocket/hub.go:119-121 (filter condition: both must be non-empty)
//   - internal/server/inbound/commands/websocket.go:72-79 (browser clients get no WithProjectID)
//
// The broadcast filter is:
//
//	if client.projectID != "" && event.ProjectID != "" && client.projectID != event.ProjectID {
//	    continue
//	}
//
// When client.projectID is empty (browser clients), the condition short-circuits
// and the event is delivered regardless of which project it belongs to.
//
// TODO(security): Browser clients should be required to specify a project ID,
// or the hub should only deliver events to clients that have proven access to
// that project.
func TestIntegration_RED_WebSocketBroadcastLeaksToUnfilteredClients(t *testing.T) {
	t.Log("RED: WebSocket clients without projectID receive events from all projects")

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
		var opts []websocket.ServeWSOption
		if pid := r.URL.Query().Get("project_id"); pid != "" {
			opts = append(opts, websocket.WithProjectID(pid))
		}
		hub.ServeWS(conn, opts...)
	}))
	defer srv.Close()

	// Connect a client WITHOUT project_id (simulates browser client from websocket.go)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	unfilteredConn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer unfilteredConn.Close()

	// Connect a client scoped to project-A
	scopedConn, _, err := gorillaws.DefaultDialer.Dial(wsURL+"?project_id=project-A", nil)
	require.NoError(t, err)
	defer scopedConn.Close()

	// Allow connections to register
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event for project-B (neither client should see this if isolation is correct)
	hub.Broadcast(websocket.Event{
		Type:      "task_created",
		ProjectID: "project-B",
		Data:      map[string]string{"task_id": "secret-task"},
	})

	// The scoped client (project-A) should NOT receive project-B's event
	scopedConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, scopedErr := scopedConn.ReadMessage()
	assert.Error(t, scopedErr, "scoped client should not receive cross-project event")

	// The unfiltered client should NOT receive project-B's event if project isolation is enforced.
	// RED: Today this assertion fails because the hub leaks cross-project events to
	// clients without a projectID.
	unfilteredConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, unfilteredErr := unfilteredConn.ReadMessage()
	assert.Error(t, unfilteredErr,
		"unfiltered client should NOT receive cross-project event (security fix required: clients without projectID must not receive project-scoped events)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 2: IDOR on project-scoped resources (task, feature, comment IDs)
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_IDORProjectScopedResources documents that project-scoped
// endpoints accept any valid UUID as a project ID in the URL path, and the
// handlers directly cast the URL parameter to a domain ID type without verifying
// that the caller has access to that project.
//
// Affected:
//   - internal/server/inbound/commands/tasks.go:56 (projectID := domain.ProjectID(mux.Vars(r)["id"]))
//   - internal/server/inbound/commands/features.go (same pattern)
//   - internal/server/inbound/commands/comments.go (same pattern)
//   - internal/server/inbound/queries/tasks.go:51,81,157 (same pattern)
//   - internal/server/inbound/queries/notifications.go:76 (same pattern)
//
// TODO(security): All project-scoped handlers should verify that the
// authenticated user (from context) has access to the project before proceeding.
func TestIntegration_RED_IDORProjectScopedResources(t *testing.T) {
	t.Log("RED: Project-scoped handlers cast URL project ID directly without access verification")

	// Parse multiple handler files to confirm the pattern: domain.ProjectID(mux.Vars(r)["id"])
	// without a subsequent access check
	handlerFiles := []string{
		"../server/inbound/commands/tasks.go",
		"../server/inbound/commands/features.go",
		"../server/inbound/commands/comments.go",
		"../server/inbound/queries/tasks.go",
	}

	for _, filePath := range handlerFiles {
		t.Run(filePath, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filePath, nil, 0)
			require.NoError(t, err)

			// Count how many times projectID is created from mux.Vars
			projectIDCasts := 0
			accessChecks := 0

			ast.Inspect(f, func(n ast.Node) bool {
				// Look for domain.ProjectID( ... )
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				name := sel.Sel.Name
				if name == "ProjectID" {
					projectIDCasts++
				}
				if strings.Contains(name, "HasAccess") || strings.Contains(name, "CheckAccess") || strings.Contains(name, "VerifyAccess") {
					accessChecks++
				}
				return true
			})

			assert.Greater(t, projectIDCasts, 0,
				"handler should use ProjectID from URL (confirms it handles project-scoped resources)")
			// RED: Today this assertion fails because no handler calls HasAccess/CheckAccess/VerifyAccess.
			// Security fix required: all project-scoped handlers must verify project membership.
			assert.Greater(t, accessChecks, 0,
				"project-scoped handlers must call HasAccess/CheckAccess/VerifyAccess to verify project membership")
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 4: Notification queries return cross-project data
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_NotificationsNoCrossProjectFilter documents that the
// /api/notifications endpoint returns ALL notifications across all projects
// without filtering by the caller's project access.
//
// Affected:
//   - internal/server/inbound/queries/notifications.go:57-71 (ListAllNotifications: no project filter)
//   - internal/server/app/notifications.go (ListNotifications passes nil projectID)
//
// TODO(security): The global notification endpoint should filter results to
// only projects the user has access to, similar to how ListProjects filters.
func TestIntegration_RED_NotificationsNoCrossProjectFilter(t *testing.T) {
	t.Log("RED: /api/notifications returns notifications from all projects without access filtering")

	// Parse the notification queries handler to confirm ListAllNotifications
	// passes nil as projectID without user-based filtering
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../server/inbound/queries/notifications.go", nil, 0)
	require.NoError(t, err)

	// Find ListAllNotifications function
	foundNilProjectID := false
	ast.Inspect(f, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "ListAllNotifications" {
			return true
		}

		// Inside this function, look for ListNotifications(ctx, nil, ...)
		ast.Inspect(funcDecl.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name == "ListNotifications" && len(call.Args) >= 2 {
				// Second arg should be nil (no project filter)
				ident, ok := call.Args[1].(*ast.Ident)
				if ok && ident.Name == "nil" {
					foundNilProjectID = true
				}
			}
			return true
		})
		return true
	})

	// RED: Today this assertion fails because ListAllNotifications passes nil as projectID.
	// Security fix required: the endpoint must filter by the user's accessible projects.
	assert.False(t, foundNilProjectID,
		"ListAllNotifications must NOT pass nil projectID; it must filter notifications to the user's accessible projects")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 5: WebSocket relay sends daemon messages to all non-daemon clients
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_WebSocketRelayNoCrossProjectFilter documents that the
// relay mechanism in the WebSocket hub forwards daemon messages to ALL non-daemon
// clients, regardless of project scope. A daemon working on project-A will have
// its messages relayed to browser clients connected for project-B.
//
// Affected:
//   - internal/pkg/websocket/hub.go:133-149 (relay: no project filtering)
//   - internal/server/init.go:78-96 (all daemon message types use relay handler)
//
// TODO(security): Relay should check client.projectID against the message's
// project context before forwarding.
func TestIntegration_RED_WebSocketRelayNoCrossProjectFilter(t *testing.T) {
	t.Log("RED: WebSocket relay forwards daemon messages to all browser clients without project filtering")

	// Parse the hub source to confirm relay does not filter by project
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../pkg/websocket/hub.go", nil, 0)
	require.NoError(t, err)

	// In the relay case, look for project filtering
	hasRelayProjectFilter := false
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "projectID" {
			// Check if this is inside the relay case by looking for isDaemon nearby
			// (heuristic: if projectID is referenced at all in the relay path, filtering exists)
			hasRelayProjectFilter = true
		}
		return true
	})

	// The relay section uses isDaemon but never checks projectID
	// We need a more targeted check: look in the relay case specifically
	// For now, verify functionally with an actual hub test
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
		var opts []websocket.ServeWSOption
		if r.URL.Query().Get("daemon") == "true" {
			opts = append(opts, websocket.AsDaemon(), websocket.WithNodeID("node-1"))
		}
		if pid := r.URL.Query().Get("project_id"); pid != "" {
			opts = append(opts, websocket.WithProjectID(pid))
		}
		hub.ServeWS(conn, opts...)
	}))
	defer srv.Close()

	// Register a relay handler for a test message type
	hub.RegisterHandler("test_relay", hub.NewRelayHandler())

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	// Connect daemon
	daemonConn, _, err := gorillaws.DefaultDialer.Dial(wsURL+"/?daemon=true", nil)
	require.NoError(t, err)
	defer daemonConn.Close()

	// Connect browser client for project-A
	browserA, _, err := gorillaws.DefaultDialer.Dial(wsURL+"/?project_id=project-A", nil)
	require.NoError(t, err)
	defer browserA.Close()

	// Connect browser client for project-B
	browserB, _, err := gorillaws.DefaultDialer.Dial(wsURL+"/?project_id=project-B", nil)
	require.NoError(t, err)
	defer browserB.Close()

	time.Sleep(50 * time.Millisecond)

	// Daemon sends a message (e.g., build event for project-A)
	msg := json.RawMessage(`{"type":"test_relay","project_id":"project-A","data":"sensitive"}`)
	err = daemonConn.WriteMessage(gorillaws.TextMessage, msg)
	require.NoError(t, err)

	// Both browser clients receive it (relay does NOT filter by project)
	browserA.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, msgA, errA := browserA.ReadMessage()

	browserB.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, msgB, errB := browserB.ReadMessage()

	assert.NoError(t, errA, "browser-A should receive relayed message for its own project")
	// RED: Today this assertion fails because the relay forwards messages to all non-daemon clients.
	// Security fix required: relay must check client projectID against the message's project context.
	assert.Error(t, errB,
		"browser-B should NOT receive a relay message scoped to project-A (relay must filter by project)")

	// Suppress "projectID referenced" since it's in the broadcast path, not relay
	_ = hasRelayProjectFilter
	_ = msgA
	_ = msgB
}
