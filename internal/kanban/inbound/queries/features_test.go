package queries_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/queries"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListFeatures tests the ListFeatures endpoint
func TestListFeatures(t *testing.T) {
	t.Run("returns all features when no filter is applied", func(t *testing.T) {
		projectID := newValidProjectID()
		feature1ID := newValidProjectID()
		feature2ID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			ListFeaturesFunc: func(ctx context.Context, pid domain.ProjectID, statuses []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
				if pid == projectID {
					return []domain.FeatureWithTaskSummary{
						{
							Feature: domain.Feature{
								ID:        feature1ID,
								ProjectID: projectID,
								Name:      "Feature A",
								Status:    domain.FeatureStatusDraft,
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
							},
						},
						{
							Feature: domain.Feature{
								ID:        feature2ID,
								ProjectID: projectID,
								Name:      "Feature B",
								Status:    domain.FeatureStatusReady,
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
							},
						},
					}, nil
				}
				return nil, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewFeatureQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(projectID)+"/features", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(projectID)})
		w := httptest.NewRecorder()

		handler.ListFeatures(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, data, 2)
	})

	t.Run("returns empty list when no features exist", func(t *testing.T) {
		projectID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			ListFeaturesFunc: func(ctx context.Context, pid domain.ProjectID, statuses []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
				return []domain.FeatureWithTaskSummary{}, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewFeatureQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(projectID)+"/features", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(projectID)})
		w := httptest.NewRecorder()

		handler.ListFeatures(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, data, 0)
	})

	t.Run("filters features by status", func(t *testing.T) {
		projectID := newValidProjectID()
		feature1ID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			ListFeaturesFunc: func(ctx context.Context, pid domain.ProjectID, statuses []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
				// Check that status filter was passed correctly
				if len(statuses) == 1 && statuses[0] == domain.FeatureStatusDraft {
					return []domain.FeatureWithTaskSummary{
						{
							Feature: domain.Feature{
								ID:        feature1ID,
								ProjectID: projectID,
								Name:      "Draft Feature",
								Status:    domain.FeatureStatusDraft,
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
							},
						},
					}, nil
				}
				return []domain.FeatureWithTaskSummary{}, nil
			},
		}

		ctrl := newTestController()
		handler := queries.NewFeatureQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(projectID)+"/features?status=draft", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(projectID)})
		w := httptest.NewRecorder()

		handler.ListFeatures(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, data, 1)
	})
}

// TestGetFeature tests the GetFeature endpoint
func TestGetFeature(t *testing.T) {
	t.Run("returns a feature by ID", func(t *testing.T) {
		featureID := newValidProjectID()
		projectID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			GetFeatureFunc: func(ctx context.Context, fid domain.FeatureID) (*domain.Feature, error) {
				if fid == featureID {
					return &domain.Feature{
						ID:        featureID,
						ProjectID: projectID,
						Name:      "Test Feature",
						Status:    domain.FeatureStatusReady,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil
				}
				return nil, domain.ErrFeatureNotFound
			},
		}

		ctrl := newTestController()
		handler := queries.NewFeatureQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(projectID)+"/features/"+string(featureID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(projectID), "featureId": string(featureID)})
		w := httptest.NewRecorder()

		handler.GetFeature(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, string(featureID), data["id"])
		assert.Equal(t, "Test Feature", data["name"])
	})
}

// TestGetFeatureStats tests the GetFeatureStats endpoint
func TestGetFeatureStats(t *testing.T) {
	t.Run("returns feature statistics", func(t *testing.T) {
		projectID := newValidProjectID()

		mockQueries := &servicetest.MockQueries{
			GetFeatureStatsFunc: func(ctx context.Context, pid domain.ProjectID) (*domain.FeatureStats, error) {
				if pid == projectID {
					return &domain.FeatureStats{
						TotalCount:      5,
						NotReadyCount:   2,
						ReadyCount:      2,
						InProgressCount: 1,
						DoneCount:       0,
						BlockedCount:    0,
					}, nil
				}
				return nil, domain.ErrProjectNotFound
			},
		}

		ctrl := newTestController()
		handler := queries.NewFeatureQueriesHandler(mockQueries, ctrl)
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+string(projectID)+"/stats/features", nil)
		req = mux.SetURLVars(req, map[string]string{"id": string(projectID)})
		w := httptest.NewRecorder()

		handler.GetFeatureStats(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, float64(5), data["total_count"])
		assert.Equal(t, float64(2), data["ready_count"])
	})
}
