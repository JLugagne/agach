package e2eapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// env reads an environment variable or returns the fallback.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// baseURL returns the API base URL from E2E_BASE_URL or default.
func baseURL() string { return env("E2E_BASE_URL", "http://localhost:18322") }

// apiURL builds an absolute API URL.
func apiURL(path string) string { return baseURL() + path }

// ---------- JSON helpers ---------------------------------------------------

// apiResponse is the standard envelope: {"status":"success","data":{...}}
type apiResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var env apiResponse[T]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
	return env.Data
}

func decodeRaw(t *testing.T, resp *http.Response) json.RawMessage {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return body
}

// ---------- Auth helpers ---------------------------------------------------

type loginResponse struct {
	User        publicUser `json:"user"`
	AccessToken string     `json:"access_token"`
}

type publicUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// login authenticates and returns the access token.
func login(t *testing.T, email, password string) string {
	t.Helper()
	resp, err := http.Post(apiURL("/api/auth/login"), "application/json",
		jsonBody(t, map[string]any{"email": email, "password": password}))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "login failed for %s", email)
	lr := decode[loginResponse](t, resp)
	require.NotEmpty(t, lr.AccessToken)
	return lr.AccessToken
}

// adminToken logs in as admin and returns the token.
func adminToken(t *testing.T) string {
	t.Helper()
	return login(t, "admin@agach.local", "admin")
}

// authReq builds an authenticated request.
func authReq(t *testing.T, method, url, token string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// doAuth performs an authenticated request and returns the response.
func doAuth(t *testing.T, method, path, token string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		r = jsonBody(t, body)
	}
	req := authReq(t, method, apiURL(path), token, r)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// requireStatus asserts the HTTP status and prints the body on failure.
func requireStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// ---------- DB helpers -----------------------------------------------------

func dbConnStr() string {
	return env("E2E_DATABASE_URL", "postgres://agach:agach@localhost:15432/agach?sslmode=disable")
}

// ---------- Generic CRUD helpers -------------------------------------------

// createAndDecode does a POST, asserts 201, and decodes the response data.
func createAndDecode[T any](t *testing.T, path, token string, body any) T {
	t.Helper()
	resp := doAuth(t, "POST", path, token, body)
	requireStatus(t, resp, http.StatusCreated)
	return decode[T](t, resp)
}

// getAndDecode does a GET, asserts 200, and decodes the response data.
func getAndDecode[T any](t *testing.T, path, token string) T {
	t.Helper()
	resp := doAuth(t, "GET", path, token, nil)
	requireStatus(t, resp, http.StatusOK)
	return decode[T](t, resp)
}

// patchAndDecode does a PATCH, asserts 200, and decodes the response data.
func patchAndDecode[T any](t *testing.T, path, token string, body any) T {
	t.Helper()
	resp := doAuth(t, "PATCH", path, token, body)
	requireStatus(t, resp, http.StatusOK)
	return decode[T](t, resp)
}

// deleteResource does a DELETE and asserts 204.
func deleteResource(t *testing.T, path, token string) {
	t.Helper()
	resp := doAuth(t, "DELETE", path, token, nil)
	requireStatus(t, resp, http.StatusNoContent)
}

// deleteResourceWithBody does a DELETE with a JSON body and asserts 204.
func deleteResourceWithBody(t *testing.T, path, token string, body any) {
	t.Helper()
	resp := doAuth(t, "DELETE", path, token, body)
	requireStatus(t, resp, http.StatusNoContent)
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }

// uniqueSlug returns a slug with a random suffix for test isolation.
var slugCounter int

func uniqueSlug(prefix string) string {
	slugCounter++
	return fmt.Sprintf("%s-%d", prefix, slugCounter)
}
