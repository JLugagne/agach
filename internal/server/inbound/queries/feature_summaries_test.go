package queries_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/queries"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListFeatureTaskSummaries_Empty verifies that a feature with no completed tasks returns an empty slice.
func TestListFeatureTaskSummaries_Empty(t *testing.T) {
	featureID := domain.NewFeatureID()

	mockQueries := &servicetest.MockQueries{
		ListFeatureTaskSummariesFunc: func(ctx context.Context, fid domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
			if fid == featureID {
				return []domain.FeatureTaskSummary{}, nil
			}
			return nil, domain.ErrFeatureNotFound
		},
	}

	ctrl := newTestController()
	handler := queries.NewFeatureSummariesHandler(mockQueries, ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/features/"+string(featureID)+"/task-summaries", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-project", "featureId": string(featureID)})
	w := httptest.NewRecorder()

	handler.ListFeatureTaskSummaries(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	data, ok := result["data"].([]interface{})
	assert.True(t, ok, "expected data to be an array")
	assert.Len(t, data, 0)
}

// TestListFeatureTaskSummaries_SortedByCompletedAt verifies that task summaries are returned in ASC order by completed_at.
func TestListFeatureTaskSummaries_SortedByCompletedAt(t *testing.T) {
	featureID := domain.NewFeatureID()

	now := time.Now().UTC().Truncate(time.Second)
	earlier := now.Add(-2 * time.Hour)
	later := now.Add(-1 * time.Hour)

	task1ID := domain.NewTaskID()
	task2ID := domain.NewTaskID()
	task3ID := domain.NewTaskID()

	mockQueries := &servicetest.MockQueries{
		ListFeatureTaskSummariesFunc: func(ctx context.Context, fid domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
			if fid == featureID {
				// Return in ASC order as the service/repository should guarantee
				return []domain.FeatureTaskSummary{
					{
						ID:                task1ID,
						Title:             "First task",
						CompletionSummary: "First done",
						CompletedByAgent:  "agent-a",
						CompletedAt:       earlier,
						FilesModified:     []string{"file1.go"},
					},
					{
						ID:                task2ID,
						Title:             "Second task",
						CompletionSummary: "Second done",
						CompletedByAgent:  "agent-b",
						CompletedAt:       later,
						FilesModified:     []string{"file2.go"},
					},
					{
						ID:                task3ID,
						Title:             "Third task",
						CompletionSummary: "Third done",
						CompletedByAgent:  "agent-c",
						CompletedAt:       now,
						FilesModified:     nil,
					},
				}, nil
			}
			return nil, domain.ErrFeatureNotFound
		},
	}

	ctrl := newTestController()
	handler := queries.NewFeatureSummariesHandler(mockQueries, ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/features/"+string(featureID)+"/task-summaries", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-project", "featureId": string(featureID)})
	w := httptest.NewRecorder()

	handler.ListFeatureTaskSummaries(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	data, ok := result["data"].([]interface{})
	require.True(t, ok, "expected data to be an array")
	require.Len(t, data, 3)

	// Verify ASC ordering by completed_at — first item is the earliest
	first := data[0].(map[string]interface{})
	second := data[1].(map[string]interface{})
	third := data[2].(map[string]interface{})

	assert.Equal(t, string(task1ID), first["task_id"])
	assert.Equal(t, string(task2ID), second["task_id"])
	assert.Equal(t, string(task3ID), third["task_id"])

	assert.Equal(t, "First task", first["title"])
	assert.Equal(t, "Second task", second["title"])
	assert.Equal(t, "Third task", third["title"])
}

// TestListFeatureTaskSummaries_OnlyCompleted verifies that incomplete tasks are excluded from the results.
func TestListFeatureTaskSummaries_OnlyCompleted(t *testing.T) {
	featureID := domain.NewFeatureID()

	completedTaskID := domain.NewTaskID()

	mockQueries := &servicetest.MockQueries{
		ListFeatureTaskSummariesFunc: func(ctx context.Context, fid domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
			if fid == featureID {
				// Only completed tasks are returned — the service filters out incomplete ones
				return []domain.FeatureTaskSummary{
					{
						ID:                completedTaskID,
						Title:             "Completed task",
						CompletionSummary: "Done successfully",
						CompletedByAgent:  "agent-a",
						CompletedAt:       time.Now().UTC(),
						FilesModified:     []string{"main.go"},
					},
				}, nil
			}
			return nil, domain.ErrFeatureNotFound
		},
	}

	ctrl := newTestController()
	handler := queries.NewFeatureSummariesHandler(mockQueries, ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/features/"+string(featureID)+"/task-summaries", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-project", "featureId": string(featureID)})
	w := httptest.NewRecorder()

	handler.ListFeatureTaskSummaries(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	data, ok := result["data"].([]interface{})
	require.True(t, ok, "expected data to be an array")
	require.Len(t, data, 1, "only completed tasks should be returned")

	item := data[0].(map[string]interface{})
	assert.Equal(t, string(completedTaskID), item["task_id"])
	assert.Equal(t, "Completed task", item["title"])
	assert.Equal(t, "Done successfully", item["completion_summary"])
	assert.Equal(t, "agent-a", item["completed_by_agent"])
}

// TestListFeatureTaskSummaries_FeatureNotFound verifies that a 404 is returned for an unknown feature ID.
func TestListFeatureTaskSummaries_FeatureNotFound(t *testing.T) {
	unknownFeatureID := domain.NewFeatureID()

	mockQueries := &servicetest.MockQueries{
		ListFeatureTaskSummariesFunc: func(ctx context.Context, fid domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
			return nil, domain.ErrFeatureNotFound
		},
	}

	ctrl := newTestController()
	handler := queries.NewFeatureSummariesHandler(mockQueries, ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/features/"+string(unknownFeatureID)+"/task-summaries", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-project", "featureId": string(unknownFeatureID)})
	w := httptest.NewRecorder()

	handler.ListFeatureTaskSummaries(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "fail", result["status"])
}
