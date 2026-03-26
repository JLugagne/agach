package e2eapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuth runs all auth tests sequentially within a single test to stay
// within the per-IP rate limit (5 burst / 15 min) on auth endpoints.
func TestAuth(t *testing.T) {
	ensureServer(t)

	t.Run("Login_Success", func(t *testing.T) {
		token := adminToken(t)

		resp := doAuth(t, "GET", "/api/auth/me", token, nil)
		requireStatus(t, resp, http.StatusOK)

		me := decode[publicUser](t, resp)
		assert.Equal(t, "admin@agach.local", me.Email)
		require.NotEmpty(t, me.ID)
	})

	t.Run("Login_InvalidCredentials", func(t *testing.T) {
		resp, err := http.Post(apiURL("/api/auth/login"), "application/json",
			jsonBody(t, map[string]any{
				"email":    "admin@agach.local",
				"password": "wrongpassword",
			}))
		require.NoError(t, err)
		requireStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})

	// Register a user reused by subsequent subtests.
	var userToken string
	var userEmail = "e2e-auth@test.local"
	var userPassword = "TestPassword123!"

	t.Run("Register", func(t *testing.T) {
		resp, err := http.Post(apiURL("/api/auth/register"), "application/json",
			jsonBody(t, map[string]any{
				"email":        userEmail,
				"password":     userPassword,
				"display_name": "E2E Auth User",
			}))
		require.NoError(t, err)
		requireStatus(t, resp, http.StatusOK)

		reg := decode[loginResponse](t, resp)
		require.NotEmpty(t, reg.AccessToken)
		assert.Equal(t, userEmail, reg.User.Email)
		userToken = reg.AccessToken

		// Verify in DB.
		pool := testPool(t)
		require.True(t, rowExists(t, pool, "users", "email = $1", userEmail))
	})

	t.Run("Me", func(t *testing.T) {
		require.NotEmpty(t, userToken)
		me := getAndDecode[publicUser](t, "/api/auth/me", userToken)
		assert.Equal(t, userEmail, me.Email)
		require.NotEmpty(t, me.ID)
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		require.NotEmpty(t, userToken)
		updated := patchAndDecode[publicUser](t, "/api/auth/me", userToken, map[string]any{
			"display_name": "Updated Name",
		})
		assert.Equal(t, "Updated Name", updated.DisplayName)
	})

	t.Run("ChangePassword", func(t *testing.T) {
		require.NotEmpty(t, userToken)
		newPassword := "NewPassword456!"

		resp := doAuth(t, "POST", "/api/auth/me/password", userToken, map[string]any{
			"current_password": userPassword,
			"new_password":     newPassword,
		})
		requireStatus(t, resp, http.StatusNoContent)
		resp.Body.Close()

		// Login with new password (uses the token obtained from the register burst).
		newToken := login(t, userEmail, newPassword)
		require.NotEmpty(t, newToken)
		userPassword = newPassword
		userToken = newToken
	})

	t.Run("Refresh", func(t *testing.T) {
		// Login and capture the refresh_token cookie from the response.
		resp, err := http.Post(apiURL("/api/auth/login"), "application/json",
			jsonBody(t, map[string]any{
				"email":    userEmail,
				"password": userPassword,
			}))
		require.NoError(t, err)
		requireStatus(t, resp, http.StatusOK)

		var refreshTokenValue string
		for _, c := range resp.Cookies() {
			if c.Name == "refresh_token" {
				refreshTokenValue = c.Value
			}
		}
		resp.Body.Close()
		require.NotEmpty(t, refreshTokenValue, "login should set a refresh_token cookie")

		// POST /api/auth/refresh with the cookie set manually.
		req, err := http.NewRequest("POST", apiURL("/api/auth/refresh"), nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshTokenValue})

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		requireStatus(t, resp, http.StatusOK)

		type refreshResp struct {
			AccessToken string `json:"access_token"`
		}
		rr := decode[refreshResp](t, resp)
		require.NotEmpty(t, rr.AccessToken, "refresh should return a new access token")

		// The new token should be usable.
		meReq, _ := http.NewRequest("GET", apiURL("/api/auth/me"), nil)
		meReq.Header.Set("Authorization", "Bearer "+rr.AccessToken)
		meResp, err := http.DefaultClient.Do(meReq)
		require.NoError(t, err)
		requireStatus(t, meResp, http.StatusOK)
		me := decode[publicUser](t, meResp)
		assert.Equal(t, userEmail, me.Email)
	})

	t.Run("Refresh_NoCookie", func(t *testing.T) {
		// Without a refresh_token cookie, should get 401.
		resp, err := http.Post(apiURL("/api/auth/refresh"), "application/json", nil)
		require.NoError(t, err)
		requireStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})

	t.Run("Logout", func(t *testing.T) {
		require.NotEmpty(t, userToken)
		resp := doAuth(t, "POST", "/api/auth/logout", userToken, nil)
		requireStatus(t, resp, http.StatusOK)

		var refreshCleared bool
		for _, c := range resp.Cookies() {
			if c.Name == "refresh_token" {
				refreshCleared = c.MaxAge < 0 || c.Value == ""
			}
		}
		resp.Body.Close()
		assert.True(t, refreshCleared, "refresh_token cookie should be cleared after logout")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		resp, err := http.Get(apiURL("/api/auth/me"))
		require.NoError(t, err)
		requireStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()

		resp = doAuth(t, "GET", "/api/auth/me", "invalid.token.here", nil)
		requireStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})
}
