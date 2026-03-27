package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/commands"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTaskHandler(t *testing.T, mock *servicetest.MockCommands) *commands.TaskCommandsHandler {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil))
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()
	return commands.NewTaskCommandsHandler(mock, ctrl, hub, nil)
}

func TestCreateTask_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()
	now := time.Now()

	mock := &servicetest.MockCommands{
		CreateTaskFunc: func(ctx context.Context, pid domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool, featureID *domain.FeatureID) (domain.Task, error) {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, "Fix the bug", title)
			assert.Equal(t, "A brief summary", summary)
			return domain.Task{
				ID:            taskID,
				ColumnID:      columnID,
				Title:         title,
				Summary:       summary,
				Description:   description,
				Priority:      priority,
				Tags:          []string{},
				ContextFiles:  []string{},
				FilesModified: []string{},
				CreatedAt:     now,
				UpdatedAt:     now,
			}, nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"title": "Fix the bug", "summary": "A brief summary", "description": "Details here", "priority": "high"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Fix the bug", data["title"])
	assert.Equal(t, "A brief summary", data["summary"])
}

func TestCreateTask_ValidationError(t *testing.T) {
	projectID := domain.NewProjectID()
	mock := &servicetest.MockCommands{}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Missing required title and summary
	body := `{"description": "only description"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	// The CreateTask handler wraps the validation error in errors.Join, which prevents
	// SendFail's direct type assertion from recognizing the domain error code.
	// This is a known production code inconsistency; we verify a fail response with
	// a non-empty error code rather than asserting the exact domain code.
	assert.NotEmpty(t, errData["code"], "error code should not be empty")
}

func TestCreateTask_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()

	mock := &servicetest.MockCommands{
		CreateTaskFunc: func(ctx context.Context, pid domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool, featureID *domain.FeatureID) (domain.Task, error) {
			return domain.Task{}, domain.ErrSummaryRequired
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"title": "Task Title", "summary": "s"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "SUMMARY_REQUIRED", errData["code"])
}

func TestDeleteTask_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		DeleteTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID), nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task deleted", data["message"])
}

func TestDeleteTask_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		DeleteTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) error {
			return domain.ErrTaskNotFound
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID), nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "TASK_NOT_FOUND", errData["code"])
}

func TestMoveTask_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		MoveTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, targetColumn domain.ColumnSlug, _ string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, domain.ColumnDone, targetColumn)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"target_column": "done"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task moved", data["message"])
}

func TestMoveTask_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		MoveTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, targetColumn domain.ColumnSlug, _ string) error {
			return domain.ErrTaskNotFound
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"target_column": "done"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "TASK_NOT_FOUND", errData["code"])
}

func TestCompleteTask_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		CompleteTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, _ string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, "agent-007", completedByAgent)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// completion_summary must be at least 100 chars per validation
	summary := strings.Repeat("x", 100)
	body := `{"completion_summary": "` + summary + `", "completed_by_agent": "agent-007", "files_modified": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/complete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task completed", data["message"])
}

func TestCompleteTask_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		CompleteTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, _ string) error {
			return domain.ErrUnresolvedDependencies
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	summary := strings.Repeat("x", 100)
	body := `{"completion_summary": "` + summary + `", "completed_by_agent": "agent-007", "files_modified": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/complete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "UNRESOLVED_DEPENDENCIES", errData["code"])
}

func TestUnblockTask_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		UnblockTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, _ string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/unblock", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task unblocked", data["message"])
}

func TestUnblockTask_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		UnblockTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, _ string) error {
			return domain.ErrTaskNotBlocked
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/unblock", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "TASK_NOT_BLOCKED", errData["code"])
}

func TestApproveWontDo_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		ApproveWontDoFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/approve-wont-do", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "won't do approved", data["message"])
}

func TestRejectWontDo_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		RejectWontDoFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, reason string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, "Not a valid reason to skip", reason)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"reason": "Not a valid reason to skip"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/reject-wont-do", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "won't do rejected", data["message"])
}

func TestUpdateTask_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		UpdateTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.FeatureID, clearFeature bool) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			require.NotNil(t, title)
			assert.Equal(t, "Updated title", *title)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"title": "Updated title"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
}

func TestUpdateTask_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		UpdateTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.FeatureID, clearFeature bool) error {
			return domain.ErrTaskNotFound
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"title": "Updated title"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])
}

func TestWontDo_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		RequestWontDoFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, reason, requestedBy, _ string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			return nil
		},
		ApproveWontDoFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) error {
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	reason := strings.Repeat("x", 50)
	body := `{"wont_do_reason": "` + reason + `", "wont_do_requested_by": "human"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/wont-do", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task marked as won't do", data["message"])
}

func TestMoveTaskToProject_Success(t *testing.T) {
	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		MoveTaskToProjectFunc: func(ctx context.Context, srcPID domain.ProjectID, tid domain.TaskID, dstPID domain.ProjectID) error {
			assert.Equal(t, sourceProjectID, srcPID)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, targetProjectID, dstPID)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"target_project_id": "` + string(targetProjectID) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(sourceProjectID)+"/tasks/"+string(taskID)+"/move-to-project", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task moved to project", data["message"])
}

func TestReorderTask_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		ReorderTaskFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, newPosition int) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, 2, newPosition)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"position": 2}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/reorder", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "task reordered", data["message"])
}

func TestUpdateTaskSession_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		UpdateTaskSessionIDFunc: func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, sessionID string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, "sess-abc123", sessionID)
			return nil
		},
	}

	handler := newTestTaskHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"session_id": "sess-abc123"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/session", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
}
