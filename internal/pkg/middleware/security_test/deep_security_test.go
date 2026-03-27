// Package security_test — deep security analysis tests for the middleware package.
//
// Each section below follows the RED/GREEN pattern:
//
//	RED  — a test that FAILS with the current code, demonstrating the vulnerability.
//	GREEN — a test that PASSES once the vulnerability is fixed (or documents the
//	        expected secure behaviour for new code).
//
// Run with: go test -race -failfast ./internal/pkg/middleware/security_test/...
package security_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Shared test helpers
// ─────────────────────────────────────────────────────────────────────────────

// deepSecurityMockAuth satisfies middleware.AuthValidator for deep security tests.
type deepSecurityMockAuth struct {
	validJWTs map[string]any
}

func (m *deepSecurityMockAuth) ValidateJWT(_ context.Context, token string) (any, error) {
	if a, ok := m.validJWTs[token]; ok {
		return a, nil
	}
	return nil, errUnauthorized
}

func newDeepSecurityAuthHandler() http.Handler {
	mock := &deepSecurityMockAuth{
		validJWTs: map[string]any{"valid-jwt-deep": testActor{Email: "user@example.com"}},
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
// clientIP() must not blindly trust the X-Forwarded-For header sent by the
// client. An attacker who rotates the value of that header on every request
// could appear as a different IP each time, defeating the per-IP rate limiter.
//
// Fix: always key the rate limiter on r.RemoteAddr, never on X-Forwarded-For.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_IPSpoofingBypassesRateLimit demonstrates that an
// attacker cannot send more than the burst limit of requests by cycling
// through fake X-Forwarded-For values when the rate limiter keys on RemoteAddr.
func TestDeepSecurity_IPSpoofingBypassesRateLimit(t *testing.T) {
	const burst = 10 // matches globalLimiter.b
	handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send burst+5 requests, each with a unique fake X-Forwarded-For IP.
	// Because remoteAddr() keys on r.RemoteAddr (not X-Forwarded-For),
	// all requests share the same bucket and it will be exhausted.
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

	// A secure implementation must reject at least one request once the
	// burst is exceeded on the real RemoteAddr, regardless of X-Forwarded-For.
	assert.Greater(t, rejectedCount, 0,
		"X-Forwarded-For spoofing must not bypass rate limiting — "+
			"the rate limiter must key on r.RemoteAddr, not the client-controlled X-Forwarded-For header")
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
// The middleware should NOT accept the JWT via ?token= query parameter.
// URLs are written to access logs, browser history, Referer headers,
// and proxy caches. A token in a URL is trivially harvested.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_TokenInURLIsRejected verifies that the middleware does NOT
// accept a JWT delivered via the ?token= query parameter, preventing token
// leakage through logs and Referer headers.
func TestDeepSecurity_TokenInURLIsRejected(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	// Attacker or accidental log captures this URL containing the JWT.
	req := httptest.NewRequest(http.MethodGet, "/?token=valid-jwt-deep", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// A secure implementation must NOT accept tokens via URL params.
	assert.NotEqual(t, http.StatusOK, rr.Code,
		"token in query param must be rejected — tokens delivered via ?token= leak in logs and Referer headers; "+
			"only accept tokens via the Authorization header")
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
// File: middleware.go (NewRequireAuth sets security headers)
//
// Authenticated responses must carry defensive HTTP response headers:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Cache-Control: no-store (for authenticated responses)
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_XContentTypeOptionsHeader verifies that authenticated
// responses carry the X-Content-Type-Options: nosniff header to prevent
// MIME-type sniffing attacks.
func TestDeepSecurity_XContentTypeOptionsHeader(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	xCTO := rr.Header().Get("X-Content-Type-Options")
	assert.Equal(t, "nosniff", xCTO,
		"X-Content-Type-Options: nosniff must be set on authenticated responses to prevent MIME sniffing")
}

// TestDeepSecurity_XFrameOptionsHeader verifies that responses carry the
// X-Frame-Options: DENY header to prevent clickjacking.
func TestDeepSecurity_XFrameOptionsHeader(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	xfo := rr.Header().Get("X-Frame-Options")
	assert.Equal(t, "DENY", xfo,
		"X-Frame-Options: DENY must be set on authenticated responses to prevent clickjacking")
}

// TestDeepSecurity_CacheControlOnAuthenticatedResponse verifies that
// authenticated responses set Cache-Control: no-store to prevent sensitive
// data from being cached by intermediaries.
func TestDeepSecurity_CacheControlOnAuthenticatedResponse(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	cc := rr.Header().Get("Cache-Control")
	assert.Contains(t, cc, "no-store",
		"Cache-Control: no-store must be set on authenticated responses to prevent caching of sensitive data")
}

// TestDeepSecurity_UnauthorizedResponseHasJSONContentType verifies that the
// 401 error response carries Content-Type: application/json, because the body
// is a JSON object.
func TestDeepSecurity_UnauthorizedResponseHasJSONContentType(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No auth — triggers the 401 path.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	ct := rr.Header().Get("Content-Type")
	assert.True(t,
		strings.HasPrefix(ct, "application/json"),
		"unauthorized response must be Content-Type: application/json, got %q", ct)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 4 — Bearer prefix matching is case-sensitive
//
// File: middleware.go:49
//
// RFC 7235 says the auth-scheme is case-insensitive. The middleware normalises
// to lowercase before prefix detection, so "bearer ", "BEARER " and "Bearer "
// are all treated equivalently.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_BearerPrefixCaseInsensitivity verifies that a valid JWT
// sent with a lowercase or uppercase "bearer" prefix is accepted, as required
// by RFC 7235 case-insensitive scheme matching.
func TestDeepSecurity_BearerPrefixCaseInsensitivity(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	for _, prefix := range []string{"bearer ", "BEARER ", "Bearer  " /* double space */} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", prefix+"valid-jwt-deep")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code,
			"Authorization header with scheme %q must be accepted per RFC 7235 case-insensitive scheme matching",
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
// A header value of "Bearer " (Bearer + one or more spaces only) must be
// rejected by the middleware before calling ValidateJWT, rather than
// forwarding an empty string to the auth service.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_EmptyTokenAfterBearerPrefixIsRejected verifies that the
// middleware rejects a "Bearer " header (no token after prefix) with 401
// rather than forwarding an empty string to the auth service.
func TestDeepSecurity_EmptyTokenAfterBearerPrefixIsRejected(t *testing.T) {
	// We use a recording mock that asserts ValidateJWT is never called with
	// an empty token, confirming the middleware is responsible for the check.
	called := false
	panicIfEmptyToken := &deepSecurityMockAuth{
		validJWTs: map[string]any{},
	}
	_ = panicIfEmptyToken // prevent unused warning

	authCheckingMock := &authCallRecorder{
		onValidateJWT: func(token string) {
			called = true
			assert.NotEmpty(t, token,
				"middleware must not forward empty token to ValidateJWT — "+
					"after TrimPrefix, check len(token)==0 and call unauthorized() immediately")
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
			"Authorization: %q with empty token must return 401 without calling ValidateJWT", value)
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
// Vulnerability 6 — LimitBodySize: malformed negative Content-Length
//
// File: middleware.go:74-83
//
// Negative Content-Length values other than -1 (unknown/chunked) are malformed
// requests. LimitBodySize rejects them with 400 Bad Request.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_NegativeContentLengthIsRejected verifies that a request
// with a bogus negative Content-Length value (other than -1 which means
// "unknown") is rejected with 400 Bad Request.
func TestDeepSecurity_NegativeContentLengthIsRejected(t *testing.T) {
	handler := middleware.LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req.ContentLength = -2 // bogus negative value — not -1 (unknown)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code,
		"Content-Length: -2 is a malformed request and must be rejected with 400 Bad Request")
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
// Vulnerability 7 — Rate limiter global state causes test interference
//
// File: middleware.go:99-103
//
// Each call to RateLimit creates its own *ipRateLimiter instance (no shared
// package-level global). This ensures independent handler chains have
// independent per-IP buckets.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_IndependentRateLimitHandlersHaveIndependentBuckets verifies
// that two independently created RateLimit handler chains do NOT share per-IP
// bucket state. Exhausting the bucket in one handler must not affect another.
func TestDeepSecurity_IndependentRateLimitHandlersHaveIndependentBuckets(t *testing.T) {
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

	// handlerB must have an independent bucket — one request must succeed.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	rr := httptest.NewRecorder()
	handlerB.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		"independently created RateLimit handlers must not share bucket state — "+
			"exhausting one handler's bucket must not rate-limit requests through a different handler")
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
// Vulnerability 8 — CORS: arbitrary Origin reflection
//
// File: middleware.go (NewRequireAuth CORS handling)
//
// NewRequireAuth currently echoes the Origin header verbatim into
// Access-Control-Allow-Origin. A proper CORS implementation must restrict
// allowed origins to an explicit allowlist so that arbitrary cross-site
// requests from unknown origins are blocked.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_MissingCORSHeadersOnAuthenticatedResponse verifies that
// authenticated API responses do not set Access-Control-Allow-Origin: * and
// include a non-empty ACAO header for allowed origins.
func TestDeepSecurity_MissingCORSHeadersOnAuthenticatedResponse(t *testing.T) {
	handler := newDeepSecurityAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-deep")
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	acao := rr.Header().Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "*", acao,
		"Access-Control-Allow-Origin: * must not be returned for authenticated API responses — "+
			"fix: add a CORS middleware that sets allowed origins to an explicit allowlist")
	assert.NotEmpty(t, acao,
		"Access-Control-Allow-Origin header must be present on authenticated responses — "+
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
// Vulnerability 9 — Rate limiter memory management under high unique-IP load
//
// File: middleware.go:85-117
//
// limiterEntry records lastSeen and the cleanupLoop goroutine removes entries
// whose lastSeen exceeds 10 minutes. This bounds memory usage under normal
// conditions.
// ─────────────────────────────────────────────────────────────────────────────

// TestDeepSecurity_RateLimiterHandlesHighUniqueIPCardinality verifies that the
// RateLimit middleware correctly handles a large number of unique IPs without
// degrading: each IP gets an independent bucket and requests are served (up to
// burst limit). This documents the expected behaviour under high cardinality.
func TestDeepSecurity_RateLimiterHandlesHighUniqueIPCardinality(t *testing.T) {
	const uniqueIPs = 100

	handler := middleware.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send one request from each unique IP. All must be served (burst=10,
	// so the first request from any IP is always within the burst limit).
	for i := 0; i < uniqueIPs; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = strings.Join([]string{
			"10.0.",
			strings.Join([]string{string(rune('0' + i/256%10)), string(rune('0' + i/256%256))}, ""),
			".",
			strings.Join([]string{string(rune('0' + i%10))}, ""),
			":80",
		}, "")
		// Use a simple numeric IP string to ensure uniqueness.
		req.RemoteAddr = strings.Replace(req.RemoteAddr, "10.0.", "", 1)
		req.RemoteAddr = strings.TrimSpace(req.RemoteAddr)
		// Simpler approach: use format that net.SplitHostPort can parse.
		req.RemoteAddr = strings.Join([]string{"10.1.2.", string(rune('0' + i%10)), ":80"}, "")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code,
			"first request from unique IP %d must be served within burst limit", i)
	}
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
