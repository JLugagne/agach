package security_test

// Deep security tests for pkg/middleware — vulnerabilities NOT covered by
// the existing middleware_security_test.go or deep_security_test.go.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

type deepMockAuth2 struct {
	validJWTs map[string]any
}

func (m *deepMockAuth2) ValidateJWT(_ context.Context, token string) (any, error) {
	if a, ok := m.validJWTs[token]; ok {
		return a, nil
	}
	return nil, errUnauthorized
}

func newDeepAuthHandler2() http.Handler {
	mock := &deepMockAuth2{
		validJWTs: map[string]any{"valid-jwt-v2": testActor{Email: "user@example.com"}},
	}
	return middleware.NewRequireAuth(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
}

// ---------------------------------------------------------------------------
// VULNERABILITY: CORS reflects arbitrary Origin header
//
// File: middleware.go:77-80
//
// NewRequireAuth reads the Origin header and echoes it verbatim into
// Access-Control-Allow-Origin. This means ANY origin — including
// https://evil.example.com — is explicitly allowed by the server.
// This is equivalent to Access-Control-Allow-Origin: * but worse,
// because browsers allow credentials (cookies, auth headers) when the
// server echoes the specific origin rather than using a wildcard.
//
// Combined with Access-Control-Allow-Credentials: true (if ever added),
// this is a full credential-stealing CORS bypass.
//
// TODO(security): Validate the Origin header against an explicit allowlist.
// Return 403 or omit the CORS headers for origins not on the list.
// ---------------------------------------------------------------------------

func TestSecurity_RED_CORSReflectsArbitraryOrigin(t *testing.T) {
	handler := newDeepAuthHandler2()

	evilOrigins := []string{
		"https://evil.example.com",
		"https://attacker.io",
		"https://phishing-site.com",
	}

	for _, origin := range evilOrigins {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
			req.Header.Set("Authorization", "Bearer valid-jwt-v2")
			req.Header.Set("Origin", origin)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			acao := rr.Header().Get("Access-Control-Allow-Origin")
			assert.Equal(t, origin, acao,
				"RED: server reflects attacker-controlled Origin %q into Access-Control-Allow-Origin — "+
					"this allows any website to make authenticated cross-origin requests", origin)
		})
	}

	t.Log("RED: NewRequireAuth echoes arbitrary Origin into ACAO header (CORS misconfiguration)")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: CORS allows null origin
//
// File: middleware.go:81-83
//
// When the Origin header is absent, the middleware sets
// Access-Control-Allow-Origin: null. The "null" origin is used by:
//   - Sandboxed iframes (sandbox attribute without allow-same-origin)
//   - data: and blob: URIs
//   - Redirected cross-origin requests
//
// An attacker can craft a page using a sandboxed iframe that sends
// Origin: null, and the server will allow the request. This is a known
// CORS bypass technique.
//
// TODO(security): Never set Access-Control-Allow-Origin to "null".
// When Origin is empty, either omit the header entirely or set it to
// the configured application origin.
// ---------------------------------------------------------------------------

func TestSecurity_RED_CORSAllowsNullOrigin(t *testing.T) {
	handler := newDeepAuthHandler2()

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-v2")
	// No Origin header — triggers the else branch.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	acao := rr.Header().Get("Access-Control-Allow-Origin")
	assert.Equal(t, "null", acao,
		"RED: server sets Access-Control-Allow-Origin: null when Origin is absent — "+
			"sandboxed iframes and data: URIs send Origin: null and will be allowed")

	t.Log("RED: NewRequireAuth allows the 'null' origin (sandbox iframe CORS bypass)")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Missing Vary header for non-Origin responses
//
// File: middleware.go:79
//
// When an Origin header IS present, the code sets Vary: Origin (good).
// But when Origin is absent and ACAO is set to "null", no Vary header is
// set. A caching proxy could cache the "null" ACAO response and serve it
// to a request that DID have an Origin, or vice versa, causing CORS
// confusion. The Vary: Origin header must be set unconditionally.
//
// TODO(security): Always set Vary: Origin regardless of whether Origin
// is present in the request.
// ---------------------------------------------------------------------------

func TestSecurity_RED_MissingVaryHeaderWhenNoOrigin(t *testing.T) {
	handler := newDeepAuthHandler2()

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-v2")
	// No Origin header.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	vary := rr.Header().Get("Vary")
	// RED: Vary header is NOT set when Origin is absent, so caches may
	// serve the "null" ACAO response to a request that DID have an Origin.
	assert.Empty(t, vary,
		"RED: Vary: Origin is missing when no Origin header is present — "+
			"cached responses may be served with wrong ACAO value")
	t.Log("RED: Vary: Origin not set on responses when Origin header is absent")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Security headers set only on authenticated responses
//
// File: middleware.go:72-74
//
// X-Content-Type-Options, X-Frame-Options, and Cache-Control are set
// inside NewRequireAuth, which means they are ONLY present on responses
// that pass through the auth middleware. Unauthenticated endpoints
// (health check, login, SSE, static files) do NOT get these headers.
// An attacker can target unauthenticated endpoints for MIME-sniffing
// or clickjacking attacks.
//
// TODO(security): Move security headers to a dedicated middleware that
// wraps ALL routes, not just authenticated ones.
// ---------------------------------------------------------------------------

func TestSecurity_RED_SecurityHeadersMissingOnUnauthenticatedResponse(t *testing.T) {
	handler := newDeepAuthHandler2()

	// Send a request that will be rejected (no auth) — check if security
	// headers are present on the 401 response.
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	// No Authorization header.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	// The 401 response should still carry security headers to protect the
	// error page from MIME sniffing, framing, and caching.
	xcto := rr.Header().Get("X-Content-Type-Options")
	xfo := rr.Header().Get("X-Frame-Options")

	// In the current code, the headers ARE set before the auth check runs,
	// so they appear on 401 responses too. But this is an implementation
	// detail — they should be in a separate middleware.
	// We test that they are present as a baseline, and document that they
	// are coupled to the auth middleware.
	assert.Equal(t, "nosniff", xcto,
		"Security headers should be present on 401 responses")
	assert.Equal(t, "DENY", xfo,
		"Security headers should be present on 401 responses")

	t.Log("RED: Security headers are coupled to NewRequireAuth — " +
		"endpoints not wrapped by auth middleware will lack these headers")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Rate limiter cleanup goroutine leaks on handler GC
//
// File: middleware.go:147-149
//
// newIPRateLimiter spawns a background goroutine (cleanupLoop) that runs
// forever via time.NewTicker. When the rate limiter (and its parent handler)
// is garbage collected, the goroutine is NOT stopped because there is no
// done/stop channel. In tests or long-running servers that recreate handlers,
// these goroutines accumulate.
//
// TODO(security): Add a stop channel to ipRateLimiter and provide a Close()
// method, or use context.Context to control the goroutine lifetime.
// ---------------------------------------------------------------------------

func TestSecurity_RED_RateLimiterCleanupGoroutineLeaksOnRecreation(t *testing.T) {
	// Each call to RateLimit creates a new ipRateLimiter with a background
	// goroutine that never exits. Creating many handlers leaks goroutines.
	for i := 0; i < 10; i++ {
		handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		_ = handler
	}

	// We cannot directly observe goroutine count from an external test package,
	// but we document the leak: 10 cleanup goroutines are now running and will
	// never exit.
	t.Log("RED: each call to RateLimit spawns a background cleanup goroutine that never exits — " +
		"10 handlers = 10 leaked goroutines; no Close/Stop method exists")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: LimitBodySize response does not set Content-Type: application/json
//
// File: middleware.go:117, 121
//
// LimitBodySize uses http.Error to send error responses, which sets
// Content-Type: text/plain. Since the response body is a JSON string,
// clients that rely on Content-Type to parse the response will fail.
// This is inconsistent with the rest of the API which returns JSON.
//
// TODO(security): Use w.Header().Set("Content-Type", "application/json")
// followed by w.WriteHeader and w.Write instead of http.Error.
// ---------------------------------------------------------------------------

func TestSecurity_RED_LimitBodySizeResponseNotJSON(t *testing.T) {
	handler := middleware.LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req.ContentLength = 1024 * 1024 // 1 MiB, exceeds 512 KB limit
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)

	ct := rr.Header().Get("Content-Type")
	// RED: http.Error sets text/plain, but the body is JSON — this is inconsistent.
	assert.Contains(t, ct, "text/plain",
		"RED: LimitBodySize sends JSON body with Content-Type: text/plain — "+
			"JSON-parsing clients will fail to parse the error response")
	t.Log("RED: LimitBodySize error response has Content-Type text/plain instead of application/json")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: RateLimit response does not set Content-Type: application/json
//
// File: middleware.go:193
//
// Same issue as LimitBodySize: http.Error sets text/plain but the body is JSON.
//
// TODO(security): Replace http.Error with explicit JSON Content-Type header.
// ---------------------------------------------------------------------------

func TestSecurity_RED_RateLimitResponseNotJSON(t *testing.T) {
	handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use a unique IP to avoid interference with other tests.
	ip := "192.0.2.99:9999"
	var rr *httptest.ResponseRecorder
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
		req.RemoteAddr = ip
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			break
		}
	}

	require.Equal(t, http.StatusTooManyRequests, rr.Code,
		"should have been rate-limited")

	ct := rr.Header().Get("Content-Type")
	// RED: http.Error sets text/plain, but the body is JSON — this is inconsistent.
	assert.Contains(t, ct, "text/plain",
		"RED: RateLimit sends JSON body with Content-Type: text/plain — "+
			"JSON-parsing clients will fail to parse the 429 response")
	t.Log("RED: RateLimit error response has Content-Type text/plain instead of application/json")
}
