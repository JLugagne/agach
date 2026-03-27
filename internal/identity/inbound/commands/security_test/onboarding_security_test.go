package security_test

// NEW security tests for onboarding and invite vulnerabilities.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/identity/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock types for onboarding tests
// ─────────────────────────────────────────────────────────────────────────────

type mockOnboardingCommands struct {
	generateCodeFunc        func(ctx context.Context, actor domain.Actor, mode domain.NodeMode, nodeName string) (domain.OnboardingCode, error)
	completeOnboardingFunc  func(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error)
}

func (m *mockOnboardingCommands) GenerateCode(ctx context.Context, actor domain.Actor, mode domain.NodeMode, nodeName string) (domain.OnboardingCode, error) {
	if m.generateCodeFunc != nil {
		return m.generateCodeFunc(ctx, actor, mode, nodeName)
	}
	return domain.OnboardingCode{}, nil
}

func (m *mockOnboardingCommands) CompleteOnboarding(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error) {
	if m.completeOnboardingFunc != nil {
		return m.completeOnboardingFunc(ctx, code, nodeName)
	}
	return "", "", domain.Node{}, nil
}

func newOnboardingTestHandler(onboarding service.OnboardingCommands, authCmds *mockAuthCommands, authQrs *mockAuthQueries) *mux.Router {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	ctrl := controller.NewController(logger)
	h := commands.NewOnboardingHandler(onboarding, authCmds, authQrs, ctrl)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-18: No rate limiting on POST /api/onboarding/complete
// File: internal/identity/inbound/commands/onboarding.go:40
//
// The onboarding code is a 6-digit numeric code (1,000,000 possibilities).
// The /api/onboarding/complete endpoint has no rate limiting, allowing an
// attacker to brute-force all possible codes in minutes and onboard a rogue
// daemon node under any user's account.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_OnboardingCompleteNoRateLimit documents that the
// /api/onboarding/complete endpoint can be called without rate limiting,
// enabling brute-force of 6-digit onboarding codes.
// TODO(security): Add rate limiting to /api/onboarding/complete (e.g., 5 attempts
// per minute per IP) to make brute-force of 6-digit codes infeasible.
func TestSecurity_RED_OnboardingCompleteNoRateLimit(t *testing.T) {
	attemptCount := 0
	onboarding := &mockOnboardingCommands{
		completeOnboardingFunc: func(_ context.Context, code string, _ string) (string, string, domain.Node, error) {
			attemptCount++
			return "", "", domain.Node{}, domain.ErrOnboardingCodeNotFound
		},
	}

	router := newOnboardingTestHandler(onboarding, &mockAuthCommands{}, &mockAuthQueries{})

	// Fire 50 rapid-fire brute force attempts from the same IP.
	// With a proper rate limiter, most should be rejected with 429.
	rateLimited := 0
	for i := 0; i < 50; i++ {
		body, _ := json.Marshal(map[string]string{
			"code": "000000",
		})
		req := httptest.NewRequest("POST", "/api/onboarding/complete", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			rateLimited++
		}
	}

	// RED: None of the requests are rate-limited today.
	assert.Greater(t, rateLimited, 0,
		"RED: /api/onboarding/complete has no rate limiting — 6-digit codes can be brute-forced "+
			"(onboarding.go:40)")
	t.Log("RED: 50 brute-force attempts against /api/onboarding/complete all succeed without rate limiting")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-19: No rate limiting on POST /api/daemon/refresh
// File: internal/identity/inbound/commands/onboarding.go:42
//
// The daemon refresh endpoint accepts a node_id and refresh_token with no
// rate limiting. An attacker can brute-force refresh tokens for known node IDs.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_DaemonRefreshNoRateLimit documents that the
// /api/daemon/refresh endpoint has no rate limiting.
// TODO(security): Add rate limiting to /api/daemon/refresh.
func TestSecurity_RED_DaemonRefreshNoRateLimit(t *testing.T) {
	authCmds := &mockAuthCommands{
		refreshDaemonTokenFunc: func(_ context.Context, _ domain.NodeID, _ string) (string, error) {
			return "", domain.ErrUnauthorized
		},
	}

	router := newOnboardingTestHandler(&mockOnboardingCommands{}, authCmds, &mockAuthQueries{})

	rateLimited := 0
	for i := 0; i < 50; i++ {
		body, _ := json.Marshal(map[string]string{
			"node_id":       "01934567-89ab-7cde-8f01-234567890abc",
			"refresh_token": "invalid-token",
		})
		req := httptest.NewRequest("POST", "/api/daemon/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			rateLimited++
		}
	}

	assert.Greater(t, rateLimited, 0,
		"RED: /api/daemon/refresh has no rate limiting — refresh tokens can be brute-forced "+
			"(onboarding.go:42)")
	t.Log("RED: 50 brute-force attempts against /api/daemon/refresh all succeed without rate limiting")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-20: No rate limiting on POST /api/auth/complete-invite
// File: internal/identity/inbound/commands/auth.go:63
//
// The complete-invite endpoint is not behind the auth rate limiter (unlike
// register and login). Invite tokens are JWTs so brute-force is not the
// concern, but the endpoint can be abused for credential-stuffing attacks
// by submitting many invite token guesses rapidly.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_CompleteInviteNoRateLimit documents that
// /api/auth/complete-invite has no rate limiting.
// TODO(security): Wrap /api/auth/complete-invite in the auth rate limiter.
func TestSecurity_RED_CompleteInviteNoRateLimit(t *testing.T) {
	authCmds := &mockAuthCommands{
		completeInviteFunc: func(_ context.Context, _, _, _ string) (domain.User, error) {
			return domain.User{}, domain.ErrUnauthorized
		},
		loginFunc: func(_ context.Context, _, _ string, _ bool) (string, string, error) {
			return "", "", domain.ErrInvalidCredentials
		},
	}

	_, router := newTestHandler(authCmds, &mockAuthQueries{})

	rateLimited := 0
	for i := 0; i < 50; i++ {
		body, _ := json.Marshal(map[string]string{
			"token":        "fake-invite-token",
			"display_name": "Attacker",
			"password":     "password123",
		})
		req := httptest.NewRequest("POST", "/api/auth/complete-invite", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			rateLimited++
		}
	}

	assert.Greater(t, rateLimited, 0,
		"RED: /api/auth/complete-invite has no rate limiting (auth.go:63)")
	t.Log("RED: 50 rapid requests to /api/auth/complete-invite all processed without rate limiting")
}
