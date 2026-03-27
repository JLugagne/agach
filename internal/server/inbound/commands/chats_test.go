package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/chats/chatstest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/commands"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestChatHandler builds a ChatsHandler backed by a MockChatSession repository.
func newTestChatHandler(t *testing.T, mock *chatstest.MockChatSession) *commands.ChatsHandler {
	t.Helper()
	chatService := app.NewChatService(mock)
	ctrl := newTestController()
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil))
	hub := websocket.NewHub(logger)
	go hub.Run()
	dataDir := t.TempDir()
	return commands.NewChatsHandler(chatService, nil, ctrl, hub, dataDir)
}

// TestChatStartSession_Success verifies that a valid POST creates a session and
// returns 200 with the session data.
func TestChatStartSession_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	featureID := domain.FeatureID(domain.NewProjectID())
	sessionID := domain.NewChatSessionID()

	mock := &chatstest.MockChatSession{
		CreateFunc: func(ctx context.Context, session domain.ChatSession) error {
			// Capture the generated ID so we can override for determinism.
			return nil
		},
		FindByIDFunc: func(ctx context.Context, id domain.ChatSessionID) (*domain.ChatSession, error) {
			return &domain.ChatSession{
				ID:        id,
				FeatureID: featureID,
				ProjectID: projectID,
				State:     domain.ChatStateActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}
	_ = sessionID

	handler := newTestChatHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/projects/"+string(projectID)+"/features/"+string(featureID)+"/chats",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "data should be an object")
	assert.Equal(t, string(featureID), data["feature_id"])
	assert.Equal(t, string(projectID), data["project_id"])
	assert.Equal(t, "active", data["state"])
}

// TestChatStartSession_RepositoryError verifies that a repository error returns
// a 500 error response.
func TestChatStartSession_RepositoryError(t *testing.T) {
	projectID := domain.NewProjectID()
	featureID := domain.FeatureID(domain.NewProjectID())

	mock := &chatstest.MockChatSession{
		CreateFunc: func(ctx context.Context, session domain.ChatSession) error {
			return &domain.Error{Code: "FEATURE_NOT_FOUND", Message: "feature not found"}
		},
	}

	handler := newTestChatHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost,
		"/api/projects/"+string(projectID)+"/features/"+string(featureID)+"/chats",
		strings.NewReader(`{}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])
}

// TestChatUploadJSONL_Success verifies a valid multipart upload stores the file
// and records the path.
func TestChatUploadJSONL_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	featureID := domain.FeatureID(domain.NewProjectID())
	sessionID := domain.NewChatSessionID()

	updatedPath := ""

	mock := &chatstest.MockChatSession{
		UpdateJSONLPathFunc: func(ctx context.Context, id domain.ChatSessionID, path string) error {
			updatedPath = path
			return nil
		},
	}

	handler := newTestChatHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Build multipart body.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("jsonl", "conversation.jsonl")
	require.NoError(t, err)
	_, err = fw.Write([]byte(`{"type":"user","content":"hello"}`))
	require.NoError(t, err)
	w.Close()

	url := "/api/projects/" + string(projectID) +
		"/features/" + string(featureID) +
		"/chats/" + sessionID.String() + "/upload"

	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "JSONL file uploaded successfully", data["message"])
	assert.NotEmpty(t, updatedPath)
	assert.True(t, strings.HasSuffix(updatedPath, ".jsonl"), "path should end with .jsonl: %s", updatedPath)
}

// TestChatUploadJSONL_MissingFile verifies that a missing "jsonl" field returns
// a fail response.
func TestChatUploadJSONL_MissingFile(t *testing.T) {
	projectID := domain.NewProjectID()
	featureID := domain.FeatureID(domain.NewProjectID())
	sessionID := domain.NewChatSessionID()

	mock := &chatstest.MockChatSession{}

	handler := newTestChatHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.Close()

	url := "/api/projects/" + string(projectID) +
		"/features/" + string(featureID) +
		"/chats/" + sessionID.String() + "/upload"

	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])
}

// TestChatUploadJSONL_WrongExtension verifies that uploading a non-.jsonl file
// returns a fail response.
func TestChatUploadJSONL_WrongExtension(t *testing.T) {
	projectID := domain.NewProjectID()
	featureID := domain.FeatureID(domain.NewProjectID())
	sessionID := domain.NewChatSessionID()

	mock := &chatstest.MockChatSession{}

	handler := newTestChatHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("jsonl", "conversation.txt")
	require.NoError(t, err)
	fw.Write([]byte("not jsonl"))
	w.Close()

	url := "/api/projects/" + string(projectID) +
		"/features/" + string(featureID) +
		"/chats/" + sessionID.String() + "/upload"

	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])
}
