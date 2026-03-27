package security_test

// Additional RED security tests for pkg/server/client/client.go — round 2.
//
// These tests cover vulnerabilities NOT already documented in client_security_test.go.
// All RED tests assert CORRECT safe behaviour that is not yet implemented;
// they are expected to FAIL against the current production code.

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

// TestSecurity_RED_NoRequestTimeout asserts that the client enforces a request
// timeout and does NOT hang indefinitely on a slow server.
// Currently the client has no timeout so requests block for the full server
// delay; this test will FAIL until a default timeout is added.
// TODO(security): set a reasonable default Timeout (e.g., 30s) on the http.Client
func TestSecurity_RED_NoRequestTimeout(t *testing.T) {
	// Create a server that delays longer than a reasonable timeout should allow.
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Simulates slow/hung server
		writeJSON(t, w, http.StatusOK, successResponse([]any{}))
	}))
	defer slowServer.Close()

	c := client.New(slowServer.URL)

	// A client with a sensible timeout (e.g., <= 2s) must time out well before
	// the 5-second server delay completes.
	start := time.Now()
	done := make(chan struct{})
	go func() {
		_, _ = c.ListProjects()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		// The request must complete (via timeout error) in well under 5s.
		// We allow up to 3s to give generous leeway for CI slowness while still
		// proving that the client did not wait the full server delay.
		assert.Less(t, elapsed, 3*time.Second,
			"RED: the client must enforce a request timeout — currently it waits the full 5s server delay with no timeout")
	case <-time.After(6 * time.Second):
		t.Fatal("test timed out — client hung for more than 6s confirming no request timeout is enforced")
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

// TestSecurity_RED_SSRF_PrivateNetworkNotBlocked asserts that RFC 1918 private
// network addresses are rejected by New() at construction time, without making
// any live network connection.  The rejection must be detectable immediately
// (not after a TCP connection timeout) so the test uses a goroutine with a
// short deadline to distinguish "blocked at construction" from "hanging on I/O".
// Currently private addresses are accepted; this test will FAIL until the
// SSRF filter is extended to cover RFC 1918 ranges.
// TODO(security): block RFC 1918 private ranges and loopback in New()
func TestSecurity_RED_SSRF_PrivateNetworkNotBlocked(t *testing.T) {
	privateAddresses := []string{
		"http://10.0.0.1:8080",
		"http://172.16.0.1:8080",
		"http://192.168.1.1:8080",
	}

	for _, addr := range privateAddresses {
		addr := addr
		t.Run(addr, func(t *testing.T) {
			// The client must be created with an error for any private network address
			// so that calling any method immediately returns an error containing
			// "blocked" (or similar) rather than attempting a live connection.
			//
			// We run ListProjects() in a goroutine with a short deadline to
			// distinguish "blocked at construction (instant)" from "connecting to
			// network (hangs)".  If the call does not return within 200ms the
			// constructor accepted the address and is making a live network attempt,
			// which is the vulnerability being documented.
			c := client.New(addr)
			require.NotNil(t, c, "client.New always returns a non-nil *Client")

			type result struct {
				err error
			}
			ch := make(chan result, 1)
			go func() {
				_, err := c.ListProjects()
				ch <- result{err}
			}()

			select {
			case res := <-ch:
				require.Error(t, res.err,
					"RED: private network address %s must be blocked — currently New() accepts it", addr)
				assert.Contains(t, res.err.Error(), "blocked",
					"RED: the error for private address %s must mention 'blocked' to confirm SSRF protection", addr)
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("RED: client.New(%q) accepted the private address and is hanging on a live connection — must be blocked at construction", addr)
			}
		})
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

// TestSecurity_RED_ErrorStatusCodeNotChecked asserts that the client returns
// an error when the server responds with HTTP 500, even if the JSON body
// contains a "success" envelope.
// Currently decodeResponse() ignores the HTTP status code; this test will
// FAIL until status-code checking is added.
// TODO(security): check resp.StatusCode before decoding; reject non-2xx responses
func TestSecurity_RED_ErrorStatusCodeNotChecked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server returns 500 but with a "success" envelope.
		writeJSON(t, w, http.StatusInternalServerError, successResponse([]any{}))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.ListProjects()

	// An HTTP 500 must be treated as an error regardless of the JSON body.
	assert.Error(t, err,
		"RED: client must return an error for HTTP 500 — currently it accepts a 500 with a success JSON envelope as success")
}

// TestSecurity_RED_InternalErrorDetailsLeakedViaErrorResponse asserts that
// raw internal database error strings from the server are NOT forwarded
// verbatim to the caller.  The client must sanitize or strip internal details
// (e.g., "pq:", SQL state codes) from error messages before surfacing them.
// Currently the raw DB error is passed through; this test will FAIL until
// sanitization is implemented.
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

	// The client must NOT expose raw database error details to its callers.
	// Internal strings like "pq:" and "SQLSTATE" must be stripped or replaced
	// with a generic error message before being returned.
	assert.NotContains(t, err.Error(), "pq:",
		"RED: the client must not leak raw database driver errors ('pq:') to callers")
	assert.NotContains(t, err.Error(), "SQLSTATE",
		"RED: the client must not leak SQL state codes to callers")
}
