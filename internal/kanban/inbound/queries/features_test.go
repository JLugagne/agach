package queries_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/queries"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleListFeatures tests the features endpoint
func TestHandleListFeatures(t *testing.T) {
	t.Run("returns all features when active_only is false or omitted", func(t *testing.T) {
		parentID := newValidProjectID()
		feature1ID := newValidProjectID()
		feature2ID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			ListSubProjectsWithSummaryFunc: func(ctx context.Context, parent domain.ProjectID) ([]domain.ProjectWithSummary, error) {
				if parent == parentID {
					return []domain.ProjectWithSummary{
						{
							Project: domain.Project{
								ID:   feature1ID,
								Name: "Feature A",
							},
						},
						{
							Project: domain.Project{
								ID:   feature2ID,
								Name: "Feature B",
							},
						},
					}, nil
				}
				return nil, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewProjectQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(parentID)+"/features", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(parentID)})
		w := httptest.NewRecorder()

		handler.HandleListFeatures(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		// Should have 2 features
		data, ok := result["data"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, data, 2)
	})

	t.Run("returns only active features when active_only=true", func(t *testing.T) {
		parentID := newValidProjectID()
		activeFeatureID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			ListFeaturesActiveOnlyFunc: func(ctx context.Context, parent domain.ProjectID) ([]domain.ProjectWithSummary, error) {
				if parent == parentID {
					return []domain.ProjectWithSummary{
						{
							Project: domain.Project{
								ID:   activeFeatureID,
								Name: "Active Feature",
							},
						},
					}, nil
				}
				return nil, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewProjectQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(parentID)+"/features?active_only=true", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(parentID)})
		w := httptest.NewRecorder()

		handler.HandleListFeatures(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		// Should have 1 active feature
		data, ok := result["data"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, data, 1)
	})

	t.Run("returns empty list when no features exist", func(t *testing.T) {
		parentID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			ListSubProjectsWithSummaryFunc: func(ctx context.Context, parent domain.ProjectID) ([]domain.ProjectWithSummary, error) {
				return []domain.ProjectWithSummary{}, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewProjectQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(parentID)+"/features", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(parentID)})
		w := httptest.NewRecorder()

		handler.HandleListFeatures(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		// Should have empty list
		data, ok := result["data"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, data, 0)
	})

	t.Run("calls ListFeaturesActiveOnly when active_only=true", func(t *testing.T) {
		parentID := newValidProjectID()
		listFeaturesActiveOnlyCalled := false

		mockQueries := &servicetest.MockQueries{
			ListFeaturesActiveOnlyFunc: func(ctx context.Context, parent domain.ProjectID) ([]domain.ProjectWithSummary, error) {
				listFeaturesActiveOnlyCalled = true
				return []domain.ProjectWithSummary{}, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewProjectQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(parentID)+"/features?active_only=true", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(parentID)})
		w := httptest.NewRecorder()

		handler.HandleListFeatures(w, req)

		assert.True(t, listFeaturesActiveOnlyCalled, "ListFeaturesActiveOnly should have been called")
	})

	t.Run("calls ListSubProjectsWithSummary when active_only=false explicitly", func(t *testing.T) {
		parentID := newValidProjectID()
		listSubProjectsWithSummaryCalled := false

		mockQueries := &servicetest.MockQueries{
			ListSubProjectsWithSummaryFunc: func(ctx context.Context, parent domain.ProjectID) ([]domain.ProjectWithSummary, error) {
				listSubProjectsWithSummaryCalled = true
				return []domain.ProjectWithSummary{}, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewProjectQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(parentID)+"/features?active_only=false", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(parentID)})
		w := httptest.NewRecorder()

		handler.HandleListFeatures(w, req)

		assert.True(t, listSubProjectsWithSummaryCalled, "ListSubProjectsWithSummary should have been called")
	})
}
