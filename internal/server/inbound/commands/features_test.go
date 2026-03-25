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
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateFeature tests the CreateFeature endpoint
func TestCreateFeature(t *testing.T) {
	t.Run("creates a new feature", func(t *testing.T) {
		projectID := newValidProjectID()
		featureID := domain.NewFeatureID()

		mockCommands := &servicetest.MockCommands{
			CreateFeatureFunc: func(ctx context.Context, pid domain.ProjectID, name, description, createdByRole, createdByAgent string) (domain.Feature, error) {
				if pid == projectID && name == "New Feature" {
					return domain.Feature{
						ID:             featureID,
						ProjectID:      projectID,
						Name:           name,
						Description:    description,
						Status:         domain.FeatureStatusDraft,
						CreatedByRole:  createdByRole,
						CreatedByAgent: createdByAgent,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil
				}
				return domain.Feature{}, domain.ErrProjectNotFound
			},
		}

		ctrl := newTestController()
		hub := websocket.NewHub(logrus.New())
		handler := commands.NewFeatureCommandsHandler(mockCommands, ctrl, hub)

		req := &pkgserver.CreateFeatureRequest{
			Name:           "New Feature",
			Description:    "A test feature",
			CreatedByRole:  "developer",
			CreatedByAgent: "agent1",
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+string(projectID)+"/features", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq = mux.SetURLVars(httpReq, map[string]string{"id": string(projectID)})
		w := httptest.NewRecorder()

		handler.CreateFeature(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, string(featureID), data["id"])
		assert.Equal(t, "New Feature", data["name"])
		assert.Equal(t, "draft", data["status"])
	})
}

// TestUpdateFeature tests the UpdateFeature endpoint
func TestUpdateFeature(t *testing.T) {
	t.Run("updates a feature", func(t *testing.T) {
		featureID := domain.NewFeatureID()

		mockCommands := &servicetest.MockCommands{
			UpdateFeatureFunc: func(ctx context.Context, fid domain.FeatureID, name, description string) error {
				if fid == featureID && name == "Updated Feature" {
					return nil
				}
				return domain.ErrFeatureNotFound
			},
		}

		ctrl := newTestController()
		hub := websocket.NewHub(logrus.New())
		handler := commands.NewFeatureCommandsHandler(mockCommands, ctrl, hub)

		newName := "Updated Feature"
		req := &pkgserver.UpdateFeatureRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPatch, "/api/projects/test-project/features/"+string(featureID), bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq = mux.SetURLVars(httpReq, map[string]string{"id": "test-project", "featureId": string(featureID)})
		w := httptest.NewRecorder()

		handler.UpdateFeature(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "feature updated", data["message"])
	})
}

// TestUpdateFeatureStatus tests the UpdateFeatureStatus endpoint
func TestUpdateFeatureStatus(t *testing.T) {
	t.Run("updates feature status", func(t *testing.T) {
		featureID := domain.NewFeatureID()

		mockCommands := &servicetest.MockCommands{
			UpdateFeatureStatusFunc: func(ctx context.Context, fid domain.FeatureID, status domain.FeatureStatus) error {
				if fid == featureID && status == domain.FeatureStatusReady {
					return nil
				}
				return domain.ErrFeatureNotFound
			},
		}

		ctrl := newTestController()
		hub := websocket.NewHub(logrus.New())
		handler := commands.NewFeatureCommandsHandler(mockCommands, ctrl, hub)

		req := &pkgserver.UpdateFeatureStatusRequest{
			Status: "ready",
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPatch, "/api/projects/test-project/features/"+string(featureID)+"/status", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq = mux.SetURLVars(httpReq, map[string]string{"id": "test-project", "featureId": string(featureID)})
		w := httptest.NewRecorder()

		handler.UpdateFeatureStatus(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "feature status updated", data["message"])
	})
}

// TestDeleteFeature tests the DeleteFeature endpoint
func TestDeleteFeature(t *testing.T) {
	t.Run("deletes a feature", func(t *testing.T) {
		featureID := domain.NewFeatureID()

		mockCommands := &servicetest.MockCommands{
			DeleteFeatureFunc: func(ctx context.Context, fid domain.FeatureID) error {
				if fid == featureID {
					return nil
				}
				return domain.ErrFeatureNotFound
			},
		}

		ctrl := newTestController()
		hub := websocket.NewHub(logrus.New())
		handler := commands.NewFeatureCommandsHandler(mockCommands, ctrl, hub)

		httpReq := httptest.NewRequest(http.MethodDelete, "/api/projects/test-project/features/"+string(featureID), nil)
		httpReq = mux.SetURLVars(httpReq, map[string]string{"id": "test-project", "featureId": string(featureID)})
		w := httptest.NewRecorder()

		handler.DeleteFeature(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "feature deleted", data["message"])
	})
}
