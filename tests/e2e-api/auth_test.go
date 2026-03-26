package e2eapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuth_Login_Success(t *testing.T) {
	token := adminToken(t)

	resp := doAuth(t, "GET", "/api/auth/me", token, nil)
	requireStatus(t, resp, http.StatusOK)

	me := decode[struct {
		User publicUser `json:"user"`
	}](t, resp)
	require.Equal(t, "admin@agach.local", me.User.Email)
	require.NotEmpty(t, me.User.ID)
}

func TestAuth_Login_InvalidCredentials(t *testing.T) {
	resp, err := http.Post(apiURL("/api/auth/login"), "application/json",
		jsonBody(t, map[string]any{
			"email":    "admin@agach.local",
			"password": "wrongpassword",
		}))
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestAuth_Register_And_Login(t *testing.T) {
	email := fmt.Sprintf("register-%s@test.local", uniqueSlug("user"))
	password := "TestPassword123!"
	displayName := "Test User"

	// Register.
	resp, err := http.Post(apiURL("/api/auth/register"), "application/json",
		jsonBody(t, map[string]any{
			"email":        email,
			"password":     password,
			"display_name": displayName,
		}))
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusCreated)

	reg := decode[loginResponse](t, resp)
	require.NotEmpty(t, reg.AccessToken)
	require.Equal(t, email, reg.User.Email)
	require.Equal(t, displayName, reg.User.DisplayName)
	require.NotEmpty(t, reg.User.ID)

	// Verify user exists in DB.
	pool := testPool(t)
	require.True(t, rowExists(t, pool, "users", "email = $1", email))

	// Login with the newly created credentials.
	token := login(t, email, password)
	require.NotEmpty(t, token)
}

func TestAuth_Me(t *testing.T) {
	token := adminToken(t)

	me := getAndDecode[struct {
		User publicUser `json:"user"`
	}](t, "/api/auth/me", token)

	require.Equal(t, "admin@agach.local", me.User.Email)
	require.NotEmpty(t, me.User.ID)
	require.NotEmpty(t, me.User.DisplayName)
	require.NotEmpty(t, me.User.Role)
}

func TestAuth_UpdateProfile(t *testing.T) {
	// Register a dedicated user so we don't mutate the admin.
	email := fmt.Sprintf("profile-%s@test.local", uniqueSlug("user"))
	password := "TestPassword123!"

	resp, err := http.Post(apiURL("/api/auth/register"), "application/json",
		jsonBody(t, map[string]any{
			"email":        email,
			"password":     password,
			"display_name": "Before",
		}))
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusCreated)
	reg := decode[loginResponse](t, resp)
	token := reg.AccessToken

	// Patch display name.
	newName := "After Update"
	updated := patchAndDecode[struct {
		User publicUser `json:"user"`
	}](t, "/api/auth/me", token, map[string]any{
		"display_name": newName,
	})
	require.Equal(t, newName, updated.User.DisplayName)

	// Verify via GET /api/auth/me.
	me := getAndDecode[struct {
		User publicUser `json:"user"`
	}](t, "/api/auth/me", token)
	require.Equal(t, newName, me.User.DisplayName)
}

func TestAuth_ChangePassword(t *testing.T) {
	// Register a dedicated user.
	email := fmt.Sprintf("chpwd-%s@test.local", uniqueSlug("user"))
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword456!"

	resp, err := http.Post(apiURL("/api/auth/register"), "application/json",
		jsonBody(t, map[string]any{
			"email":        email,
			"password":     oldPassword,
			"display_name": "PwdUser",
		}))
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusCreated)
	reg := decode[loginResponse](t, resp)
	token := reg.AccessToken

	// Change password.
	resp = doAuth(t, "POST", "/api/auth/me/password", token, map[string]any{
		"current_password": oldPassword,
		"new_password":     newPassword,
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Old password should no longer work.
	resp, err = http.Post(apiURL("/api/auth/login"), "application/json",
		jsonBody(t, map[string]any{
			"email":    email,
			"password": oldPassword,
		}))
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	// New password should work.
	newToken := login(t, email, newPassword)
	require.NotEmpty(t, newToken)
}

func TestAuth_Logout(t *testing.T) {
	// Register a dedicated user with remember_me to get a refresh cookie.
	email := fmt.Sprintf("logout-%s@test.local", uniqueSlug("user"))
	password := "TestPassword123!"

	resp, err := http.Post(apiURL("/api/auth/register"), "application/json",
		jsonBody(t, map[string]any{
			"email":        email,
			"password":     password,
			"display_name": "LogoutUser",
		}))
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusCreated)
	reg := decode[loginResponse](t, resp)
	token := reg.AccessToken

	// Logout.
	resp = doAuth(t, "POST", "/api/auth/logout", token, nil)
	requireStatus(t, resp, http.StatusOK)

	// Check that the refresh_token cookie is cleared (MaxAge <= 0 or empty value).
	var refreshCleared bool
	for _, c := range resp.Cookies() {
		if c.Name == "refresh_token" {
			refreshCleared = c.MaxAge < 0 || c.Value == ""
		}
	}
	resp.Body.Close()
	require.True(t, refreshCleared, "refresh_token cookie should be cleared after logout")
}

func TestAuth_Unauthorized(t *testing.T) {
	// No token at all.
	resp, err := http.Get(apiURL("/api/auth/me"))
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	// Invalid token.
	resp = doAuth(t, "GET", "/api/auth/me", "invalid.token.here", nil)
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}
