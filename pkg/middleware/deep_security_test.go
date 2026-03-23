// Package middleware_test — deep security analysis tests.
//
// Each section below follows the RED/GREEN pattern:
//
//	RED  — a test that FAILS with the current code, demonstrating the vulnerability.
//	GREEN — a test that PASSES once the vulnerability is fixed (or documents the
//	        expected secure behaviour for new code).
//
// Run with: go test -race -failfast ./pkg/middleware/...
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Shared test helpers
// ─────────────────────────────────────────────────────────────────────────────

// deepSecurityMockAuth satisfies middleware.AuthValidator for deep security tests.
type deepSecurityMockAuth struct {
	validJWTs    map[string]any
	validAPIKeys map[string]any
}

func (m *deepSecurityMockAuth) ValidateJWT(_ context.Context, token string) (any, error) {
	if a, ok := m.validJWTs[token]; ok {
		return a, nil
	}
	return nil, errUnauthorized
}

func (m *deepSecurityMockAuth) ValidateAPIKey(_ context.Context, key string) (any, error) {
	if a, ok := m.validAPIKeys[key]; ok {
		return a, nil
	}
	return nil, errUnauthorized
}

func newDeepSecurityAuthHandler() http.Handler {
	mock := &deepSecurityMockAuth{
		validJWTs:    map[string]any{"valid-jwt-deep": testActor{Email: "user@example.com"}},
		validAPIKeys: map[string]any{"agach_validkey_deep": testActor{Email: "user@example.com"}},
	}
	return middleware.NewRequireAuth(mock)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 1 — IP spoofing via X-Forwarded-For bypasses rate limiter
//
// File: middleware.go:124-131
//
// clientIP() blindly trusts the X-Forwarded-For header sent by the client.
// An attacker who rotates the value of that header on every request can appear
// as a different IP each time, completely defeating the per-IP rate limiter.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_IPSpoofingBypassesRateLimit demonstrates that an
// attacker can send more than the burst limit of requests without being
// rate-limited by cycling through fake X-Forwarded-For values.
//
// RED: this test FAILS with current code because spoofing IS effective.
func TestDeepSecurity_RED_IPSpoofingBypassesRateLimit(t *testing.T) {
	const burst = 10 // matches globalLimiter.b
	handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send burst+5 requests, each with a unique fake X-Forwarded-For IP.
	// With the current implementation every request gets a fresh limiter bucket,
	// so none of them are ever rate-limited.
	rejectedCount := 0
	for i := 0; i < burst+5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/secure", nil)
		req.RemoteAddr = "10.0.0.1:9999" // single real connection
		req.Header.Set("X-Forwarded-For", strings.Join([]string{
			"1.2.3.", // different fake IP per request
		}, "") + string(rune('0'+i%10)))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			rejectedCount++
		}
	}

	// RED assertion: a secure implementation must reject at least one request
	// once the burst is exceeded on the real RemoteAddr.
	// Current code never rejects because it trusts X-Forwarded-For.
	assert.Greater(t, rejectedCount, 0,
		"RED: X-Forwarded-For spoofing allows unlimited requests — "+
			"fix: only trust X-Forwarded-For when request arrives from a known trusted proxy CIDR; "+
			"or remove X-Forwarded-For trust entirely and always key on r.RemoteAddr")
}

// TestDeepSecurity_GREEN_RateLimitKeyedOnRemoteAddr verifies the secure
// behaviour: when X-Forwarded-For is NOT trusted, the real RemoteAddr is used
// and the bucket is properly exhausted.
//
// GREEN: passes because RemoteAddr-based limiting already works.
func TestDeepSecurity_GREEN_RateLimitKeyedOnRemoteAddr(t *testing.T) {
	const burst = 10
	handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use a unique IP so this test does not share the global bucket with others.
	ip := "192.0.2.77:12300"
	var gotRejected bool
	for i := 0; i < burst+5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/secure", nil)
		req.RemoteAddr = ip
		// No X-Forwarded-For — forces RemoteAddr path.
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			gotRejected = true
			break
		}
	}

	assert.True(t, gotRejected,
		"GREEN: exhausting the burst on a fixed RemoteAddr must eventually return 429")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 2 — JWT token exposed in URL query parameter leaks via logs
//
// File: middleware.go:36-38
//
// The middleware accepts the JWT via ?token= query parameter "for WebSocket
// upgrades". URLs are written to access logs, browser history, Referer headers,
// and proxy caches. A token in a URL is trivially harvested.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_TokenInURLAppearsInRefererHeader demonstrates that a
// token accepted via query param could leak if the page makes any outbound
// request (image, script, API call) — the browser sends the full URL as Referer.
//
// RED: the test shows the middleware ACCEPTS the token from the query param,
// confirming the feature exists and is therefore a leakage surface.
func TestDeepSecurity_RED_TokenInURLAppearsInRefererHeader(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	// Attacker or accidental log captures this URL containing the JWT.
	req := httptest.NewRequest(http.MethodGet, "/?token=valid-jwt-deep", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// RED assertion: a secure implementation must NOT accept tokens via URL
	// params (or must at minimum respond with a redirect stripping the param).
	assert.NotEqual(t, http.StatusOK, rr.Code,
		"RED: token accepted via query param leaks in logs/Referer — "+
			"fix: reject ?token= param entirely, or only allow it over WSS with a short TTL nonce")
}

// TestDeepSecurity_GREEN_TokenInHeaderIsAccepted verifies the secure path:
// the same token sent via Authorization header is properly accepted.
//
// GREEN: passes with current code.
func TestDeepSecurity_GREEN_TokenInHeaderIsAccepted(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code,
		"GREEN: valid JWT in Authorization header must be accepted")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 3 — Missing security headers (information disclosure / XSS)
//
// File: middleware.go (no SecurityHeaders middleware exists)
//
// None of the middleware functions set defensive HTTP response headers:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Cache-Control: no-store (for authenticated responses)
//   - Content-Type on error responses
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_MissingXContentTypeOptionsHeader verifies that
// authenticated responses carry the X-Content-Type-Options header.
//
// RED: fails because no middleware sets this header.
func TestDeepSecurity_RED_MissingXContentTypeOptionsHeader(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	xCTO := rr.Header().Get("X-Content-Type-Options")
	assert.Equal(t, "nosniff", xCTO,
		"RED: X-Content-Type-Options: nosniff header is missing — "+
			"fix: add a SecurityHeaders middleware that sets this and other security headers")
}

// TestDeepSecurity_RED_MissingXFrameOptionsHeader verifies that responses
// carry the X-Frame-Options header to prevent clickjacking.
//
// RED: fails because no middleware sets this header.
func TestDeepSecurity_RED_MissingXFrameOptionsHeader(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	xfo := rr.Header().Get("X-Frame-Options")
	assert.Equal(t, "DENY", xfo,
		"RED: X-Frame-Options: DENY header is missing — "+
			"fix: add to SecurityHeaders middleware")
}

// TestDeepSecurity_RED_MissingCacheControlOnAuthenticatedResponse verifies
// that authenticated responses set Cache-Control: no-store to prevent
// sensitive data from being cached by intermediaries.
//
// RED: fails because no middleware sets Cache-Control.
func TestDeepSecurity_RED_MissingCacheControlOnAuthenticatedResponse(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	cc := rr.Header().Get("Cache-Control")
	assert.Contains(t, cc, "no-store",
		"RED: Cache-Control: no-store is missing on authenticated responses — "+
			"fix: set Cache-Control: no-store, no-cache in SecurityHeaders middleware")
}

// TestDeepSecurity_RED_UnauthorizedResponseMissingJSONContentType verifies
// that the 401 error response carries Content-Type: application/json, because
// the body is a JSON object. Currently http.Error sets text/plain which causes
// JSON-parsing clients to fail silently or show garbled output (information
// disclosure through inconsistent error format).
//
// RED: fails because http.Error in unauthorized() sets Content-Type: text/plain.
// Fix: replace http.Error with explicit header + WriteHeader + json body write.
func TestDeepSecurity_RED_UnauthorizedResponseMissingJSONContentType(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No auth — triggers the 401 path.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	ct := rr.Header().Get("Content-Type")
	assert.True(t,
		strings.HasPrefix(ct, "application/json"),
		"RED: unauthorized response must be Content-Type: application/json, got %q — "+
			"fix: replace http.Error in unauthorized() with w.Header().Set(\"Content-Type\", \"application/json\") + w.WriteHeader + json body write", ct)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 4 — Bearer prefix matching is case-sensitive, causing
//                   confusing auth failures and potential bypass edge cases
//
// File: middleware.go:49
//
// strings.TrimPrefix(authHeader, "Bearer ") only strips the exact string
// "Bearer ". If the client sends "bearer valid-jwt" or "BEARER valid-jwt",
// the prefix is NOT stripped and the entire string (including the word
// "bearer") is forwarded to ValidateJWT. This is both a usability defect
// and a subtle contract violation (RFC 7235 says the auth-scheme is
// case-insensitive).
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_BearerPrefixCaseSensitivity shows that a valid JWT
// sent with a lowercase "bearer" prefix is rejected even though RFC 7235
// requires case-insensitive scheme matching.
//
// RED: fails because the middleware rejects a valid token with lowercase prefix.
func TestDeepSecurity_RED_BearerPrefixCaseSensitivity(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	for _, prefix := range []string{"bearer ", "BEARER ", "Bearer  " /* double space */} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", prefix+"valid-jwt-deep")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code,
			"RED: Authorization header with scheme %q should be treated case-insensitively per RFC 7235; "+
				"fix: normalise authHeader with strings.ToLower before prefix detection, then extract token",
			prefix)
	}
}

// TestDeepSecurity_GREEN_BearerPrefixExactCaseWorks verifies the happy path
// that already works.
//
// GREEN: passes with current code.
func TestDeepSecurity_GREEN_BearerPrefixExactCaseWorks(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code,
		"GREEN: exact-case Bearer prefix must be accepted")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 5 — Whitespace-only Authorization header reaches ValidateJWT
//                   with an empty token
//
// File: middleware.go:31, 48-57
//
// A header value of "Bearer " (Bearer + one or more spaces only) is trimmed to
// "Bearer" by TrimSpace. That is non-empty, so the authHeader != "" check
// passes, and then TrimPrefix("Bearer ", "Bearer ") produces an empty string
// "" which is forwarded to ValidateJWT. The auth service receives an empty
// token and may or may not behave securely — this relies entirely on the auth
// service rather than the middleware itself rejecting clearly invalid input.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_EmptyTokenAfterBearerPrefix verifies that the middleware
// itself rejects a "Bearer " header (no token after prefix) with 401 rather
// than forwarding an empty string to the auth service.
//
// RED: currently the middleware forwards "" to ValidateJWT; if the mock auth
// service (correctly) rejects "", the test passes — but the middleware should
// reject this before ever calling ValidateJWT.
func TestDeepSecurity_RED_EmptyTokenAfterBearerPrefix(t *testing.T) {
	// We use a recording mock that panics if ValidateJWT is called with an empty
	// token, to confirm the middleware is responsible for the check.
	called := false
	panicIfEmptyToken := &deepSecurityMockAuth{
		validJWTs: map[string]any{},
	}
	_ = panicIfEmptyToken // prevent unused warning

	authCheckingMock := &authCallRecorder{
		onValidateJWT: func(token string) {
			called = true
			assert.NotEmpty(t, token,
				"RED: middleware must not forward empty token to ValidateJWT — "+
					"fix: after TrimPrefix, check len(token)==0 and call unauthorized() immediately")
		},
	}

	handler := middleware.NewRequireAuth(authCheckingMock)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, value := range []string{"Bearer ", "Bearer   ", "Bearer\t"} {
		called = false
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", value)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code,
			"RED: Authorization: %q with empty token must return 401 without calling ValidateJWT", value)
		// If called == true AND token was empty, the inner assert above fires.
		_ = called
	}
}

// authCallRecorder is a minimal AuthValidator mock that invokes a callback when
// ValidateJWT is called, allowing tests to inspect the token value.
type authCallRecorder struct {
	onValidateJWT func(token string)
}

func (a *authCallRecorder) ValidateJWT(_ context.Context, token string) (any, error) {
	if a.onValidateJWT != nil {
		a.onValidateJWT(token)
	}
	return nil, errUnauthorized
}

func (a *authCallRecorder) ValidateAPIKey(_ context.Context, _ string) (any, error) {
	return nil, errUnauthorized
}

// TestDeepSecurity_GREEN_NonEmptyTokenIsForwardedToValidateJWT confirms that a
// well-formed header sends the token (not the prefix) to ValidateJWT.
//
// GREEN: passes with current code.
func TestDeepSecurity_GREEN_NonEmptyTokenIsForwardedToValidateJWT(t *testing.T) {
	var received string
	mock := &authCallRecorder{
		onValidateJWT: func(token string) { received = token },
	}
	handler := middleware.NewRequireAuth(mock)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer my-token-value")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "my-token-value", received,
		"GREEN: token portion (without 'Bearer ') must be forwarded to ValidateJWT")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 6 — LimitBodySize does not limit GET/HEAD/DELETE requests
//                   that carry a body (unusual but valid HTTP)
//
// File: middleware.go:74-83
//
// The body limit applies to ALL methods, but there is no guard for Content-
// Length: -1 (unknown, chunked) combined with a maliciously large streaming
// body. http.MaxBytesReader IS applied, but only after the Content-Length
// check. A client that omits Content-Length and streams 1 GB will exercise
// MaxBytesReader — that is correct — but it depends on the handler actually
// reading the body. If no handler reads it, MaxBytesReader never triggers.
//
// More concretely: negative Content-Length values (< -1) are not rejected.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_NegativeContentLengthNotRejected verifies that a request
// with a bogus negative Content-Length value (other than -1 which means
// "unknown") is rejected rather than silently passed through.
//
// RED: current code only rejects Content-Length > maxBodyBytes; a negative
// value other than -1 (malformed request) is passed through.
func TestDeepSecurity_RED_NegativeContentLengthNotRejected(t *testing.T) {
	handler := middleware.LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req.ContentLength = -2 // bogus negative value — not -1 (unknown)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code,
		"RED: Content-Length: -2 is a malformed request and must be rejected with 400 — "+
			"fix: add check: if r.ContentLength < -1 { http.Error(..., 400); return }")
}

// TestDeepSecurity_GREEN_UnknownContentLengthIsAllowed verifies that -1
// (unknown/chunked) continues to be accepted.
//
// GREEN: passes with current code.
func TestDeepSecurity_GREEN_UnknownContentLengthIsAllowed(t *testing.T) {
	handler := middleware.LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("data"))
	req.ContentLength = -1 // unknown — chunked transfer
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		"GREEN: Content-Length: -1 (unknown) must be accepted")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 7 — Rate limiter global state causes test interference and
//                   potential production state persistence across handler reuse
//
// File: middleware.go:99-103
//
// globalLimiter is a package-level var. All calls to RateLimit share this
// single instance. There is no way to inject a custom limiter for testing,
// which means tests that exhaust a specific IP bucket can interfere with
// other tests or production deployments that reuse the package.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_GlobalLimiterStateLeaksBetweenHandlers shows that two
// independent RateLimit handler chains share the same underlying bucket state
// for the same IP. This is a test-isolation defect with production implications.
//
// RED: the test demonstrates that exhausting the limit in one handler chain
// also exhausts it in a separately constructed chain for the same IP.
func TestDeepSecurity_RED_GlobalLimiterStateLeaksBetweenHandlers(t *testing.T) {
	const burst = 10
	const ip = "192.0.2.222:1111"

	handlerA := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handlerB := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust bucket via handlerA.
	for i := 0; i < burst+5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		handlerA.ServeHTTP(httptest.NewRecorder(), req)
	}

	// Now send one request via handlerB — should be independent but will be
	// rate-limited because globalLimiter is shared.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	rr := httptest.NewRecorder()
	handlerB.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		"RED: independently created RateLimit handlers must not share bucket state — "+
			"fix: accept a *ipRateLimiter parameter in a NewRateLimit constructor so each handler "+
			"gets its own limiter; export a constructor and remove the package-level globalLimiter")
}

// TestDeepSecurity_GREEN_SeparateIPsHaveIndependentBuckets verifies that
// two different IPs do not share a bucket (the positive isolation case).
//
// GREEN: passes with current code.
func TestDeepSecurity_GREEN_SeparateIPsHaveIndependentBuckets(t *testing.T) {
	const burst = 10
	const ipA = "192.0.2.230:1000"
	const ipB = "192.0.2.231:1001"

	handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust bucket for ipA.
	for i := 0; i < burst+5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ipA
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	// ipB should have its own fresh bucket.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ipB
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		"GREEN: exhausting ipA's bucket must not affect ipB")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 8 — No protection against CORS / missing Vary header
//
// File: middleware.go (no CORS middleware)
//
// There is no CORS middleware. Any origin can make cross-site requests to the
// API. Combined with cookie/session auth (if added later), this is a CSRF
// vector. Even without cookies, missing Access-Control-Allow-Origin or an
// overly permissive wildcard is a disclosure risk.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_MissingCORSHeadersOnAuthenticatedResponse verifies that
// authenticated API responses include CORS restriction headers so browsers
// cannot be used as a cross-site request proxy.
//
// RED: fails because no CORS middleware exists.
func TestDeepSecurity_RED_MissingCORSHeadersOnAuthenticatedResponse(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	acao := rr.Header().Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "*", acao,
		"RED: Access-Control-Allow-Origin: * must not be returned for authenticated API responses — "+
			"fix: add a CORS middleware that sets allowed origins to an explicit allowlist")
	assert.NotEmpty(t, acao,
		"RED: missing Access-Control-Allow-Origin header — "+
			"fix: add a CORS middleware that explicitly sets the allowed origin or returns a 403 for disallowed origins")
}

// TestDeepSecurity_GREEN_PreflightOptionsReturns204 is a GREEN target:
// once a CORS middleware is added, OPTIONS preflight requests must return 204
// with the correct headers.
//
// GREEN (expectation, currently fails because no CORS middleware exists):
func TestDeepSecurity_GREEN_PreflightOptionsReturns204(t *testing.T) {
	t.Skip("GREEN target: add CORS middleware that handles OPTIONS preflight with 204 and correct CORS headers")
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 9 — Rate limiter memory leak (no cleanup of stale entries)
//
// File: middleware.go:85-117
//
// limiterEntry records lastSeen but there is no goroutine or function that
// removes entries whose lastSeen is old. Under sustained unique-IP traffic the
// limiters map grows without bound, eventually exhausting heap memory.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RED_RateLimiterMapGrowsUnbounded demonstrates that after N
// requests from N unique IPs the internal map has N entries and nothing removes
// them. The test is necessarily a documentation test (we cannot inspect
// unexported state from outside the package), so it is marked as a RED doc test.
//
// RED: documents the issue; the actual memory-leak check cannot be asserted
// from a black-box test — it requires either package-internal access or a
// lint/static analysis rule.
func TestDeepSecurity_RED_RateLimiterMapGrowsUnbounded(t *testing.T) {
	t.Log("RED (documentation): the globalLimiter.limiters map has no cleanup goroutine. " +
		"Under sustained unique-IP load the map grows without bound. " +
		"Fix: run a background ticker that deletes entries where time.Since(lastSeen) > TTL (e.g. 5 minutes). " +
		"Verify with: go test -memprofile=mem.out and inspect the map allocation.")
	// No hard assertion — this is a documentation/awareness test. Actual
	// enforcement requires static analysis or a memory-profiling integration test.
}

// TestDeepSecurity_GREEN_MultipleRequestsSameIPDoNotMultiplyMapEntries serves
// as a partial GREEN: the same IP re-uses its existing bucket entry.
//
// GREEN: passes with current code (existing entry is reused, not duplicated).
func TestDeepSecurity_GREEN_MultipleRequestsSameIPDoNotMultiplyMapEntries(t *testing.T) {
	const ip = "192.0.2.240:5555"
	handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Five requests from the same IP — all allowed (burst=10 for fresh IP).
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "request %d must pass", i)
	}
	// If the bucket were re-created each time (map insert on every request), the
	// burst would reset each time and never be exhausted. We verify the bucket IS
	// consumed by sending burst+1 requests and expecting at least one rejection.
	var rejected bool
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			rejected = true
			break
		}
	}
	assert.True(t, rejected,
		"GREEN: the same IP must share a single bucket; exhausting it must eventually produce 429")
}
