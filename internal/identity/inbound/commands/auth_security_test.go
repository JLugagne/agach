package commands_test

// Security tests for internal/identity/inbound/commands/auth.go
//
// Each vulnerability has:
//   - a RED test: demonstrates the bug. The assertion checks the *insecure*
//     behaviour, so it currently PASSES (showing the bug is present).
//   - a GREEN test: asserts the correct, secure behaviour. This will FAIL
//     against the current production code and must PASS after the fix.
//
// Mock types (mockAuthCommands, mockAuthQueries, newTestHandler) are declared
// in auth_handler_test.go in the same package (commands_test).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-1  auth.go:407–414
// IP spoofing via X-Forwarded-For bypasses rate limiting.
//
// clientIPFromRequest trusts the first value in X-Forwarded-For without any
// trusted-proxy allowlist. Rotating fake IPs grants a fresh token-bucket per
// request, effectively disabling the per-IP limiter.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — after fixing trusted-proxy validation, the same RemoteAddr must be
// rate-limited regardless of the forged X-Forwarded-For value.
func TestSecurity_GREEN_RateLimitBypass_XForwardedFor_SameRemoteAddr(t *testing.T) {
	cmds := &mockAuthCommands{
		loginFunc: func(_ context.Context, _, _ string, _ bool) (string, string, error) {
			return "", "", domain.ErrInvalidCredentials
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	body, _ := json.Marshal(map[string]string{"email": "x@x.com", "password": "badpassword"})

	var codes []int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.0.0.%d", i+1)) // rotating fake IP
		req.RemoteAddr = "192.168.1.1:1234"                               // same real IP
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		codes = append(codes, rr.Code)
	}

	assert.Equal(t, http.StatusTooManyRequests, codes[5],
		"GREEN: 6th request from same RemoteAddr must be rate-limited even with rotating X-Forwarded-For")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-2  controller.go:127
// No request body size limit — enables DoS via large payloads.
//
// DecodeAndValidate uses json.NewDecoder(r.Body).Decode without wrapping the
// body in http.MaxBytesReader, so arbitrarily large bodies are fully read.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — after adding a body size limit, a 1 MB body must be rejected with 413.
func TestSecurity_GREEN_NoBodySizeLimit_LargePayloadRejected(t *testing.T) {
	loginCalled := false
	cmds := &mockAuthCommands{
		loginFunc: func(_ context.Context, _, _ string, _ bool) (string, string, error) {
			loginCalled = true
			return "", "", domain.ErrInvalidCredentials
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	garbage := make([]byte, 1024*1024)
	for i := range garbage {
		garbage[i] = 'a'
	}
	payload := fmt.Sprintf(`{"email":"x@x.com","password":"badpassword","extra":%q}`, string(garbage))

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code,
		"GREEN: server must respond 413 for oversized request body")
	assert.False(t, loginCalled,
		"GREEN: service must NOT be called for oversized body")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-3  auth.go:50–51
// No rate limiting on POST /api/auth/apikeys (API key creation).
//
// /api/auth/register and /api/auth/login are wrapped with authLimiter.middleware
// but /api/auth/apikeys is registered with a plain HandleFunc. An authenticated
// attacker can create API keys in a tight loop without being throttled.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — after applying rate limiting, the endpoint must throttle after burst.
func TestSecurity_GREEN_RateLimit_CreateAPIKey_ThrottledAfterBurst(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}
	cmds := &mockAuthCommands{
		createAPIKeyFunc: func(_ context.Context, _ domain.Actor, _ string, _ []string, _ *time.Time) (domain.APIKey, string, error) {
			return domain.APIKey{ID: domain.NewAPIKeyID(), Name: "k", Scopes: []string{}}, "raw-key", nil
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}
	_, router := newTestHandler(cmds, qrs)

	body := []byte(`{"name":"test-key","scopes":[]}`)

	got429 := false
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("POST", "/api/auth/apikeys", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer valid-token")
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}

	assert.True(t, got429,
		"GREEN: CreateAPIKey must be rate-limited after burst is exceeded")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-4  auth.go:308–313
// X-Forwarded-Proto is trusted unconditionally for the Secure cookie flag.
//
// isSecure() sets Secure=true when X-Forwarded-Proto == "https" even if the
// connection has no TLS (req.TLS is nil) and there is no trusted-proxy check.
// A client on a plain HTTP path can forge this header to get a Secure-flagged
// cookie, or, more critically, the server may issue non-Secure cookies on a
// real TLS deployment behind a proxy that does not set this header.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — without a configured trusted-proxy allowlist, a forged
//
//	X-Forwarded-Proto must NOT affect the Secure flag.
func TestSecurity_GREEN_XForwardedProto_NotTrustedWithoutConfig(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "u@example.com", Role: domain.RoleMember}
	user := domain.User{ID: actor.UserID, Email: actor.Email}
	cmds := &mockAuthCommands{
		loginFunc: func(_ context.Context, _, _ string, _ bool) (string, string, error) {
			return "access-token", "refresh-token", nil
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc:    func(_ context.Context, _ string) (domain.Actor, error) { return actor, nil },
		getCurrentUserFunc: func(_ context.Context, _ domain.Actor) (domain.User, error) { return user, nil },
	}
	_, router := newTestHandler(cmds, qrs)

	body, _ := json.Marshal(map[string]string{"email": "u@example.com", "password": "password123"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https") // forged, no TLS
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var refreshCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	require.NotNil(t, refreshCookie)

	// GREEN: Secure must NOT be set when req.TLS is nil and the proxy is untrusted.
	assert.False(t, refreshCookie.Secure,
		"GREEN: Secure flag must NOT be set from an untrusted X-Forwarded-Proto on a plain HTTP connection")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-5  auth.go:355–361
// Excessive data exposure — revoked API keys returned in default listing.
//
// toPublicAPIKeys does not filter out keys whose RevokedAt is non-nil.
// Every key the user ever created (including revoked ones) is returned,
// leaking key metadata (name, scopes, timestamps) that should be hidden
// after revocation.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — after fixing the handler/service, only non-revoked keys must appear.
func TestSecurity_GREEN_ExcessiveDataExposure_RevokedKeysFiltered(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}
	revokedAt := time.Now().Add(-24 * time.Hour)
	mixedKeys := []domain.APIKey{
		{ID: domain.NewAPIKeyID(), Name: "old-key", Scopes: []string{"server:read"}, RevokedAt: &revokedAt},
		{ID: domain.NewAPIKeyID(), Name: "active-key", Scopes: []string{"server:write"}},
	}

	cmds := &mockAuthCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) { return actor, nil },
		listAPIKeysFunc: func(_ context.Context, _ domain.Actor) ([]domain.APIKey, error) {
			return mixedKeys, nil
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("GET", "/api/auth/apikeys", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	data, _ := resp["data"].([]interface{})

	// GREEN: only the active key must be present.
	require.Len(t, data, 1,
		"GREEN: only active (non-revoked) keys must appear in the default listing")
	key, _ := data[0].(map[string]interface{})
	assert.Equal(t, "active-key", key["name"])
}
