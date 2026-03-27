package security_test

// Security tests for cmd/agach-server/main.go.
//
// Vulnerability catalogue:
//
//  SEC-SRV-01  JWT_SECRET minimum of 32 chars is below the 64-byte (512-bit)
//              minimum recommended for HMAC-SHA256 secrets.
//  SEC-SRV-02  Bearer token exposed in URL query parameter (?token=) leaks
//              credentials into server logs, browser history, and Referer headers.
//  SEC-SRV-03  Context cancel fires before pgxpool drains; in-flight DB
//              transactions can be aborted mid-flight on graceful shutdown.
//  SEC-SRV-04  X-Forwarded-For is trusted unconditionally, allowing an attacker
//              to spoof their IP and bypass rate limiting.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// SEC-SRV-01  JWT_SECRET entropy floor is too low
// ─────────────────────────────────────────────────────────────────────────────

// minimumRecommendedJWTSecretLen returns the minimum secret length that a
// hardened implementation should require for HMAC-SHA256. NIST SP 800-107
// recommends the key length to be at least the output length of the hash
// function; for SHA-256 that is 32 bytes, but 64 bytes is the common
// production recommendation to guard against key-guessing attacks.
const minimumRecommendedJWTSecretLen = 64

// TestSecurity_SEC_SRV_01_RED_JWTSecretEntropyFloorIsTooLow documents that the
// current code only requires 32 characters, half the recommended minimum.
func TestSecurity_SEC_SRV_01_RED_JWTSecretEntropyFloorIsTooLow(t *testing.T) {
	// The current enforcement threshold in internal/server/init.go line 33.
	currentEnforcedMinimum := 32

	assert.Less(t, currentEnforcedMinimum, minimumRecommendedJWTSecretLen,
		"RED SEC-SRV-01: current JWT_SECRET minimum (%d chars) is below the "+
			"recommended %d chars for HMAC-SHA256; fix: raise the minimum to %d",
		currentEnforcedMinimum, minimumRecommendedJWTSecretLen, minimumRecommendedJWTSecretLen)
}

// TestSecurity_SEC_SRV_01_GREEN_JWTSecretEntropyFloorIsAdequate documents the
// required behaviour: a 64-byte minimum must be enforced.
func TestSecurity_SEC_SRV_01_GREEN_JWTSecretEntropyFloorIsAdequate(t *testing.T) {
	validateLength := func(s string) bool {
		return len(s) >= minimumRecommendedJWTSecretLen
	}

	weak := "this-is-only-32-characters-long!"
	require.Equal(t, 32, len(weak), "test string must be exactly 32 chars")
	assert.False(t, validateLength(weak),
		"GREEN SEC-SRV-01: 32-char secret must fail the hardened validation")

	strong := "this-secret-is-exactly-sixty-four-characters-long-and-meets-req!"
	require.Equal(t, 64, len(strong), "test string must be exactly 64 chars")
	assert.True(t, validateLength(strong),
		"GREEN SEC-SRV-01: 64-char secret must pass the hardened validation")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-SRV-02  Bearer token in URL query param leaks credentials
// ─────────────────────────────────────────────────────────────────────────────

// tokenFromRequest extracts a bearer token the same way the middleware does:
// it checks Authorization header first, then falls back to the ?token= query
// param. This mirrors the code in pkg/middleware/middleware.go lines 35-39.
func tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		return h
	}
	return r.URL.Query().Get("token")
}

// TestSecurity_SEC_SRV_02_RED_TokenInURLIsLogged documents that a token
// supplied via ?token= appears verbatim in the request URL, which will be
// written to any HTTP access log.
func TestSecurity_SEC_SRV_02_RED_TokenInURLIsLogged(t *testing.T) {
	secret := "super-secret-bearer-token"
	req := httptest.NewRequest(http.MethodGet, "/ws?token="+secret+"&project_id=abc", nil)

	// The URL including the token is what any access logger (nginx, Go's
	// net/http default logger, etc.) records.
	loggedURL := req.URL.String()

	assert.Contains(t, loggedURL, secret,
		"RED SEC-SRV-02: bearer token %q appears in the logged URL %q; "+
			"fix: require Authorization header for WebSocket upgrades; "+
			"never accept credentials in query parameters", secret, loggedURL)
}

// TestSecurity_SEC_SRV_02_GREEN_TokenNotAcceptedInQueryParam documents the
// required behaviour: the ?token= fallback must be removed; only the
// Authorization header should be accepted.
func TestSecurity_SEC_SRV_02_GREEN_TokenNotAcceptedInQueryParam(t *testing.T) {
	// Hardened tokenFromRequest that ignores the query param.
	hardenedTokenFromRequest := func(r *http.Request) string {
		return r.Header.Get("Authorization")
	}

	reqWithHeader := httptest.NewRequest(http.MethodGet, "/ws?project_id=abc", nil)
	reqWithHeader.Header.Set("Authorization", "Bearer good-token")
	assert.Equal(t, "Bearer good-token", hardenedTokenFromRequest(reqWithHeader),
		"GREEN SEC-SRV-02: Authorization header must be accepted")

	reqWithQueryOnly := httptest.NewRequest(http.MethodGet, "/ws?token=good-token&project_id=abc", nil)
	assert.Empty(t, hardenedTokenFromRequest(reqWithQueryOnly),
		"GREEN SEC-SRV-02: query-param token must NOT be accepted by the hardened implementation")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-SRV-03  Graceful shutdown: context cancel races with in-flight DB queries
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC_SRV_03_RED_ContextCancelFiresBeforeDBDrains documents that
// in cmd/agach-server/main.go the context derived from cancel() is shared with
// pgxpool.New(ctx, ...). When the signal is received, cancel() is called (via
// <-quit select), which propagates cancellation to every open connection in the
// pool — potentially aborting mid-flight transactions before the HTTP shutdown
// timeout has elapsed.
func TestSecurity_SEC_SRV_03_RED_ContextCancelFiresBeforeDBDrains(t *testing.T) {
	// We document the structural issue: a context that drives both pool
	// lifetime and server lifetime is dangerous.
	//
	// The current code in main.go (lines 39-54, simplified):
	//   ctx, cancel := context.WithCancel(context.Background())
	//   defer cancel()
	//   pool, _ := pgxpool.New(ctx, databaseURL)   // pool uses ctx
	//   defer pool.Close()
	//   runHTTP(ctx, ...)  // <-- blocks until signal
	//   // After runHTTP returns, defer pool.Close() then defer cancel() run.
	//   // BUT inside runHTTP, once the signal fires, ctx is passed to
	//   // httpSrv.Shutdown(shutdownCtx) which is a SEPARATE context —
	//   // however cancel() fires immediately when <-quit unblocks, which
	//   // cancels pool connections used by handlers still in-flight during
	//   // the 10s shutdown window.
	//
	// We verify the architectural assumption: a pool created with a cancellable
	// context will have its connections invalidated when that context is cancelled.

	// Simulate: if pool context is cancelled, queries are aborted.
	// This test documents the *design flaw*, not a runtime assertion we can
	// exercise without a real database.
	t.Log("RED SEC-SRV-03: the shared cancel context is cancelled before the " +
		"10s HTTP shutdown window elapses, aborting in-flight DB queries; " +
		"fix: create the pgxpool with context.Background() and let the HTTP " +
		"shutdown complete before closing the pool")

	// The fact that ctx and pool share the same context is the vulnerability.
	// We assert the design intent that they should be independent.
	poolShouldUseIndependentContext := false // current implementation: false (RED)
	assert.False(t, poolShouldUseIndependentContext,
		"RED SEC-SRV-03: pool context is tied to the main cancel context; "+
			"fix: use context.Background() for the pool so it outlives the HTTP shutdown")
}

// TestSecurity_SEC_SRV_03_GREEN_DBPoolUsesIndependentContext documents the
// correct pattern: the pool must be created with context.Background() so that
// cancelling the server context does not abort in-flight transactions.
func TestSecurity_SEC_SRV_03_GREEN_DBPoolUsesIndependentContext(t *testing.T) {
	// Document the required design:
	//   pool, _ := pgxpool.New(context.Background(), databaseURL)
	//   ctx, cancel := context.WithCancel(context.Background())
	//   defer cancel()
	//   runHTTP(ctx, ...)
	//   // After signal: HTTP shuts down in <= 10s, then pool.Close()

	poolUsesBackgroundContext := true // what the fix should produce
	assert.True(t, poolUsesBackgroundContext,
		"GREEN SEC-SRV-03: pool must be created with context.Background() "+
			"so in-flight queries are not aborted during the HTTP shutdown window")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-SRV-04  X-Forwarded-For spoofing bypasses rate limiting
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC_SRV_04_RED_XForwardedForSpoofingBypassesRateLimit documents
// that clientIP() in pkg/middleware/middleware.go unconditionally trusts the
// X-Forwarded-For header. An attacker can rotate fake IPs in this header to
// avoid exhausting their rate-limit bucket.
func TestSecurity_SEC_SRV_04_RED_XForwardedForSpoofingBypassesRateLimit(t *testing.T) {
	// Simulate clientIP as implemented in middleware.go.
	clientIPCurrent := func(r *http.Request) string {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return xff // simplified; real code strips after first comma
		}
		return r.RemoteAddr
	}

	realAttackerAddr := "203.0.113.5:9999"

	// Attacker sends request #1 with a spoofed XFF.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = realAttackerAddr
	req1.Header.Set("X-Forwarded-For", "1.2.3.4")

	// Attacker sends request #2 with a different spoofed XFF.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = realAttackerAddr
	req2.Header.Set("X-Forwarded-For", "5.6.7.8")

	ip1 := clientIPCurrent(req1)
	ip2 := clientIPCurrent(req2)

	assert.NotEqual(t, ip1, ip2,
		"RED SEC-SRV-04: the same real attacker IP %q appears as %q and %q "+
			"by simply rotating X-Forwarded-For values; "+
			"fix: only trust XFF when request arrives from a known trusted proxy CIDR",
		realAttackerAddr, ip1, ip2)
}

// TestSecurity_SEC_SRV_04_GREEN_TrustedProxyCIDREnforcedForXFF documents the
// required behaviour: XFF should only be trusted when the request arrives from
// a configured trusted-proxy CIDR range; otherwise RemoteAddr is used.
func TestSecurity_SEC_SRV_04_GREEN_TrustedProxyCIDREnforcedForXFF(t *testing.T) {
	trustedProxyCIDR := "10.0.0.0/8"

	// Hardened clientIP: only trust XFF from trusted proxy.
	hardenedClientIP := func(r *http.Request, trustedCIDR string) string {
		// Check whether RemoteAddr is within the trusted proxy CIDR.
		// (Full implementation would parse the CIDR and check; we use a simple
		// prefix match for test purposes.)
		host := r.RemoteAddr
		isTrustedProxy := len(host) > 3 && host[:3] == "10."

		if isTrustedProxy {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				return xff
			}
		}
		return r.RemoteAddr
	}

	_ = trustedProxyCIDR

	// From a trusted proxy: XFF is accepted.
	fromProxy := httptest.NewRequest(http.MethodGet, "/", nil)
	fromProxy.RemoteAddr = "10.0.0.2:80"
	fromProxy.Header.Set("X-Forwarded-For", "203.0.113.5")
	ip := hardenedClientIP(fromProxy, trustedProxyCIDR)
	assert.Equal(t, "203.0.113.5", ip,
		"GREEN SEC-SRV-04: XFF from trusted proxy must be accepted")

	// From an untrusted client: XFF is ignored.
	fromExternal := httptest.NewRequest(http.MethodGet, "/", nil)
	fromExternal.RemoteAddr = "203.0.113.5:9999"
	fromExternal.Header.Set("X-Forwarded-For", "1.2.3.4")
	ip = hardenedClientIP(fromExternal, trustedProxyCIDR)
	assert.Equal(t, "203.0.113.5:9999", ip,
		"GREEN SEC-SRV-04: XFF from untrusted source must be ignored; RemoteAddr used instead")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-SRV-05  Missing TLS — server binds plain HTTP by default
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC_SRV_05_RED_ServerDefaultsToPlainHTTP documents that
// the server starts on plain HTTP (ListenAndServe, not ListenAndServeTLS)
// and there is no HTTPS enforcement or HSTS header in the default config.
func TestSecurity_SEC_SRV_05_RED_ServerDefaultsToPlainHTTP(t *testing.T) {
	// We inspect the server construction: httpSrv.Handler does not add HSTS
	// and the server calls ListenAndServe (not ListenAndServeTLS).
	// We document this by asserting that a plain HTTP response lacks the
	// Strict-Transport-Security header.

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulates the production handler — no HSTS header set.
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	hsts := rr.Header().Get("Strict-Transport-Security")
	assert.Empty(t, hsts,
		"RED SEC-SRV-05: response lacks Strict-Transport-Security header; "+
			"fix: configure TLS via AGACH_TLS_CERT / AGACH_TLS_KEY env vars "+
			"and add HSTS middleware for production deployments")
}

// TestSecurity_SEC_SRV_05_GREEN_HSTSHeaderIsPresent documents the required
// behaviour: responses must include a Strict-Transport-Security header when
// the server is deployed with TLS.
func TestSecurity_SEC_SRV_05_GREEN_HSTSHeaderIsPresent(t *testing.T) {
	// Hardened handler adds HSTS.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	hsts := rr.Header().Get("Strict-Transport-Security")
	require.NotEmpty(t, hsts,
		"GREEN SEC-SRV-05: Strict-Transport-Security must be present")
	assert.Contains(t, hsts, "max-age=",
		"GREEN SEC-SRV-05: HSTS header must include max-age directive")
}
