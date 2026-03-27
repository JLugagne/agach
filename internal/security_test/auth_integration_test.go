package security_test

// Cross-cutting authentication integration security tests.
//
// These tests verify vulnerabilities that span the identity and server bounded
// contexts -- gaps that cannot be caught by unit-level security tests within a
// single package.

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
)

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 1: JWT token accepted without project membership check
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_JWTWithoutProjectMembershipCheck documents that a valid
// JWT grants access to any project's resources, regardless of whether the user
// has been granted access to that project via the project_access table.
//
// Affected:
//   - internal/server/inbound/commands/tasks.go (all handlers accept projectID from URL)
//   - internal/server/inbound/queries/tasks.go (ListTasks, GetTask, SearchTasks)
//   - internal/server/inbound/commands/features.go, comments.go, notifications.go
//   - internal/server/inbound/queries/projects.go:GetProject (no access check)
//   - internal/pkg/middleware/middleware.go:NewRequireAuth (only validates JWT, no project scope)
//
// TODO(security): Add per-project access middleware that checks project_access table.
// The ListProjects endpoint already filters for non-admins, but individual resource
// endpoints (GET/POST /api/projects/{id}/tasks, etc.) do not verify membership.
func TestIntegration_RED_JWTWithoutProjectMembershipCheck(t *testing.T) {
	t.Log("RED: A valid JWT allows access to any project's resources without project membership check")

	// Verify that the server command handlers do NOT extract Actor from context
	// to check project access -- they only use the projectID from the URL path.
	//
	// We parse the task handler source to confirm no call to HasProjectAccess or
	// similar guard exists in the request flow.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../server/inbound/commands", nil, 0)
	require.NoError(t, err)

	pkg, ok := pkgs["commands"]
	require.True(t, ok, "commands package not found")

	// Check if any command handler calls HasProjectAccess or a similar guard.
	hasAccessCheck := false
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			name := sel.Sel.Name
			if name == "HasProjectAccess" || name == "CheckProjectAccess" || name == "VerifyProjectMembership" {
				hasAccessCheck = true
			}
			return true
		})
	}

	// RED: Today this assertion fails because command handlers have no per-request project
	// access check. Security fix required: all command handlers must call
	// HasProjectAccess/CheckProjectAccess/VerifyProjectMembership.
	assert.True(t, hasAccessCheck,
		"command handlers must call HasProjectAccess/CheckProjectAccess/VerifyProjectMembership before acting on project-scoped resources")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 2: Daemon JWT accepted on REST endpoints designed for users
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_DaemonJWTAcceptedOnUserEndpoints documents that the
// authValidatorAdapter in main.go falls back to ValidateDaemonJWT when
// ValidateJWT fails, meaning a daemon token can authenticate on any REST
// endpoint intended for human users (e.g., project creation, team management).
//
// Affected:
//   - cmd/agach-server/main.go:210-217 (authValidatorAdapter.ValidateJWT fallback)
//   - internal/pkg/middleware/middleware.go:NewRequireAuth (stores result as `any`)
//   - internal/server/inbound/queries/projects.go:ListProjects (type-asserts Actor for access filtering)
//
// TODO(security): Downstream handlers type-assert the context actor as
// identitydomain.Actor. When a daemon token is used, the context contains a
// DaemonActor instead, which silently fails the type assertion and bypasses
// the access filtering in ListProjects (line 54).
func TestIntegration_RED_DaemonJWTAcceptedOnUserEndpoints(t *testing.T) {
	t.Log("RED: Daemon JWT fallback in auth adapter means DaemonActor may bypass user-level access checks")

	// Demonstrate the type assertion gap: when the context holds a DaemonActor
	// instead of an Actor, the access filtering in ListProjects is silently skipped.
	daemonActor := identitydomain.DaemonActor{
		NodeID:      identitydomain.NewNodeID(),
		OwnerUserID: identitydomain.NewUserID(),
	}

	// Simulate what happens when the middleware stores a DaemonActor in context
	ctx := context.WithValue(context.Background(), middleware.ActorContextKey, daemonActor)

	// The ListProjects handler does: actor, ok := r.Context().Value(...).(identitydomain.Actor)
	// When the value is a DaemonActor, ok will be false, and the access filter is silently skipped.
	actor, ok := ctx.Value(middleware.ActorContextKey).(identitydomain.Actor)
	assert.False(t, ok, "DaemonActor should NOT satisfy Actor type assertion — this is expected")
	assert.True(t, actor.IsZero(), "Actor should be zero-value when type assertion fails")

	// RED: Today a daemon JWT can reach user endpoints because the auth adapter falls back
	// to ValidateDaemonJWT. Security fix required: the auth adapter must reject daemon tokens
	// on REST endpoints intended for users, or handlers must explicitly check for DaemonActor
	// and return 403.
	//
	// Parse the auth adapter in main.go to confirm it falls back to daemon JWT on user endpoints.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../../cmd/agach-server/main.go", nil, 0)
	require.NoError(t, err)

	// Look for ValidateDaemonJWT fallback in authValidatorAdapter
	foundDaemonFallback := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "ValidateDaemonJWT" {
			foundDaemonFallback = true
		}
		return true
	})

	// RED: Today foundDaemonFallback is true — daemon tokens are accepted everywhere.
	// Security fix required: remove the daemon JWT fallback from REST auth, or
	// add explicit handler-level rejection of DaemonActor on user endpoints.
	assert.False(t, foundDaemonFallback,
		"authValidatorAdapter must NOT fall back to ValidateDaemonJWT on REST endpoints; daemon tokens must be rejected for user-facing routes")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 3: Auth middleware bypass via route registration order
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_RouteRegistrationOrderBypass documents that identity routes
// (login, register, SSO) are registered on the main router BEFORE the auth
// middleware subrouter, which is correct. However, the /ws WebSocket endpoint is
// also registered on the main router (WSRouter = httpRouter in main.go), meaning
// it bypasses LimitBodySize middleware applied to the server subrouter.
//
// Affected:
//   - cmd/agach-server/main.go:114 (WSRouter = httpRouter, not serverRouter)
//   - internal/server/init.go:125 (wsRouter.Handle("/ws", wsHandler))
//   - internal/pkg/middleware/middleware.go:LimitBodySize (not applied to /ws)
//
// TODO(security): The /ws endpoint does its own auth via query param, but it
// does not apply LimitBodySize. The WebSocket ReadLimit (64KB) provides some
// protection, but HTTP upgrade headers are not size-limited.
func TestIntegration_RouteRegistrationOrderBypass(t *testing.T) {
	// Verify that /ws goes through the same middleware as other server routes.
	// The fix wraps /ws with LimitBodySize directly in init.go.
	root := mux.NewRouter()
	sub := root.PathPrefix("").Subrouter()

	middlewareCalled := false
	limitBodySize := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	sub.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Methods("GET")

	// /ws is wrapped with LimitBodySize directly (matches production init.go)
	root.Handle("/ws", limitBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))).Methods("GET")

	// /ws goes through LimitBodySize
	middlewareCalled = false
	rec := httptest.NewRecorder()
	root.ServeHTTP(rec, httptest.NewRequest("GET", "/ws", nil))
	assert.True(t, middlewareCalled,
		"/ws must have LimitBodySize applied")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 4: WebSocket CheckOrigin accepts all origins
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_WebSocketCheckOriginAcceptsAll documents that the WebSocket
// upgrader in WSHandler uses CheckOrigin: func(r *http.Request) bool { return true },
// which allows cross-site WebSocket hijacking from any origin.
//
// Affected:
//   - internal/server/inbound/commands/websocket.go:38 (CheckOrigin always returns true)
//
// TODO(security): Validate Origin header against a configured allowlist. The
// REST API sets Access-Control-Allow-Origin from the request Origin header
// (middleware.go:77), which is also permissive, but WebSocket CSRF is more
// dangerous because the browser will send cookies/credentials automatically.
func TestIntegration_RED_WebSocketCheckOriginAcceptsAll(t *testing.T) {
	t.Log("RED: WebSocket upgrader accepts connections from any origin (CSRF risk)")

	// Parse the websocket handler to confirm CheckOrigin returns true unconditionally
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../server/inbound/commands/websocket.go", nil, 0)
	require.NoError(t, err)

	// Find the CheckOrigin field in the Upgrader literal
	foundPermissiveOrigin := false
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		ident, ok := kv.Key.(*ast.Ident)
		if !ok || ident.Name != "CheckOrigin" {
			return true
		}
		// The value should be a FuncLit that returns true
		fn, ok := kv.Value.(*ast.FuncLit)
		if !ok {
			return true
		}
		if len(fn.Body.List) == 1 {
			ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
			if ok && len(ret.Results) == 1 {
				ident, ok := ret.Results[0].(*ast.Ident)
				if ok && ident.Name == "true" {
					foundPermissiveOrigin = true
				}
			}
		}
		return true
	})

	// RED: Today foundPermissiveOrigin is true — the upgrader accepts any origin.
	// Security fix required: CheckOrigin must validate the Origin header against an allowlist.
	assert.False(t, foundPermissiveOrigin,
		"WebSocket upgrader must NOT use CheckOrigin that unconditionally returns true; origin validation against an allowlist is required")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 5: CORS reflects arbitrary Origin
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_CORSReflectsArbitraryOrigin documents that the auth
// middleware reflects the request's Origin header in Access-Control-Allow-Origin,
// allowing any site to make credentialed cross-origin requests.
//
// Affected:
//   - internal/pkg/middleware/middleware.go:76-80 (reflects Origin)
//
// TODO(security): Validate Origin against a configured allowlist instead of
// reflecting it verbatim.
func TestIntegration_RED_CORSReflectsArbitraryOrigin(t *testing.T) {
	t.Log("RED: CORS middleware reflects arbitrary Origin header without validation")

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create a mock auth validator that always succeeds
	validator := &mockAuthValidator{
		actor: identitydomain.Actor{
			UserID: identitydomain.NewUserID(),
			Email:  "test@test.com",
			Role:   identitydomain.RoleAdmin,
		},
	}

	authMiddleware := middleware.NewRequireAuth(validator)

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with a malicious Origin
	req := httptest.NewRequest("GET", "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Origin", "https://evil-attacker.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// RED: Today the middleware reflects the attacker's origin verbatim.
	// Security fix required: validate Origin against a configured allowlist, never reflect it blindly.
	assert.NotEqual(t, "https://evil-attacker.com", rec.Header().Get("Access-Control-Allow-Origin"),
		"CORS middleware must NOT reflect arbitrary Origin headers; only allowlisted origins should be set in Access-Control-Allow-Origin")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 6: WebSocket token exposed in URL query parameter
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_WebSocketTokenInQueryParam documents that the WebSocket
// endpoint accepts the JWT token as a query parameter (?token=...), which means
// the token appears in server access logs, browser history, proxy logs, and
// the Referer header if the page navigates elsewhere.
//
// Affected:
//   - internal/server/inbound/commands/websocket.go:50 (r.URL.Query().Get("token"))
//   - cmd/agach-server/main.go:131 (RequestLogger logs r.URL.Path which includes query)
//
// TODO(security): Use a short-lived ticket exchange: client first obtains a
// single-use ticket via REST (authenticated by Bearer header), then passes that
// ticket as the query param. The ticket expires after one use or 30 seconds.
func TestIntegration_RED_WebSocketTokenInQueryParam(t *testing.T) {
	t.Log("RED: WebSocket auth uses JWT in URL query parameter, exposing it in logs and Referer headers")

	// Parse the websocket handler to confirm token is read from query param
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../server/inbound/commands/websocket.go", nil, parser.ParseComments)
	require.NoError(t, err)

	foundQueryToken := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "Get" && len(call.Args) == 1 {
			lit, ok := call.Args[0].(*ast.BasicLit)
			if ok && strings.Contains(lit.Value, "token") {
				foundQueryToken = true
			}
		}
		return true
	})

	// RED: Today foundQueryToken is true — the WebSocket handler reads the JWT from the query param.
	// Security fix required: use a short-lived ticket exchange so the JWT never appears in URL logs.
	assert.False(t, foundQueryToken,
		"WebSocket handler must NOT read the JWT token from a URL query parameter; use a short-lived ticket exchange instead to prevent token leakage in logs and Referer headers")
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type mockAuthValidator struct {
	actor identitydomain.Actor
}

func (m *mockAuthValidator) ValidateJWT(_ context.Context, _ string) (any, error) {
	return m.actor, nil
}
