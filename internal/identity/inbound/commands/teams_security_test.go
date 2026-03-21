package commands_test

// Security tests for internal/identity/inbound/commands/teams.go
//
// Each vulnerability has:
//   - a RED test: demonstrates the insecure behaviour (currently passes).
//   - a GREEN test: asserts the correct, secure behaviour (fails until fixed).
//
// Mock types are declared in teams_handler_test.go and auth_handler_test.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-6  teams.go:53
// GET /api/identity/teams — missing authentication.
//
// ListTeams calls h.queries.ListTeams directly without any auth check.
// Any unauthenticated caller can enumerate all teams.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — after adding an auth guard, unauthenticated requests must be rejected.
func TestSecurity_GREEN_ListTeams_RequiresAuthentication(t *testing.T) {
	qrs := &mockTeamQueries{
		listTeamsFunc: func(_ context.Context) ([]domain.Team, error) {
			return []domain.Team{{ID: domain.NewTeamID(), Name: "Engineering", Slug: "eng"}}, nil
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, unauthorizedAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/teams", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// GREEN: unauthenticated request must be rejected.
	assert.Equal(t, http.StatusUnauthorized, rr.Code,
		"GREEN: GET /api/identity/teams must require authentication")
}

// GREEN — an authenticated request must still work normally.
func TestSecurity_GREEN_ListTeams_AuthenticatedSucceeds(t *testing.T) {
	teams := []domain.Team{
		{ID: domain.NewTeamID(), Name: "Engineering", Slug: "engineering"},
	}

	qrs := &mockTeamQueries{
		listTeamsFunc: func(_ context.Context) ([]domain.Team, error) {
			return teams, nil
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, adminAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/teams", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		"GREEN: authenticated GET /api/identity/teams must succeed")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-7  teams.go:132
// GET /api/identity/users — missing authentication.
//
// ListUsers calls h.queries.ListUsers directly without any auth check,
// exposing every user's email, role, and team membership to anonymous callers.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — after adding an auth guard, unauthenticated requests must be rejected.
func TestSecurity_GREEN_ListUsers_RequiresAuthentication(t *testing.T) {
	qrs := &mockTeamQueries{
		listUsersFunc: func(_ context.Context) ([]domain.User, error) {
			return []domain.User{
				{ID: domain.NewUserID(), Email: "alice@example.com"},
			}, nil
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, unauthorizedAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/users", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code,
		"GREEN: GET /api/identity/users must require authentication")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-8  teams.go:238–242
// Bearer token extraction fallback — raw header value used as token.
//
// TeamsHandler.actorFromRequest strips "Bearer " only when the prefix is
// exactly present. When the caller omits the prefix, the full raw
// Authorization header value is passed directly to ValidateJWT. This differs
// from AuthCommandsHandler (which uses strings.TrimPrefix — same behaviour, but
// explicit) and may silently accept malformed credentials depending on the JWT
// library.
//
// The fix should reject any Authorization header that does not begin with
// "Bearer " (case-insensitive or strict) instead of silently falling through.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — after enforcing the "Bearer " prefix, a header without it must be
// rejected with 401 before calling ValidateJWT.
func TestSecurity_GREEN_TeamsHandler_BearerPrefixEnforced(t *testing.T) {
	validateCalled := false
	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			validateCalled = true
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, authQrs)

	body, _ := json.Marshal(map[string]string{"name": "Eng", "slug": "eng"})
	req := httptest.NewRequest("POST", "/api/identity/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "rawtoken123") // no "Bearer " prefix
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// GREEN: request must be rejected without calling ValidateJWT.
	assert.Equal(t, http.StatusUnauthorized, rr.Code,
		"GREEN: Authorization header without 'Bearer ' prefix must be rejected with 401")
	assert.False(t, validateCalled,
		"GREEN: ValidateJWT must NOT be called for a malformed Authorization header")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-9  teams.go:291–306
// Excessive data exposure in GET /api/identity/users — email addresses
// are returned to any authenticated user, not just admins.
//
// userToPublicMap includes the user's email unconditionally. In a system where
// only admins should manage teams, regular members should not be able to
// harvest the full email list.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN — a member-role user must not receive email addresses in the user list.
func TestSecurity_GREEN_ListUsers_EmailHiddenFromMembers(t *testing.T) {
	users := []domain.User{
		{ID: domain.NewUserID(), Email: "alice@example.com", Role: domain.RoleAdmin},
		{ID: domain.NewUserID(), Email: "bob@example.com", Role: domain.RoleMember},
	}

	qrs := &mockTeamQueries{
		listUsersFunc: func(_ context.Context) ([]domain.User, error) {
			return users, nil
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, memberAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/users", nil)
	req.Header.Set("Authorization", "Bearer member-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		// If the auth guard (VULN-7 fix) blocks members entirely, that also
		// satisfies this requirement — accept 401/403 as a valid fix path.
		assert.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, rr.Code,
			"GREEN: member-role user must be blocked from the user list endpoint")
		return
	}

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	data, _ := resp["data"].([]interface{})
	for _, item := range data {
		u, _ := item.(map[string]interface{})
		_, emailPresent := u["email"]
		assert.False(t, emailPresent,
			"GREEN: email field must not be returned to member-role callers")
	}
}
