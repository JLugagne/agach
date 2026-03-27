package security_test

// Deep security tests for pkg/middleware — vulnerabilities NOT covered by
// the existing middleware_security_test.go or deep_security_test.go.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
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
// Fix: Validate the Origin header against an explicit allowlist.
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
			assert.NotEqual(t, origin, acao,
				"RED: server must NOT reflect attacker-controlled Origin %q into Access-Control-Allow-Origin — "+
					"fix: validate Origin against an explicit allowlist; reject or omit CORS headers for unknown origins", origin)
		})
	}
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
// Fix: Never set Access-Control-Allow-Origin to "null".
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
	assert.NotEqual(t, "null", acao,
		"RED: server must NOT set Access-Control-Allow-Origin: null when Origin is absent — "+
			"sandboxed iframes and data: URIs send Origin: null and would be allowed; "+
			"fix: omit ACAO header when no Origin is present, or set it to an explicit allowed origin")
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
// Fix: Always set Vary: Origin regardless of whether Origin
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
	// RED: Vary: Origin must be set unconditionally so caches never serve
	// a response with the wrong ACAO value to a client with a different Origin.
	assert.Contains(t, vary, "Origin",
		"RED: Vary: Origin must be set even when no Origin header is present — "+
			"fix: move w.Header().Set(\"Vary\", \"Origin\") outside the origin != \"\" branch")
}

// ---------------------------------------------------------------------------
// SECURITY: Security headers on unauthenticated responses
//
// File: middleware.go:72-74
//
// X-Content-Type-Options, X-Frame-Options, and Cache-Control are set
// inside NewRequireAuth before the auth check runs, so they appear on
// both authenticated and unauthenticated (401) responses. This is
// correct behavior, but it is an implementation detail: these headers
// are coupled to the auth middleware rather than being applied via a
// dedicated SecurityHeaders middleware that wraps ALL routes.
//
// Note: endpoints not wrapped by NewRequireAuth will lack these headers.
// ---------------------------------------------------------------------------

func TestSecurity_SecurityHeadersPresentOnUnauthenticatedResponse(t *testing.T) {
	handler := newDeepAuthHandler2()

	// Send a request that will be rejected (no auth) — verify security
	// headers are present on the 401 response.
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	// No Authorization header.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	// Security headers should be present on 401 responses because
	// NewRequireAuth sets them before performing the auth check.
	xcto := rr.Header().Get("X-Content-Type-Options")
	xfo := rr.Header().Get("X-Frame-Options")

	assert.Equal(t, "nosniff", xcto,
		"Security headers must be present on 401 responses")
	assert.Equal(t, "DENY", xfo,
		"Security headers must be present on 401 responses")
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
// Fix: Add a stop channel to ipRateLimiter and provide a Close()
// method, or use context.Context to control the goroutine lifetime.
// ---------------------------------------------------------------------------

func TestSecurity_RED_RateLimiterCleanupGoroutineLeaksOnRecreation(t *testing.T) {
	const handlerCount = 10

	before := runtime.NumGoroutine()

	for i := 0; i < handlerCount; i++ {
		handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		_ = handler
	}

	// Allow goroutines to start.
	runtime.Gosched()

	after := runtime.NumGoroutine()

	// RED: a secure implementation would not spawn long-lived goroutines when
	// creating handlers (or would provide a Close() method to stop them).
	// Current code spawns one cleanup goroutine per RateLimit() call with no
	// way to stop it, so the goroutine count must grow.
	assert.Equal(t, before, after,
		"RED: each call to RateLimit must not spawn a background goroutine that cannot be stopped — "+
			"got %d goroutines before, %d after creating %d handlers; "+
			"fix: provide a Close()/Stop() method or accept a context to control goroutine lifetime",
		before, after, handlerCount)
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
// Fix: Use w.Header().Set("Content-Type", "application/json")
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
	// RED: the body is a JSON object but Content-Type must be application/json
	// so that JSON-parsing clients can correctly parse the error response.
	assert.Contains(t, ct, "application/json",
		"RED: LimitBodySize error response must have Content-Type: application/json — "+
			"fix: replace http.Error with explicit w.Header().Set(\"Content-Type\", \"application/json\") + json body write")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: RateLimit response does not set Content-Type: application/json
//
// File: middleware.go:193
//
// Same issue as LimitBodySize: http.Error sets text/plain but the body is JSON.
//
// Fix: Replace http.Error with explicit JSON Content-Type header.
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
	// RED: the body is a JSON object but Content-Type must be application/json
	// so that JSON-parsing clients can correctly parse the 429 response.
	assert.Contains(t, ct, "application/json",
		"RED: RateLimit error response must have Content-Type: application/json — "+
			"fix: replace http.Error with explicit w.Header().Set(\"Content-Type\", \"application/json\") + json body write")
}
