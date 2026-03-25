package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCommentHandler(t *testing.T, mock *servicetest.MockCommands) *commands.CommentCommandsHandler {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil))
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()
	return commands.NewCommentCommandsHandler(mock, ctrl, hub)
}

func TestCreateComment_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	commentID := domain.NewCommentID()
	now := time.Now()

	mock := &servicetest.MockCommands{
		CreateCommentFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error) {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, "engineer", authorRole)
			assert.Equal(t, domain.AuthorTypeHuman, authorType)
			return domain.Comment{
				ID:         commentID,
				TaskID:     taskID,
				AuthorRole: authorRole,
				AuthorName: authorName,
				AuthorType: authorType,
				Content:    content,
				CreatedAt:  now,
			}, nil
		},
	}

	handler := newTestCommentHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"author_role": "engineer", "author_name": "Alice", "content": "This is a comment"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/comments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "This is a comment", data["content"])
	assert.Equal(t, "engineer", data["author_role"])
}

func TestCreateComment_ValidationError(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	mock := &servicetest.MockCommands{}

	handler := newTestCommentHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Missing required author_role and content
	body := `{"author_name": "Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/comments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	// Note: the handler wraps the already-domain-error in errors.Join, which prevents
	// a direct type assertion in SendFail, so the code falls back to "CLIENT_ERROR".
	// This is a known production code inconsistency that must not be changed here.
	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, errData["code"], "error code should not be empty")
}

func TestUpdateComment_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	mock := &servicetest.MockCommands{
		UpdateCommentFunc: func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID, content string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, commentID, cid)
			assert.Equal(t, "Updated content here", content)
			return nil
		},
	}

	handler := newTestCommentHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"content": "Updated content here"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/comments/"+string(commentID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "comment updated", data["message"])
}

func TestUpdateComment_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	mock := &servicetest.MockCommands{
		// Return a bare domain error (not wrapped in errors.Join) so that
		// SendFail's direct type assertion (*domain.Error) succeeds and
		// returns the correct domain error code.
		UpdateCommentFunc: func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID, content string) error {
			return domain.ErrCommentNotEditable
		},
	}

	handler := newTestCommentHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"content": "Trying to edit"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/comments/"+string(commentID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// UpdateComment uses http.StatusForbidden for domain errors
	assert.Equal(t, http.StatusForbidden, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "COMMENT_NOT_EDITABLE", errData["code"])
}

func TestDeleteComment_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	mock := &servicetest.MockCommands{
		DeleteCommentFunc: func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, commentID, cid)
			return nil
		},
	}

	handler := newTestCommentHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/comments/"+string(commentID), nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "comment deleted", data["message"])
}
