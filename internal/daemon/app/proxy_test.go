package app_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/app"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func proxyClient(socketPath, apiKey string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}
}

func doProxyReq(t *testing.T, client *http.Client, method, path, apiKey string, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, "http://localhost"+path, body)
	require.NoError(t, err)
	req.Header.Set("X-Api-Key", apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func TestProxy_RewritesProjectID(t *testing.T) {
	var captured struct {
		path  string
		auth  string
		apiKeyForwarded string
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.path = r.URL.Path
		captured.auth = r.Header.Get("Authorization")
		captured.apiKeyForwarded = r.Header.Get("X-Api-Key")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"message": "ok"}})
	}))
	defer backend.Close()

	socketPath := filepath.Join(t.TempDir(), "proxy.sock")
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	proxy := app.NewSidecarProxy(
		socketPath, "test-key", "proj-42", "feat-7", backend.URL,
		func() string { return "my-jwt-token" },
		func() error { return nil },
		logger,
	)
	require.NoError(t, proxy.Start(context.Background()))
	defer proxy.Stop()

	client := proxyClient(socketPath, "test-key")
	resp := doProxyReq(t, client, http.MethodGet, "/api/projects/_/tasks", "test-key", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/api/projects/proj-42/tasks", captured.path)
	assert.Equal(t, "Bearer my-jwt-token", captured.auth)
	assert.Equal(t, "", captured.apiKeyForwarded, "X-Api-Key should be stripped before forwarding")
}

func TestProxy_RewritesFeatureID(t *testing.T) {
	var capturedPath string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"message": "ok"}})
	}))
	defer backend.Close()

	socketPath := filepath.Join(t.TempDir(), "proxy.sock")
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	proxy := app.NewSidecarProxy(
		socketPath, "key", "proj-1", "feat-99", backend.URL,
		func() string { return "tok" },
		func() error { return nil },
		logger,
	)
	require.NoError(t, proxy.Start(context.Background()))
	defer proxy.Stop()

	client := proxyClient(socketPath, "key")
	resp := doProxyReq(t, client, http.MethodPatch, "/api/projects/_/features/_/changelogs", "key", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/api/projects/proj-1/features/feat-99/changelogs", capturedPath)
}

func TestProxy_InjectsFeatureID_OnTaskCreation(t *testing.T) {
	var capturedBody map[string]any

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"id": "t1"}})
	}))
	defer backend.Close()

	socketPath := filepath.Join(t.TempDir(), "proxy.sock")
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	proxy := app.NewSidecarProxy(
		socketPath, "key", "proj-1", "feat-55", backend.URL,
		func() string { return "tok" },
		func() error { return nil },
		logger,
	)
	require.NoError(t, proxy.Start(context.Background()))
	defer proxy.Stop()

	client := proxyClient(socketPath, "key")
	body, _ := json.Marshal(map[string]string{"title": "new task", "summary": "test"})
	resp := doProxyReq(t, client, http.MethodPost, "/api/projects/_/tasks", "key", io.NopCloser(io.Reader(bytes(body))))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "feat-55", capturedBody["feature_id"])
	assert.Equal(t, "new task", capturedBody["title"])
}

func TestProxy_DoesNotInjectFeatureID_OnSubresource(t *testing.T) {
	var capturedBody map[string]any

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{"message": "ok"}})
	}))
	defer backend.Close()

	socketPath := filepath.Join(t.TempDir(), "proxy.sock")
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	proxy := app.NewSidecarProxy(
		socketPath, "key", "proj-1", "feat-55", backend.URL,
		func() string { return "tok" },
		func() error { return nil },
		logger,
	)
	require.NoError(t, proxy.Start(context.Background()))
	defer proxy.Stop()

	client := proxyClient(socketPath, "key")
	body, _ := json.Marshal(map[string]string{"blocked_reason": "waiting"})
	resp := doProxyReq(t, client, http.MethodPost, "/api/projects/_/tasks/task-1/block", "key", io.NopCloser(io.Reader(bytes(body))))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_, hasFeatureID := capturedBody["feature_id"]
	assert.False(t, hasFeatureID, "feature_id should not be injected for sub-resource endpoints")
}

func TestProxy_RejectsWrongAPIKey(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("backend should not be reached")
	}))
	defer backend.Close()

	socketPath := filepath.Join(t.TempDir(), "proxy.sock")
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	proxy := app.NewSidecarProxy(
		socketPath, "correct-key", "proj-1", "feat-1", backend.URL,
		func() string { return "tok" },
		func() error { return nil },
		logger,
	)
	require.NoError(t, proxy.Start(context.Background()))
	defer proxy.Stop()

	client := proxyClient(socketPath, "wrong-key")
	resp := doProxyReq(t, client, http.MethodGet, "/api/projects/_/tasks", "wrong-key", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProxy_RefreshesTokenOn401(t *testing.T) {
	refreshCalled := false

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"status": "fail", "error": map[string]string{"code": "UNAUTHORIZED", "message": "expired"}})
	}))
	defer backend.Close()

	socketPath := filepath.Join(t.TempDir(), "proxy.sock")
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	proxy := app.NewSidecarProxy(
		socketPath, "key", "proj-1", "", backend.URL,
		func() string { return "tok" },
		func() error { refreshCalled = true; return nil },
		logger,
	)
	require.NoError(t, proxy.Start(context.Background()))
	defer proxy.Stop()

	client := proxyClient(socketPath, "key")
	resp := doProxyReq(t, client, http.MethodGet, "/api/projects/_/tasks", "key", nil)
	defer resp.Body.Close()

	assert.True(t, refreshCalled, "refresh should be called on 401")
}

func TestProxy_StopRemovesSocket(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer backend.Close()

	socketPath := filepath.Join(t.TempDir(), "proxy.sock")
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	proxy := app.NewSidecarProxy(
		socketPath, "key", "p", "f", backend.URL,
		func() string { return "t" },
		func() error { return nil },
		logger,
	)
	require.NoError(t, proxy.Start(context.Background()))

	// Socket file should exist
	_, err := net.Dial("unix", socketPath)
	require.NoError(t, err)

	proxy.Stop()

	// Socket file should be gone
	_, err = net.Dial("unix", socketPath)
	assert.Error(t, err)
}

// bytes is a helper to convert []byte to an io.Reader via bytes.NewReader.
func bytes(b []byte) *bytesReader { return &bytesReader{b: b, pos: 0} }

type bytesReader struct {
	b   []byte
	pos int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}
