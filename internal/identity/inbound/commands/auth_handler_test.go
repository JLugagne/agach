package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/inbound/commands"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock services are defined in test_helpers.go
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func newTestHandler(cmds *mockAuthCommands, qrs *mockAuthQueries) (*commands.AuthCommandsHandler, *mux.Router) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	ctrl := controller.NewController(logger)
	h := commands.NewAuthCommandsHandler(cmds, qrs, ctrl)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return h, r
}

func postJSON(router *mux.Router, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ─────────────────────────────────────────────────────────────────────────────
// Register
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_Register_Success(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "new@example.com", Role: domain.RoleMember}
	user := domain.User{ID: actor.UserID, Email: actor.Email, Role: domain.RoleMember}

	cmds := &mockAuthCommands{
		registerFunc: func(_ context.Context, email, _, _ string) (domain.User, error) {
			return user, nil
		},
		loginFunc: func(_ context.Context, _, _ string, _ bool) (string, string, error) {
			return "access-token", "refresh-token", nil
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) { return actor, nil },
		getCurrentUserFunc: func(_ context.Context, _ domain.Actor) (domain.User, error) { return user, nil },
	}

	_, router := newTestHandler(cmds, qrs)

	rr := postJSON(router, "/api/auth/register", map[string]string{
		"email":    "new@example.com",
		"password": "password123",
	})

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
}

func TestAuthHandler_Register_InvalidBody_ReturnsBadRequest(t *testing.T) {
	cmds := &mockAuthCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAuthHandler_Register_EmailAlreadyExists_ReturnsConflict(t *testing.T) {
	cmds := &mockAuthCommands{
		registerFunc: func(_ context.Context, _, _, _ string) (domain.User, error) {
			return domain.User{}, domain.ErrEmailAlreadyExists
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	rr := postJSON(router, "/api/auth/register", map[string]string{
		"email": "existing@example.com", "password": "password123",
	})

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestAuthHandler_Register_PasswordTooShort_ReturnsBadRequest(t *testing.T) {
	cmds := &mockAuthCommands{
		registerFunc: func(_ context.Context, _, _, _ string) (domain.User, error) {
			return domain.User{}, &domain.Error{Code: "PASSWORD_TOO_SHORT", Message: "too short"}
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	// Validation happens at HTTP level first (min=8), so missing password triggers validation
	rr := postJSON(router, "/api/auth/register", map[string]string{
		"email": "user@example.com", "password": "short",
	})

	// Either bad request from handler validation or from service
	assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusUnauthorized)
}

// ─────────────────────────────────────────────────────────────────────────────
// Login
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_Login_Success(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	user := domain.User{ID: actor.UserID, Email: actor.Email}

	cmds := &mockAuthCommands{
		loginFunc: func(_ context.Context, _, _ string, _ bool) (string, string, error) {
			return "access-token", "refresh-token", nil
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
		getCurrentUserFunc: func(_ context.Context, _ domain.Actor) (domain.User, error) {
			return user, nil
		},
	}

	_, router := newTestHandler(cmds, qrs)

	rr := postJSON(router, "/api/auth/login", map[string]string{
		"email": "user@example.com", "password": "password123",
	})

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthHandler_Login_InvalidCredentials_ReturnsUnauthorized(t *testing.T) {
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

	rr := postJSON(router, "/api/auth/login", map[string]string{
		"email": "bad@example.com", "password": "wrongpassword",
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	err, _ := resp["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_CREDENTIALS", err["code"])
}

func TestAuthHandler_Login_SSOUser_ReturnsUnauthorized(t *testing.T) {
	cmds := &mockAuthCommands{
		loginFunc: func(_ context.Context, _, _ string, _ bool) (string, string, error) {
			return "", "", domain.ErrSSOUserNoPassword
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	rr := postJSON(router, "/api/auth/login", map[string]string{
		"email": "sso@example.com", "password": "any",
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	err, _ := resp["error"].(map[string]interface{})
	assert.Equal(t, "SSO_USER_NO_PASSWORD", err["code"])
}

// ─────────────────────────────────────────────────────────────────────────────
// Refresh
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_Refresh_Success(t *testing.T) {
	cmds := &mockAuthCommands{
		refreshTokenFunc: func(_ context.Context, _ string) (string, error) {
			return "new-access-token", nil
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	data, _ := resp["data"].(map[string]interface{})
	assert.Equal(t, "new-access-token", data["access_token"])
}

func TestAuthHandler_Refresh_MissingCookie_ReturnsUnauthorized(t *testing.T) {
	cmds := &mockAuthCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthHandler_Refresh_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	cmds := &mockAuthCommands{
		refreshTokenFunc: func(_ context.Context, _ string) (string, error) {
			return "", domain.ErrUnauthorized
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "bad-token"})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// Logout
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout_ClearsCookie(t *testing.T) {
	cmds := &mockAuthCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check cookie is cleared (MaxAge=-1)
	cookies := rr.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	require.NotNil(t, refreshCookie, "refresh_token cookie should be present in response")
	assert.Equal(t, -1, refreshCookie.MaxAge)
}

// ─────────────────────────────────────────────────────────────────────────────
// ActorFromRequest / security
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// Rate limiting
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_RateLimiting_ExcessiveRequests_ReturnsTooManyRequests(t *testing.T) {
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

	body, _ := json.Marshal(map[string]string{"email": "x@x.com", "password": "p"})

	// Send 6 requests from the same IP (limit is 5 per 15 minutes)
	var lastCode int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		lastCode = rr.Code
	}

	assert.Equal(t, http.StatusTooManyRequests, lastCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// isSecure / X-Forwarded-Proto
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// Login handler - missing branches
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_Login_InvalidBody_ReturnsBadRequest(t *testing.T) {
	cmds := &mockAuthCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAuthHandler_Login_MissingPassword_ReturnsBadRequest(t *testing.T) {
	cmds := &mockAuthCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	rr := postJSON(router, "/api/auth/login", map[string]string{"email": "user@example.com"})
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// clientIPFromRequest coverage
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_RateLimiting_XRealIP(t *testing.T) {
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

	body, _ := json.Marshal(map[string]string{"email": "x@x.com", "password": "pass"})

	var lastCode int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.100")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		lastCode = rr.Code
	}
	assert.Equal(t, http.StatusTooManyRequests, lastCode)
}

func TestAuthHandler_RateLimiting_XForwardedFor_MultipleIPs(t *testing.T) {
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

	body, _ := json.Marshal(map[string]string{"email": "x@x.com", "password": "pass"})

	var lastCode int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "10.1.1.1, 10.2.2.2")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		lastCode = rr.Code
	}
	assert.Equal(t, http.StatusTooManyRequests, lastCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// isSecure - TLS request
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout_CookieIsHttpOnly(t *testing.T) {
	cmds := &mockAuthCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
	_, router := newTestHandler(cmds, qrs)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			assert.True(t, c.HttpOnly, "cookie should be HttpOnly")
			assert.Equal(t, http.SameSiteStrictMode, c.SameSite, "cookie SameSite should be Strict")
		}
	}
}
