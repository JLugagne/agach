package proxy_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/sidecar/domain"
	"github.com/JLugagne/agach-mcp/internal/sidecar/outbound/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAPIKey = "test-key-abc"

// startTestServer starts an HTTP server on a Unix socket and returns the
// socket path and a cleanup function. The handler records requests.
func startTestServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)

	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)
	t.Cleanup(func() {
		srv.Close()
		os.Remove(socketPath)
	})

	return socketPath
}

func successResponse(data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   data,
		})
	}
}

func TestClient_CreateTask(t *testing.T) {
	var captured struct {
		method string
		path   string
		apiKey string
		body   domain.CreateTaskRequest
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.apiKey = r.Header.Get("X-Api-Key")
		json.NewDecoder(r.Body).Decode(&captured.body)
		successResponse(map[string]string{"id": "new-task-id"})(w, r)
	})
	socketPath := startTestServer(t, mux)

	client := proxy.New(socketPath, testAPIKey)
	resp, err := client.CreateTask(context.Background(), domain.CreateTaskRequest{
		Title:   "Test task",
		Summary: "A test",
	})

	require.NoError(t, err)
	assert.Equal(t, "new-task-id", resp.ID)
	assert.Equal(t, http.MethodPost, captured.method)
	assert.Equal(t, "/api/projects/_/tasks", captured.path)
	assert.Equal(t, testAPIKey, captured.apiKey)
	assert.Equal(t, "Test task", captured.body.Title)
}

func TestClient_AddDependency(t *testing.T) {
	var captured struct {
		path string
		body map[string]string
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captured.path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&captured.body)
		successResponse(map[string]string{"message": "ok"})(w, r)
	})
	socketPath := startTestServer(t, mux)

	client := proxy.New(socketPath, testAPIKey)
	err := client.AddDependency(context.Background(), "task-1", "task-0")

	require.NoError(t, err)
	assert.Equal(t, "/api/projects/_/tasks/task-1/dependencies", captured.path)
	assert.Equal(t, "task-0", captured.body["depends_on_task_id"])
}

func TestClient_CompleteTask(t *testing.T) {
	var captured struct {
		path string
		body domain.CompleteTaskRequest
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captured.path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&captured.body)
		successResponse(map[string]string{"message": "ok"})(w, r)
	})
	socketPath := startTestServer(t, mux)

	client := proxy.New(socketPath, testAPIKey)
	err := client.CompleteTask(context.Background(), "task-99", domain.CompleteTaskRequest{
		CompletionSummary: "All done",
		FilesModified:     []string{"main.go"},
		CompletedByAgent:  "green",
	})

	require.NoError(t, err)
	assert.Equal(t, "/api/projects/_/tasks/task-99/complete", captured.path)
	assert.Equal(t, "All done", captured.body.CompletionSummary)
	assert.Equal(t, []string{"main.go"}, captured.body.FilesModified)
}

func TestClient_MoveTask(t *testing.T) {
	var captured struct {
		path string
		body map[string]string
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captured.path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&captured.body)
		successResponse(map[string]string{"message": "ok"})(w, r)
	})
	socketPath := startTestServer(t, mux)

	client := proxy.New(socketPath, testAPIKey)
	err := client.MoveTask(context.Background(), "task-5", "in_progress")

	require.NoError(t, err)
	assert.Equal(t, "/api/projects/_/tasks/task-5/move", captured.path)
	assert.Equal(t, "in_progress", captured.body["target_column"])
}

func TestClient_BlockTask(t *testing.T) {
	var captured struct {
		path string
		body domain.BlockTaskRequest
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captured.path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&captured.body)
		successResponse(map[string]string{"message": "ok"})(w, r)
	})
	socketPath := startTestServer(t, mux)

	client := proxy.New(socketPath, testAPIKey)
	err := client.BlockTask(context.Background(), "task-7", domain.BlockTaskRequest{
		BlockedReason:  "Needs API key",
		BlockedByAgent: "red",
	})

	require.NoError(t, err)
	assert.Equal(t, "/api/projects/_/tasks/task-7/block", captured.path)
	assert.Equal(t, "Needs API key", captured.body.BlockedReason)
}

func TestClient_RequestWontDo(t *testing.T) {
	var captured struct {
		path string
		body domain.WontDoRequest
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captured.path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&captured.body)
		successResponse(map[string]string{"message": "ok"})(w, r)
	})
	socketPath := startTestServer(t, mux)

	client := proxy.New(socketPath, testAPIKey)
	err := client.RequestWontDo(context.Background(), "task-9", domain.WontDoRequest{
		WontDoReason:      "Out of scope",
		WontDoRequestedBy: "reviewer",
	})

	require.NoError(t, err)
	assert.Equal(t, "/api/projects/_/tasks/task-9/wont-do", captured.path)
	assert.Equal(t, "Out of scope", captured.body.WontDoReason)
}

func TestClient_UpdateFeatureChangelogs(t *testing.T) {
	var captured struct {
		method string
		path   string
		body   domain.FeatureChangelogsRequest
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		json.NewDecoder(r.Body).Decode(&captured.body)
		successResponse(map[string]string{"message": "ok"})(w, r)
	})
	socketPath := startTestServer(t, mux)

	userCL := "New login"
	client := proxy.New(socketPath, testAPIKey)
	err := client.UpdateFeatureChangelogs(context.Background(), domain.FeatureChangelogsRequest{
		UserChangelog: &userCL,
	})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, captured.method)
	assert.Equal(t, "/api/projects/_/features/_/changelogs", captured.path)
	require.NotNil(t, captured.body.UserChangelog)
	assert.Equal(t, "New login", *captured.body.UserChangelog)
}

func TestClient_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "fail",
			"error": map[string]string{
				"code":    "TASK_NOT_FOUND",
				"message": "task does not exist",
			},
		})
	})
	socketPath := startTestServer(t, mux)

	client := proxy.New(socketPath, testAPIKey)
	err := client.MoveTask(context.Background(), "bad-id", "done")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "TASK_NOT_FOUND")
	assert.Contains(t, err.Error(), "task does not exist")
}
