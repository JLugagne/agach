package e2eapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// node_id tracking on task events
// ---------------------------------------------------------------------------

func TestNodeID_MoveTask(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)
	pool := testPool(t)

	task := createTask(t, token, projectID, nil)

	// Move to in_progress with node_id
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/move", projectID, task.ID),
		token,
		map[string]any{"target_column": "in_progress", "node_id": "node-abc-123"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify via API
	got := getTask(t, token, projectID, task.ID)
	require.Equal(t, "node-abc-123", got.NodeID)

	// Verify in DB
	dbNodeID := queryNullableString(t, pool, "SELECT node_id FROM tasks WHERE id = $1", task.ID)
	require.NotNil(t, dbNodeID)
	require.Equal(t, "node-abc-123", *dbNodeID)
}

func TestNodeID_MoveTask_UpdatesNodeID(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	task := createTask(t, token, projectID, nil)

	// First move with node-1
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/move", projectID, task.ID),
		token,
		map[string]any{"target_column": "in_progress", "node_id": "node-1"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	got := getTask(t, token, projectID, task.ID)
	require.Equal(t, "node-1", got.NodeID)

	// Move back to todo with node-2
	resp = doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/move", projectID, task.ID),
		token,
		map[string]any{"target_column": "todo", "node_id": "node-2"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	got = getTask(t, token, projectID, task.ID)
	require.Equal(t, "node-2", got.NodeID)
}

func TestNodeID_MoveTask_PreservesWhenEmpty(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	task := createTask(t, token, projectID, nil)

	// Move with node_id
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/move", projectID, task.ID),
		token,
		map[string]any{"target_column": "in_progress", "node_id": "node-keep"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Move again without node_id — should preserve the previous value
	resp = doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/move", projectID, task.ID),
		token,
		map[string]any{"target_column": "todo"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	got := getTask(t, token, projectID, task.ID)
	require.Equal(t, "node-keep", got.NodeID, "node_id should be preserved when not provided")
}

func TestNodeID_CompleteTask(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)
	pool := testPool(t)

	task := createTask(t, token, projectID, nil)
	moveTask(t, token, projectID, task.ID, "in_progress")

	summary := strings.Repeat("Completed task with node tracking. ", 5)
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/complete", projectID, task.ID),
		token,
		map[string]any{
			"completion_summary": summary,
			"files_modified":     []string{"main.go"},
			"completed_by_agent": "e2e-agent",
			"node_id":            "node-complete-42",
		},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	got := getTask(t, token, projectID, task.ID)
	require.Equal(t, "node-complete-42", got.NodeID)

	dbNodeID := queryNullableString(t, pool, "SELECT node_id FROM tasks WHERE id = $1", task.ID)
	require.NotNil(t, dbNodeID)
	require.Equal(t, "node-complete-42", *dbNodeID)
}

func TestNodeID_BlockTask(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)
	pool := testPool(t)

	task := createTask(t, token, projectID, nil)
	moveTask(t, token, projectID, task.ID, "in_progress")

	reason := strings.Repeat("Blocked: waiting for external dependency. ", 3)
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/block", projectID, task.ID),
		token,
		map[string]any{
			"blocked_reason":   reason,
			"blocked_by_agent": "e2e-blocker",
			"node_id":          "node-block-99",
		},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	got := getTask(t, token, projectID, task.ID)
	require.Equal(t, "node-block-99", got.NodeID)
	require.True(t, got.IsBlocked)

	dbNodeID := queryNullableString(t, pool, "SELECT node_id FROM tasks WHERE id = $1", task.ID)
	require.NotNil(t, dbNodeID)
	require.Equal(t, "node-block-99", *dbNodeID)
}

func TestNodeID_WontDo(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	task := createTask(t, token, projectID, nil)
	moveTask(t, token, projectID, task.ID, "in_progress")

	reason := strings.Repeat("This task is no longer needed due to scope change. ", 2)
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/wont-do", projectID, task.ID),
		token,
		map[string]any{
			"wont_do_reason":       reason,
			"wont_do_requested_by": "e2e-human",
			"node_id":              "node-wontdo-7",
		},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	got := getTask(t, token, projectID, task.ID)
	require.Equal(t, "node-wontdo-7", got.NodeID)
	require.True(t, got.WontDoRequested)
}

// ---------------------------------------------------------------------------
// node_id tracking on feature status changes
// ---------------------------------------------------------------------------

func TestNodeID_FeatureStatusUpdate(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)
	pool := testPool(t)

	type feature struct {
		ID     string `json:"id"`
		NodeID string `json:"node_id"`
		Status string `json:"status"`
	}

	created := createAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{"name": "NodeID Feature", "description": "node_id test"})
	require.Empty(t, created.NodeID, "node_id should be empty on creation")

	statusPath := fmt.Sprintf("/api/projects/%s/features/%s/status", projectID, created.ID)
	featurePath := fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID)

	// Update status with node_id
	patchAndDecode[struct {
		Message string `json:"message"`
	}](t, statusPath, token,
		map[string]any{"status": "ready", "node_id": "node-feat-1"})

	got := getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "ready", got.Status)
	require.Equal(t, "node-feat-1", got.NodeID)

	// Verify in DB
	dbNodeID := queryNullableString(t, pool, "SELECT node_id FROM features WHERE id = $1", created.ID)
	require.NotNil(t, dbNodeID)
	require.Equal(t, "node-feat-1", *dbNodeID)

	// Update status again with a different node_id
	patchAndDecode[struct {
		Message string `json:"message"`
	}](t, statusPath, token,
		map[string]any{"status": "in_progress", "node_id": "node-feat-2"})

	got = getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "in_progress", got.Status)
	require.Equal(t, "node-feat-2", got.NodeID)
}

func TestNodeID_FeatureStatus_PreservesWhenEmpty(t *testing.T) {
	token := adminToken(t)
	projectID := createProjectForFeatures(t, token)

	type feature struct {
		ID     string `json:"id"`
		NodeID string `json:"node_id"`
		Status string `json:"status"`
	}

	created := createAndDecode[feature](t,
		fmt.Sprintf("/api/projects/%s/features", projectID), token,
		map[string]any{"name": "NodeID Preserve Feature", "description": "preserve test"})

	statusPath := fmt.Sprintf("/api/projects/%s/features/%s/status", projectID, created.ID)
	featurePath := fmt.Sprintf("/api/projects/%s/features/%s", projectID, created.ID)

	// Set node_id on first status update
	patchAndDecode[struct {
		Message string `json:"message"`
	}](t, statusPath, token,
		map[string]any{"status": "ready", "node_id": "node-preserve"})

	// Update status without node_id — should preserve
	patchAndDecode[struct {
		Message string `json:"message"`
	}](t, statusPath, token,
		map[string]any{"status": "in_progress"})

	got := getAndDecode[feature](t, featurePath, token)
	require.Equal(t, "in_progress", got.Status)
	require.Equal(t, "node-preserve", got.NodeID, "node_id should be preserved when not provided")
}
