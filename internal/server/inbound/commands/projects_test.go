package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func newTestProjectHandler(t *testing.T, mock *servicetest.MockCommands) (*commands.ProjectCommandsHandler, *websocket.Hub) {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil)) // suppress log output in tests
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()
	return commands.NewProjectCommandsHandler(mock, ctrl, hub), hub
}

func TestCreateProject_Success(t *testing.T) {
	projectID := domain.NewProjectID()
	now := time.Now()

	mock := &servicetest.MockCommands{
		CreateProjectFunc: func(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			return domain.Project{
				ID:             projectID,
				Name:           name,
				Description:    description,
				CreatedByRole:  createdByRole,
				CreatedByAgent: createdByAgent,
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	}

	handler, _ := newTestProjectHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"name": "My Project", "description": "A test project", "created_by_role": "architect", "created_by_agent": "agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "data should be an object")
	assert.Equal(t, "My Project", data["name"])
	assert.Equal(t, "A test project", data["description"])
}

func TestCreateProject_ValidationError(t *testing.T) {
	mock := &servicetest.MockCommands{}

	handler, _ := newTestProjectHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Empty name fails validation (required field)
	body := `{"name": "", "description": "A test project"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "INVALID_PROJECT_REQUEST", errData["code"])
}

func TestCreateProject_DomainError(t *testing.T) {
	mock := &servicetest.MockCommands{
		// Return the bare domain error so that SendFail's direct type assertion
		// (*domain.Error) succeeds and returns the correct error code.
		CreateProjectFunc: func(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			return domain.Project{}, domain.ErrProjectAlreadyExists
		},
	}

	handler, _ := newTestProjectHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"name": "Duplicate Project"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "PROJECT_ALREADY_EXISTS", errData["code"])
}

func TestCreateProject_ServerError(t *testing.T) {
	mock := &servicetest.MockCommands{
		CreateProjectFunc: func(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			return domain.Project{}, errors.New("unexpected database failure")
		},
	}

	handler, _ := newTestProjectHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"name": "Some Project"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "error", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", errData["code"])
}

func TestDeleteProject_Success(t *testing.T) {
	projectID := domain.NewProjectID()

	mock := &servicetest.MockCommands{
		DeleteProjectFunc: func(ctx context.Context, id domain.ProjectID) error {
			assert.Equal(t, projectID, id)
			return nil
		},
	}

	handler, _ := newTestProjectHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+string(projectID), nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])

	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "project deleted", data["message"])
}

func TestDeleteProject_DomainError(t *testing.T) {
	projectID := domain.NewProjectID()

	mock := &servicetest.MockCommands{
		DeleteProjectFunc: func(ctx context.Context, id domain.ProjectID) error {
			return domain.ErrProjectNotFound
		},
	}

	handler, _ := newTestProjectHandler(t, mock)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+string(projectID), nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])

	errData, ok := resp["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "PROJECT_NOT_FOUND", errData["code"])
}
