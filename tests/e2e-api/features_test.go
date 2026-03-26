package e2eapi

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// createProjectForFeatures creates a fresh project and returns its ID.
func createProjectForFeatures(t *testing.T, token string) string {
	t.Helper()
	type proj struct {
		ID string `json:"id"`
	}
	p := createAndDecode[proj](t, "/api/projects", token, map[string]any{
		"name":        "Feature Test Project " + t.Name(),
		"description": "test",
	})
	return p.ID
}

// ---------- 1. CRUD ---------------------------------------------------------

func TestFeatures_CRUD(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	type feature struct {
		ID             string `json:"id"`
		ProjectID      string `json:"project_id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		UserChangelog  string `json:"user_changelog"`
		TechChangelog  string `json:"tech_changelog"`
		Status         string `json:"status"`
		CreatedByRole  string `json:"created_by_role"`
		CreatedByAgent string `json:"created_by_agent"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	}

	// Create.
	created := createAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{
			"name":             "Login Page",
			"description":      "OAuth2 login page",
			"created_by_role":  "architect",
			"created_by_agent": "claude",
		})
	require.NotEmpty(t, created.ID)
	require.Equal(t, projectID, created.ProjectID)
	require.Equal(t, "Login Page", created.Name)
	require.Equal(t, "OAuth2 login page", created.Description)
	require.Equal(t, "draft", created.Status)
	require.Equal(t, "architect", created.CreatedByRole)
	require.Equal(t, "claude", created.CreatedByAgent)

	// Get.
	got := getAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID), token)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "Login Page", got.Name)

	// List — must contain the created feature.
	type featureWithSummary struct {
		feature
		TaskSummary struct {
			BacklogCount    int `json:"backlog_count"`
			TodoCount       int `json:"todo_count"`
			InProgressCount int `json:"in_progress_count"`
			DoneCount       int `json:"done_count"`
			BlockedCount    int `json:"blocked_count"`
		} `json:"task_summary"`
	}
	list := getAndDecode[[]featureWithSummary](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token)
	var found bool
	for _, f := range list {
		if f.ID == created.ID {
			found = true
			require.Equal(t, "Login Page", f.Name)
			break
		}
	}
	require.True(t, found, "created feature should appear in list")

	// Update name.
	patchAndDecode[struct {
		Message string `json:"message"`
	}](t, fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID), token,
		map[string]any{"name": "Updated Login Page"})

	updated := getAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID), token)
	require.Equal(t, "Updated Login Page", updated.Name)

	// Delete.
	deleteResource(t, fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID), token)

	// Verify gone.
	resp := doAuth(t, "GET",
		fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID), token, nil)
	require.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest,
		"expected 404 or 400 after delete, got %d", resp.StatusCode)
	resp.Body.Close()
}

// ---------- 2. Status Workflow ----------------------------------------------

func TestFeatures_StatusWorkflow(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	type feature struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	created := createAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{"name": "Status WF Feature", "description": "wf test"})
	require.Equal(t, "draft", created.Status)

	statusPath := fmt.Sprintf("/api/projects/%s/features/%s/status", projectID, created.ID)
	featurePath := fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID)

	// draft → ready
	patchAndDecode[struct{ Message string `json:"message"` }](t, statusPath, token,
		map[string]any{"status": "ready"})
	f := getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "ready", f.Status)

	// ready → in_progress
	patchAndDecode[struct{ Message string `json:"message"` }](t, statusPath, token,
		map[string]any{"status": "in_progress"})
	f = getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "in_progress", f.Status)

	// in_progress → done
	patchAndDecode[struct{ Message string `json:"message"` }](t, statusPath, token,
		map[string]any{"status": "done"})
	f = getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "done", f.Status)
}

// ---------- 3. Changelogs ---------------------------------------------------

func TestFeatures_Changelogs(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	type feature struct {
		ID            string `json:"id"`
		UserChangelog string `json:"user_changelog"`
		TechChangelog string `json:"tech_changelog"`
	}

	created := createAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{"name": "Changelog Feature", "description": "cl test"})
	require.Empty(t, created.UserChangelog)
	require.Empty(t, created.TechChangelog)

	changelogPath := fmt.Sprintf("/api/projects/%s/features/%s/changelogs", projectID, created.ID)
	featurePath := fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID)

	// Update both changelogs.
	patchAndDecode[struct{ Message string `json:"message"` }](t, changelogPath, token,
		map[string]any{
			"user_changelog": "Added dark mode support",
			"tech_changelog": "Refactored CSS variables",
		})

	got := getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "Added dark mode support", got.UserChangelog)
	require.Equal(t, "Refactored CSS variables", got.TechChangelog)

	// Update only user_changelog.
	patchAndDecode[struct{ Message string `json:"message"` }](t, changelogPath, token,
		map[string]any{
			"user_changelog": "Added dark mode and high contrast",
		})

	got = getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "Added dark mode and high contrast", got.UserChangelog)
}

// ---------- 4. WithTasks (FeatureWithSummary) --------------------------------

func TestFeatures_WithTasks(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	// Create feature.
	type featureBasic struct {
		ID string `json:"id"`
	}
	feat := createAndDecode[featureBasic](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{"name": "Tasks Feature", "description": "with tasks"})

	// Create a task (without feature_id — CreateTask validates it against projects).
	type taskBasic struct {
		ID string `json:"id"`
	}
	task := createAndDecode[taskBasic](t,
		fmt.Sprintf("/api/projects/%s/tasks", projectID), token,
		map[string]any{
			"title":   "Implement login",
			"summary": "Implement the login page UI",
		})

	// Link the task to the feature via UpdateTask (which sets feature_id without sub-project validation).
	patchAndDecode[struct {
		Message string `json:"message"`
	}](t, fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, task.ID), token,
		map[string]any{"feature_id": feat.ID})

	// List features — check task_summary includes the task.
	type featureWithSummary struct {
		ID          string `json:"id"`
		TaskSummary struct {
			BacklogCount    int `json:"backlog_count"`
			TodoCount       int `json:"todo_count"`
			InProgressCount int `json:"in_progress_count"`
			DoneCount       int `json:"done_count"`
			BlockedCount    int `json:"blocked_count"`
		} `json:"task_summary"`
	}
	list := getAndDecode[[]featureWithSummary](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token)

	var found *featureWithSummary
	for i := range list {
		if list[i].ID == feat.ID {
			found = &list[i]
			break
		}
	}
	require.NotNil(t, found, "feature should appear in list")

	totalTasks := found.TaskSummary.BacklogCount +
		found.TaskSummary.TodoCount +
		found.TaskSummary.InProgressCount +
		found.TaskSummary.DoneCount +
		found.TaskSummary.BlockedCount
	require.GreaterOrEqual(t, totalTasks, 1, "task_summary should reflect at least 1 task")
}

// ---------- 5. Task Summaries -----------------------------------------------

func TestFeatures_TaskSummaries(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	// Create feature.
	type featureBasic struct {
		ID string `json:"id"`
	}
	feat := createAndDecode[featureBasic](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{"name": "Summary Feature", "description": "summaries test"})

	// Create a task (without feature_id — CreateTask validates it against projects).
	type taskBasic struct {
		ID string `json:"id"`
	}
	task := createAndDecode[taskBasic](t,
		fmt.Sprintf("/api/projects/%s/tasks", projectID), token,
		map[string]any{
			"title":   "Build dashboard",
			"summary": "Build the main dashboard page",
		})

	// Link the task to the feature via UpdateTask (which sets feature_id without sub-project validation).
	patchResp := doAuth(t, "PATCH", fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, task.ID), token,
		map[string]any{"feature_id": feat.ID})
	requireStatus(t, patchResp, http.StatusOK)
	patchResp.Body.Close()

	// Move to in_progress (required before completing).
	moveResp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/move", projectID, task.ID), token,
		map[string]any{"target_column": "in_progress"})
	requireStatus(t, moveResp, http.StatusOK)
	moveResp.Body.Close()

	// Complete the task so it appears in task-summaries.
	completionSummary := "Implemented the main dashboard page with charts and widgets. " +
		"Added responsive layout, dark mode support, and real-time data updates. " +
		"All unit tests pass."
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/complete", projectID, task.ID), token,
		map[string]any{
			"completion_summary":  completionSummary,
			"files_modified":     []string{"dashboard.go", "dashboard_test.go"},
			"completed_by_agent": "claude",
		})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Get task-summaries for the feature.
	type taskSummary struct {
		TaskID            string    `json:"task_id"`
		Title             string    `json:"title"`
		CompletionSummary string    `json:"completion_summary"`
		CompletedByAgent  string    `json:"completed_by_agent"`
		CompletedAt       time.Time `json:"completed_at"`
		FilesModified     []string  `json:"files_modified"`
	}
	summaries := getAndDecode[[]taskSummary](t,
		fmt.Sprintf("/api/projects/%s/features/%s/task-summaries", projectID, feat.ID), token)
	require.NotEmpty(t, summaries, "should have at least one task summary")

	var found bool
	for _, s := range summaries {
		if s.TaskID == task.ID {
			found = true
			require.Equal(t, "Build dashboard", s.Title)
			require.Equal(t, completionSummary, s.CompletionSummary)
			require.Equal(t, "claude", s.CompletedByAgent)
			require.False(t, s.CompletedAt.IsZero(), "completed_at should be set")
			require.Contains(t, s.FilesModified, "dashboard.go")
			break
		}
	}
	require.True(t, found, "completed task should appear in task-summaries")
}

// ---------- 6. Stats --------------------------------------------------------

func TestFeatures_Stats(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	// Create a feature so stats are non-empty.
	type featureBasic struct {
		ID string `json:"id"`
	}
	createAndDecode[featureBasic](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{"name": "Stats Feature", "description": "stats test"})

	type featureStats struct {
		TotalCount      int `json:"total_count"`
		NotReadyCount   int `json:"not_ready_count"`
		ReadyCount      int `json:"ready_count"`
		InProgressCount int `json:"in_progress_count"`
		DoneCount       int `json:"done_count"`
		BlockedCount    int `json:"blocked_count"`
	}

	stats := getAndDecode[featureStats](t,
		fmt.Sprintf("/api/projects/%s/stats/features", projectID), token)
	require.GreaterOrEqual(t, stats.TotalCount, 1, "should have at least 1 feature")
	// A newly created feature defaults to "draft" which is not_ready.
	require.GreaterOrEqual(t, stats.NotReadyCount, 1, "draft feature should count as not_ready")
}

// ---------- 7. DB Verification ----------------------------------------------

func TestFeatures_DB_Verification(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	type feature struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	created := createAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{
			"name":        "DB Verify Feature",
			"description": "db verification test",
		})
	require.NotEmpty(t, created.ID)

	pool := testPool(t)
	require.True(t, rowExists(t, pool, "features", "id = $1", created.ID),
		"feature should exist in features table")

	// Verify the stored name matches.
	name := queryString(t, pool,
		"SELECT name FROM features WHERE id = $1", created.ID)
	require.Equal(t, "DB Verify Feature", name)
}
