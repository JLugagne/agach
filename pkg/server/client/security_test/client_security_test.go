package security_test

// Security tests for pkg/server/client/client.go
//
// Each vulnerability section contains:
//   - RED test  : demonstrates the missing protection (currently the code behaves
//                 unsafely — the test documents that fact)
//   - GREEN test: the correct safe behaviour that SHOULD be enforced
//
// Note on RED tests: because Go panics/compile errors are not the right way to
// express "this should be rejected at the API boundary", RED tests are written
// as assertions that describe the CURRENT (broken) behaviour and are expected to
// pass as-is.  Comments explain what the correct behaviour should be.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/pkg/server/client"
)

// --- inlined helpers from helpers_test.go ---

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("writeJSON: %v", err)
	}
}

func successResponse(data any) map[string]any {
	return map[string]any{
		"status": "success",
		"data":   data,
	}
}

// decodeJSONBody is a helper used by security tests.
func decodeJSONBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// ─── VULNERABILITY 1 ────────────────────────────────────────────────────────
// SSRF — client.New accepts any URL scheme, including file://, ftp://, or
// internal network addresses.  There is no scheme or host validation.
//
// File: pkg/server/client/client.go lines 21-26
//
// An attacker who controls the baseURL (e.g., via a config file, environment
// variable, or MCP tool input) can make the agent issue requests to internal
// services, local files, or metadata endpoints (e.g., http://169.254.169.254).

func TestSecurity_RED_SSRF_ArbitrarySchemeAccepted(t *testing.T) {
	// RED: New() accepts a file:// URL without error.
	// This should be rejected; instead the constructor returns a client silently.
	c := client.New("file:///etc/passwd")
	assert.NotNil(t, c, "RED: New() should reject file:// scheme but currently accepts it silently")
}

func TestSecurity_RED_SSRF_InternalMetadataEndpointAccepted(t *testing.T) {
	// RED: New() accepts the AWS EC2 metadata endpoint without error.
	c := client.New("http://169.254.169.254")
	assert.NotNil(t, c, "RED: New() should reject internal/link-local URLs but currently accepts them")
}

func TestSecurity_GREEN_SSRF_ValidHTTPURLAccepted(t *testing.T) {
	// GREEN: a normal http:// URL to a known host must be accepted.
	c := client.New("http://localhost:8080")
	require.NotNil(t, c, "GREEN: a valid http:// URL must be accepted by New()")
}

func TestSecurity_GREEN_SSRF_ValidHTTPSURLAccepted(t *testing.T) {
	// GREEN: an https:// URL must be accepted.
	c := client.New("https://server.example.com")
	require.NotNil(t, c, "GREEN: a valid https:// URL must be accepted by New()")
}

// ─── VULNERABILITY 2 ────────────────────────────────────────────────────────
// Missing TLS enforcement — the default http.Client uses the system transport
// with no TLS-specific configuration, and New() does not reject plain-HTTP
// URLs.  Credentials and task data sent over HTTP are visible to
// man-in-the-middle observers on the same network.
//
// File: pkg/server/client/client.go lines 21-26
//
// Additionally, there is no mechanism to configure a custom CA or to enforce
// minimum TLS version.

func TestSecurity_RED_NoTLSEnforcement_PlainHTTPAllowed(t *testing.T) {
	// RED: the client willingly talks over plaintext HTTP — observed by
	// checking that a request to a plain HTTP test server succeeds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL) // srv.URL is http://
	_, err := c.ListProjects()
	assert.NoError(t, err, "RED: the client should enforce HTTPS for production use but currently allows plaintext HTTP")
}

func TestSecurity_GREEN_TLS_DefaultClientTrustsPublicCerts(t *testing.T) {
	// GREEN: the default http.Client does perform certificate verification
	// (it just cannot be customised via client.New).  We verify this by
	// confirming that a TLS server whose certificate the client does not
	// trust is rejected with a certificate-related error.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.ListProjects()
	// The test server uses a self-signed cert that the default transport rejects.
	require.Error(t, err, "GREEN: certificate verification must reject self-signed certs")
	assert.Contains(t, err.Error(), "certificate",
		"GREEN: the error must mention certificate verification, confirming cert checks are active")
}

// ─── VULNERABILITY 3 ────────────────────────────────────────────────────────
// Path injection via unescaped ID parameters — methods such as GetProject,
// GetColumns, ListProjectRoles, etc., concatenate caller-supplied strings
// directly into URL paths without url.PathEscape.
//
// File: pkg/server/client/client.go lines 101-111 (GetProject), 127-133
// (ListProjectRoles), 364-370 (GetColumns), etc.
//
// A malicious ID like "../../admin" or "proj-1/../../other-route" can reach
// unintended endpoints.  The Go http.Client normalises "/../" sequences in
// the path, so the server receives a traversed path rather than an error.

func TestSecurity_RED_PathInjection_GetColumnsTraversal(t *testing.T) {
	// RED: "proj/../../admin" is concatenated into the path without escaping.
	// The server receives the raw dotted path "/api/projects/proj/../../admin/columns"
	// because url.PathEscape is not used.  A server that resolves "../.." would
	// route the request to the wrong endpoint entirely.
	// The ID should be percent-encoded so the server receives
	// "/api/projects/proj%2F..%2F..%2Fadmin/columns" (a single path segment).
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, _ = c.GetColumns("proj/../../admin")

	// The server receives the raw un-encoded path — this IS the vulnerability.
	// url.PathEscape should have been applied so the slashes and dots become
	// "%2F" and "%2E%2E" respectively.
	assert.Equal(t, "/api/projects/proj/../../admin/columns", receivedPath,
		"RED: path traversal sequences in the ID are NOT percent-encoded — url.PathEscape must be used")
	// After safe encoding the server would receive a path that still starts with
	// /api/projects/ and contains the encoded segment, not a traversal.
	assert.NotContains(t, receivedPath, "%2F",
		"RED: the path contains literal slashes, not percent-encoded ones")
}

func TestSecurity_GREEN_PathInjection_SafeIDIsPassedThrough(t *testing.T) {
	// GREEN: a normal UUID-style ID reaches the correct endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/projects/abc-123/columns", r.URL.Path)
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.GetColumns("abc-123")
	require.NoError(t, err, "GREEN: a safe ID must reach the correct path")
}

func TestSecurity_RED_PathInjection_GetProjectTraversal(t *testing.T) {
	// RED: projectID with embedded slashes is concatenated unsafely.
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		writeJSON(t, w, http.StatusOK, successResponse(map[string]any{
			"id": "x", "name": "x", "description": "",
			"created_by_role": "", "created_by_agent": "", "default_role": "",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, _ = c.GetProject("proj/extra-segment")

	// The path received by the server will be "/api/projects/proj/extra-segment"
	// instead of "/api/projects/proj%2Fextra-segment" — proving the ID is not encoded.
	assert.Equal(t, "/api/projects/proj/extra-segment", receivedPath,
		"RED: an ID with an embedded slash creates an extra path segment — url.PathEscape should be used")
}

func TestSecurity_GREEN_PathInjection_SimpleProjectIDIsCorrect(t *testing.T) {
	// GREEN: a simple alphanumeric ID goes to exactly the right path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/projects/proj-42", r.URL.Path)
		writeJSON(t, w, http.StatusOK, successResponse(map[string]any{
			"id": "proj-42", "name": "X", "description": "",
			"created_by_role": "", "created_by_agent": "", "default_role": "",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	result, err := c.GetProject("proj-42")
	require.NoError(t, err)
	assert.Equal(t, "proj-42", result.ID)
}

// ─── VULNERABILITY 4 ────────────────────────────────────────────────────────
// Credential exposure — the only way to authenticate with this client is to
// embed credentials inside the baseURL (e.g., http://user:pass@host/).
// There is no API to set an Authorization header or API key separately.
// Embedded URL credentials persist in process memory for the client lifetime
// and may leak in error messages, logs, and stack traces.
//
// File: pkg/server/client/client.go lines 16-26

func TestSecurity_GREEN_NoCredentials_RequestIsAnonymous(t *testing.T) {
	// GREEN: a URL without credentials sends no Authorization header.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"),
			"GREEN: without credentials in the URL no Authorization header should be sent")
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.ListProjects()
	require.NoError(t, err)
}

// ─── VULNERABILITY 5 ────────────────────────────────────────────────────────
// No response size limit — decodeResponse reads the entire response body with
// json.NewDecoder without any io.LimitReader guard.  A malicious or
// compromised server can return a multi-megabyte JSON body, causing the agent
// process to exhaust memory.
//
// File: pkg/server/client/client.go lines 77-89

func TestSecurity_RED_NoResponseSizeLimit_LargeBodyConsumed(t *testing.T) {
	// RED: the client reads a 2 MB response body without error.
	largePadding := strings.Repeat("x", 2*1024*1024) // 2 MB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a valid JSON envelope with a large string inside the data.
		body := `{"status":"success","data":[{"id":"` + largePadding +
			`","name":"x","description":"","created_by_role":"","created_by_agent":"",` +
			`"default_role":"","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}]}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	result, err := c.ListProjects()

	// Currently succeeds — the client reads the full 2 MB response.
	assert.NoError(t, err, "RED: a 2 MB response body is consumed without error — a response size limit should be enforced")
	assert.Len(t, result, 1, "RED: the oversized response was decoded as if normal")
}

func TestSecurity_GREEN_NormalSizedResponseIsDecoded(t *testing.T) {
	// GREEN: a normal-sized response is decoded correctly.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	result, err := c.ListProjects()
	require.NoError(t, err, "GREEN: a normal response must decode without error")
	assert.Empty(t, result)
}

// ─── VULNERABILITY 6 ────────────────────────────────────────────────────────
// GetColumnCounts uses a magic limit of 9999 — fetching up to 4 × 9999 full
// task objects just to count them.  This is a DoS amplification vector: an
// attacker who can populate columns can force the client to download and
// deserialise enormous payloads on every count check.
//
// File: pkg/server/client/client.go lines 340-360

func TestSecurity_RED_GetColumnCounts_ExcessiveDataFetch(t *testing.T) {
	// RED: GetColumnCounts fetches up to 9999 full task objects per column.
	var maxLimit string
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if l := r.URL.Query().Get("limit"); l != "" {
			maxLimit = l
		}
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.GetColumnCounts("proj-1")
	require.NoError(t, err)

	assert.Equal(t, 4, callCount, "GetColumnCounts must make 4 HTTP calls (one per column)")
	assert.Equal(t, "9999", maxLimit,
		"RED: GetColumnCounts requests up to 9999 full task objects per column — a count-only endpoint should be used")
}

func TestSecurity_GREEN_GetColumnCounts_ReturnsCorrectCounts(t *testing.T) {
	// GREEN: the counts returned are correct for the data the server returns.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		col := r.URL.Query().Get("column")
		switch col {
		case "todo":
			writeJSON(t, w, http.StatusOK, successResponse(make([]any, 3)))
		case "in_progress":
			writeJSON(t, w, http.StatusOK, successResponse(make([]any, 1)))
		default:
			writeJSON(t, w, http.StatusOK, successResponse([]any{}))
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	counts, err := c.GetColumnCounts("proj-1")
	require.NoError(t, err)
	assert.Equal(t, 3, counts.Todo)
	assert.Equal(t, 1, counts.InProgress)
}

// ─── VULNERABILITY 7 ────────────────────────────────────────────────────────
// UpdateTaskSessionID sends a caller-supplied session_id string with no length
// or format validation.  A very long or specially crafted string is forwarded
// to the server verbatim.
//
// File: pkg/server/client/client.go lines 213-221

func TestSecurity_GREEN_UpdateTaskSessionID_NormalSessionID(t *testing.T) {
	var receivedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSONBody(r, &receivedBody)
		writeJSON(t, w, http.StatusOK, successResponse(map[string]string{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	err := c.UpdateTaskSessionID("proj-1", "task-1", "session-abc-123")
	require.NoError(t, err, "GREEN: a normal session_id must be sent without error")
	assert.Equal(t, "session-abc-123", receivedBody["session_id"])
}
