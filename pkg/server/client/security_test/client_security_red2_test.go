package security_test

// Additional RED security tests for pkg/server/client/client.go — round 2.
//
// These tests cover vulnerabilities NOT already documented in client_security_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/pkg/server/client"
)

// ---- VULNERABILITY 8 --------------------------------------------------------
// No HTTP request timeout — the client is created with `&http.Client{}`
// which has Timeout=0, meaning requests can hang indefinitely. A malicious or
// unresponsive server can hold client goroutines open forever, exhausting
// resources in the agent process.
//
// File: pkg/server/client/client.go line 39 — `httpClient: &http.Client{}`

// TestSecurity_RED_NoRequestTimeout documents that the client has no timeout
// and will hang indefinitely on a slow server.
// TODO(security): set a reasonable default Timeout (e.g., 30s) on the http.Client
func TestSecurity_RED_NoRequestTimeout(t *testing.T) {
	// Create a server that delays longer than a reasonable timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Simulates slow/hung server
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer slowServer.Close()

	c := client.New(slowServer.URL)

	// If the client had a timeout, this would fail after the timeout.
	// Instead it will hang for the full 5 seconds.
	start := time.Now()
	done := make(chan struct{})
	go func() {
		_, _ = c.ListProjects()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		// The request completed after the full 5s delay, proving no timeout was enforced
		assert.GreaterOrEqual(t, elapsed, 4*time.Second,
			"RED: the client waited the full server delay — no request timeout is enforced")
		t.Log("RED: client has no HTTP timeout — requests can hang indefinitely on unresponsive servers")
	case <-time.After(6 * time.Second):
		t.Fatal("test timed out — server/client deadlock")
	}
}

// ---- VULNERABILITY 9 --------------------------------------------------------
// Private/internal network SSRF — the client blocks a few specific metadata
// endpoint IPs (169.254.169.254, metadata.google.internal, 169.254.170.2)
// and link-local unicast addresses, but does NOT block RFC 1918 private
// ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) or loopback (127.0.0.0/8
// besides the metadata check).
//
// An attacker who controls the baseURL can target internal services on these
// private networks.
//
// File: pkg/server/client/client.go lines 23-72

// TestSecurity_RED_SSRF_PrivateNetworkNotBlocked documents that private
// network addresses are accepted by the client constructor.
// TODO(security): block RFC 1918 private ranges and loopback in New()
func TestSecurity_RED_SSRF_PrivateNetworkNotBlocked(t *testing.T) {
	privateAddresses := []string{
		"http://10.0.0.1:8080",
		"http://172.16.0.1:8080",
		"http://192.168.1.1:8080",
	}

	for _, addr := range privateAddresses {
		c := client.New(addr)
		// Try to make a request — it will fail because the host doesn't exist,
		// but the constructor should have rejected it upfront.
		_, err := c.ListProjects()
		// If err is nil or a connection error (not a "blocked" error), the
		// constructor accepted the private address.
		if err != nil {
			assert.NotContains(t, err.Error(), "blocked",
				"RED: private address %s is not blocked by the client constructor", addr)
		}
		t.Logf("RED: private network address %s accepted by client.New() — should be blocked for SSRF prevention", addr)
	}
}

// ---- VULNERABILITY 10 -------------------------------------------------------
// HTTP response status code not checked — decodeResponse() reads and decodes
// the response body regardless of the HTTP status code. A 500 Internal Server
// Error with a valid JSON envelope is silently treated as success. Error
// responses that include internal details in the JSON message are passed
// through to callers.
//
// File: pkg/server/client/client.go lines 129-142

// TestSecurity_RED_ErrorStatusCodeNotChecked documents that the client accepts
// a 500 response as success if the JSON envelope has status="success".
// TODO(security): check resp.StatusCode before decoding; reject non-2xx responses
func TestSecurity_RED_ErrorStatusCodeNotChecked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server returns 500 but with a "success" envelope
		writeJSON(t, w, http.StatusInternalServerError, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	result, err := c.ListProjects()

	// The client should reject this because the HTTP status is 500.
	// Currently it decodes the body and returns success.
	assert.NoError(t, err,
		"RED: client accepts HTTP 500 with success JSON envelope as a successful response")
	assert.Empty(t, result)
	t.Log("RED: decodeResponse() ignores HTTP status code — 500 with valid JSON envelope treated as success")
}

// TestSecurity_RED_InternalErrorDetailsLeakedViaErrorResponse documents that
// internal error details from the server error response are passed directly
// to the caller without sanitization.
// TODO(security): sanitize error messages from server responses before returning to callers
func TestSecurity_RED_InternalErrorDetailsLeakedViaErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error": map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": "pq: relation \"users\" does not exist (SQLSTATE 42P01)",
			},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.ListProjects()
	require.Error(t, err)

	// The raw internal database error is forwarded to the caller
	assert.Contains(t, err.Error(), "pq:",
		"RED: internal database error details are leaked to the client caller")
	assert.Contains(t, err.Error(), "SQLSTATE",
		"RED: SQL state codes are exposed in client error messages")
	t.Log("RED: server error messages containing internal details (DB errors) are passed through to callers unsanitized")
}
