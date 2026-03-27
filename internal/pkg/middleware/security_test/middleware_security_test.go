package security_test

// Security RED tests for the middleware package.
//
// These tests document security properties. Most of the original vulnerabilities
// (fake tokens, no crypto validation) are now fixed by NewRequireAuth which does
// real JWT/API-key validation. The remaining open items are noted below.
//
// Naming convention: TestSecurity_<VulnerabilityName>

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

// ─── Inlined helpers (from middleware_test.go) ───────────────────────────────

var errUnauthorized = errors.New("authentication required")

type testActor struct {
	Email string
}

type mockAuthQueries struct {
	validJWTs map[string]any
}

func (m *mockAuthQueries) ValidateJWT(_ context.Context, token string) (any, error) {
	if a, ok := m.validJWTs[token]; ok {
		return a, nil
	}
	return nil, errUnauthorized
}

var validActor = testActor{Email: "test@example.com"}

func newTestAuthMiddleware() http.Handler {
	mock := &mockAuthQueries{
		validJWTs: map[string]any{"valid-jwt": validActor},
	}
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return middleware.NewRequireAuth(mock)(okHandler)
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. RequireAuth rejects arbitrary (cryptographically invalid) tokens — FIXED
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RequireAuthRejectsFakeToken verifies that the auth middleware
// rejects tokens that are syntactically present but not in the mock auth store.
func TestSecurity_RequireAuthRejectsFakeToken(t *testing.T) {
	handler := newTestAuthMiddleware()

	fakeTokens := []struct {
		name  string
		value string
	}{
		{"random string", "Bearer this-is-not-a-valid-jwt"},
		{"literal word invalid", "invalid"},
		{"UUID not in store", "Bearer 00000000-0000-0000-0000-000000000000"},
		{"raw garbage", "aaaaaaaaaaaaaaaa"},
	}

	for _, tc := range fakeTokens {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/projects", nil)
			req.Header.Set("Authorization", tc.value)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code,
				"fake token %q must be rejected with 401", tc.value)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. RequireAuth does not distinguish admin from member role — RBAC still open
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RequireAuthDoesNotEnforceRBAC documents that role-based access
// control is not yet enforced at the middleware level.
func TestSecurity_RequireAuthDoesNotEnforceRBAC(t *testing.T) {
	t.Skip("RED: RBAC not yet enforced — fix: extract Actor role and check IsAdmin() in handlers")
}
