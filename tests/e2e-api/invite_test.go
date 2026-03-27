package e2eapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Inline types
// ---------------------------------------------------------------------------

type inviteResponse struct {
	InviteToken string `json:"invite_token"`
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestInvite_FullWorkflow(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)
	pool := testPool(t)

	email := fmt.Sprintf("invited-%d@test.local", slugCounter+100)

	// 1. Admin invites a user by email.
	resp := doAuth(t, "POST", "/api/identity/users/invite", token, map[string]any{
		"email": email,
	})
	requireStatus(t, resp, http.StatusOK)
	inv := decode[inviteResponse](t, resp)
	require.NotEmpty(t, inv.InviteToken, "should return an invite token")

	// 2. Verify user was created in DB with empty password.
	require.True(t, rowExists(t, pool, "users", "email = $1", email),
		"invited user should exist in DB")
	pwHash := queryString(t, pool,
		"SELECT COALESCE(pgp_sym_decrypt(password_hash, $2)::text, '') FROM users WHERE email = $1",
		email, "e2e-test-secret-at-least-32-bytes!")
	assert.Empty(t, pwHash, "invited user should have empty password hash")

	// 3. Verify user appears in the admin user list.
	users := listUsers(t, token)
	var found bool
	for _, u := range users {
		if u.Email == email {
			found = true
			assert.Equal(t, "member", u.Role)
			assert.Empty(t, u.DisplayName)
			break
		}
	}
	require.True(t, found, "invited user should appear in user list")

	// 4. Complete the invite (public endpoint, no auth).
	completeResp, err := http.Post(apiURL("/api/auth/complete-invite"), "application/json",
		jsonBody(t, map[string]any{
			"token":        inv.InviteToken,
			"display_name": "Invited User",
			"password":     "SecurePass123!",
		}))
	require.NoError(t, err)
	requireStatus(t, completeResp, http.StatusOK)

	lr := decode[loginResponse](t, completeResp)
	require.NotEmpty(t, lr.AccessToken, "should return access token after completing invite")
	assert.Equal(t, email, lr.User.Email)
	assert.Equal(t, "Invited User", lr.User.DisplayName)
	assert.Equal(t, "member", lr.User.Role)

	// 5. The new user can use their token.
	me := getAndDecode[publicUser](t, "/api/auth/me", lr.AccessToken)
	assert.Equal(t, email, me.Email)
	assert.Equal(t, "Invited User", me.DisplayName)

	// 6. The new user can login with their password.
	newToken := login(t, email, "SecurePass123!")
	require.NotEmpty(t, newToken)
}

func TestInvite_CompleteTwiceFails(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	email := fmt.Sprintf("invite-twice-%d@test.local", slugCounter+200)

	// Invite.
	inv := createAndDecode[inviteResponse](t, "/api/identity/users/invite", token, map[string]any{
		"email": email,
	})
	require.NotEmpty(t, inv.InviteToken)

	// Complete once.
	completeResp, err := http.Post(apiURL("/api/auth/complete-invite"), "application/json",
		jsonBody(t, map[string]any{
			"token":        inv.InviteToken,
			"display_name": "First Complete",
			"password":     "SecurePass123!",
		}))
	require.NoError(t, err)
	requireStatus(t, completeResp, http.StatusOK)
	completeResp.Body.Close()

	// Complete again with the same token — should fail.
	resp2, err := http.Post(apiURL("/api/auth/complete-invite"), "application/json",
		jsonBody(t, map[string]any{
			"token":        inv.InviteToken,
			"display_name": "Second Attempt",
			"password":     "AnotherPass123!",
		}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode,
		"completing an already-used invite should fail")
	resp2.Body.Close()
}

func TestInvite_InvalidToken(t *testing.T) {
	ensureServer(t)

	resp, err := http.Post(apiURL("/api/auth/complete-invite"), "application/json",
		jsonBody(t, map[string]any{
			"token":        "invalid.jwt.token",
			"display_name": "Hacker",
			"password":     "Password123!",
		}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"invalid token should be rejected")
	resp.Body.Close()
}

func TestInvite_PasswordTooShort(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	email := fmt.Sprintf("invite-short-%d@test.local", slugCounter+300)

	inv := createAndDecode[inviteResponse](t, "/api/identity/users/invite", token, map[string]any{
		"email": email,
	})

	resp, err := http.Post(apiURL("/api/auth/complete-invite"), "application/json",
		jsonBody(t, map[string]any{
			"token":        inv.InviteToken,
			"display_name": "Short Pass",
			"password":     "short",
		}))
	require.NoError(t, err)
	// Validation rejects at handler level (min=8) or service level.
	assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity,
		"short password should be rejected, got %d", resp.StatusCode)
	resp.Body.Close()
}

func TestInvite_DuplicateEmailFails(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	// admin@agach.local already exists.
	resp := doAuth(t, "POST", "/api/identity/users/invite", token, map[string]any{
		"email": "admin@agach.local",
	})
	assert.Equal(t, http.StatusConflict, resp.StatusCode,
		"inviting an existing email should return 409")
	resp.Body.Close()
}

func TestInvite_NonAdminForbidden(t *testing.T) {
	ensureServer(t)

	// Register a regular member user.
	email := fmt.Sprintf("member-invite-%d@test.local", slugCounter+400)
	regResp, err := http.Post(apiURL("/api/auth/register"), "application/json",
		jsonBody(t, map[string]any{
			"email":        email,
			"password":     "MemberPass123!",
			"display_name": "Member User",
		}))
	require.NoError(t, err)
	requireStatus(t, regResp, http.StatusOK)
	lr := decode[loginResponse](t, regResp)

	// Try to invite as member — should be forbidden.
	resp := doAuth(t, "POST", "/api/identity/users/invite", lr.AccessToken, map[string]any{
		"email": "someone@test.local",
	})
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"non-admin should not be able to invite users")
	resp.Body.Close()
}
