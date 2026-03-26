package e2eapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// taskRespWithSeen extends taskResp with the seen_at field.
type taskRespWithSeen struct {
	taskResp
	SeenAt *time.Time `json:"seen_at"`
}

// ---------------------------------------------------------------------------
// 1. MarkSeen
// ---------------------------------------------------------------------------

func TestTasks_MarkSeen(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	task := createTask(t, token, projectID, nil)

	// Mark the task as seen
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/seen", projectID, task.ID),
		token, nil,
	)
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Verify seen_at is set via GET
	got := getAndDecode[taskRespWithSeen](t,
		fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, task.ID),
		token,
	)
	require.NotNil(t, got.SeenAt, "seen_at should be set after marking task as seen")

	// Calling seen again should be idempotent (still 204, seen_at unchanged)
	firstSeenAt := *got.SeenAt
	resp = doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/seen", projectID, task.ID),
		token, nil,
	)
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	got2 := getAndDecode[taskRespWithSeen](t,
		fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, task.ID),
		token,
	)
	require.NotNil(t, got2.SeenAt)
	require.Equal(t, firstSeenAt.Unix(), got2.SeenAt.Unix(),
		"seen_at should not change on second call (idempotent)")
}

// ---------------------------------------------------------------------------
// 2. UpdateSession
// ---------------------------------------------------------------------------

func TestTasks_UpdateSession(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	task := createTask(t, token, projectID, nil)

	// Update session_id
	sessionID := "ses_" + uniqueSlug("session")
	resp := doAuth(t, "PATCH",
		fmt.Sprintf("/api/projects/%s/tasks/%s/session", projectID, task.ID),
		token,
		map[string]any{"session_id": sessionID},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify via GET
	got := getTask(t, token, projectID, task.ID)
	require.Equal(t, sessionID, got.SessionID, "session_id should match after update")

	// Update to a different session
	newSessionID := "ses_" + uniqueSlug("session2")
	resp = doAuth(t, "PATCH",
		fmt.Sprintf("/api/projects/%s/tasks/%s/session", projectID, task.ID),
		token,
		map[string]any{"session_id": newSessionID},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	got = getTask(t, token, projectID, task.ID)
	require.Equal(t, newSessionID, got.SessionID, "session_id should update to new value")
}

// ---------------------------------------------------------------------------
// 3. SearchTasks
// ---------------------------------------------------------------------------

func TestTasks_SearchTasks(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	keyword := uniqueSlug("searchterm")
	_ = createTask(t, token, projectID, map[string]any{
		"title":   "Alpha " + keyword + " Task",
		"summary": "Task with the search keyword in title",
	})
	_ = createTask(t, token, projectID, map[string]any{
		"title":   "Unrelated Beta Task " + uniqueSlug("other"),
		"summary": "Task that should not match the search",
	})

	// Search by keyword
	results := getAndDecode[[]taskWithDetails](t,
		fmt.Sprintf("/api/projects/%s/tasks/search?q=%s", projectID, keyword),
		token,
	)
	require.GreaterOrEqual(t, len(results), 1, "search should return at least the matching task")
	for _, r := range results {
		require.Contains(t, r.Title, keyword,
			"all search results should contain the keyword in title")
	}
}

// ---------------------------------------------------------------------------
// 4. GetNextTasks
// ---------------------------------------------------------------------------

func TestTasks_GetNextTasks(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	// Create a global agent and assign it to the project.
	agentSlug := uniqueSlug("nextagent")
	resp := doAuth(t, "POST", "/api/agents", token, map[string]any{
		"slug": agentSlug,
		"name": "Next Tasks Agent",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
	t.Cleanup(func() {
		r := doAuth(t, "DELETE", fmt.Sprintf("/api/agents/%s", agentSlug), token, nil)
		r.Body.Close()
	})

	resp = doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/agents", projectID),
		token,
		map[string]any{"agent_slug": agentSlug},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Create a task assigned to that agent (lands in todo by default).
	task := createTask(t, token, projectID, map[string]any{
		"assigned_role": agentSlug,
	})

	// GET next-tasks filtered by role
	type nextTask struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Role      string `json:"role"`
		ProjectID string `json:"project_id"`
		SessionID string `json:"session_id"`
	}
	next := getAndDecode[[]nextTask](t,
		fmt.Sprintf("/api/projects/%s/next-tasks?role=%s", projectID, agentSlug),
		token,
	)
	require.GreaterOrEqual(t, len(next), 1, "next-tasks should return at least one task")

	found := false
	for _, nt := range next {
		if nt.ID == task.ID {
			found = true
			require.Equal(t, agentSlug, nt.Role)
			require.Equal(t, projectID, nt.ProjectID)
			break
		}
	}
	require.True(t, found, "the created task should appear in next-tasks")
}

// ---------------------------------------------------------------------------
// 5. ListTasksByAgent
// ---------------------------------------------------------------------------

func TestTasks_ListTasksByAgent(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	// Create and assign agent to the project.
	agentSlug := uniqueSlug("listagent")
	resp := doAuth(t, "POST", "/api/agents", token, map[string]any{
		"slug": agentSlug,
		"name": "List Tasks Agent",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
	t.Cleanup(func() {
		r := doAuth(t, "DELETE", fmt.Sprintf("/api/agents/%s", agentSlug), token, nil)
		r.Body.Close()
	})

	resp = doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/agents", projectID),
		token,
		map[string]any{"agent_slug": agentSlug},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Create a task assigned to the agent.
	task := createTask(t, token, projectID, map[string]any{
		"assigned_role": agentSlug,
	})

	// GET tasks by agent
	type tasksByAgentResp struct {
		AgentSlug string     `json:"agent_slug"`
		TaskCount int        `json:"task_count"`
		Tasks     []taskResp `json:"tasks"`
	}
	result := getAndDecode[tasksByAgentResp](t,
		fmt.Sprintf("/api/projects/%s/agents/%s/tasks", projectID, agentSlug),
		token,
	)
	require.Equal(t, agentSlug, result.AgentSlug)
	require.GreaterOrEqual(t, result.TaskCount, 1)

	found := false
	for _, tk := range result.Tasks {
		if tk.ID == task.ID {
			found = true
			require.Equal(t, agentSlug, tk.AssignedRole)
			break
		}
	}
	require.True(t, found, "the created task should appear in agent task list")
}

// ---------------------------------------------------------------------------
// 6. Unblock Task
// ---------------------------------------------------------------------------

func TestTasks_Unblock(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)
	pool := testPool(t)

	task := createTask(t, token, projectID, nil)
	moveTask(t, token, projectID, task.ID, "in_progress")

	// Put the task into the blocked column via DB (same pattern as TestTasks_WontDo_Reject).
	blockedColID := queryString(t, pool,
		"SELECT id FROM columns WHERE project_id = $1 AND slug = 'blocked'", projectID)

	blockedReason := "Waiting for external API credentials to be provisioned"
	_, err := pool.Exec(context.Background(),
		`UPDATE tasks SET column_id = $1, is_blocked = 1,
		 blocked_reason = $2, blocked_by_agent = 'e2e-agent', updated_at = NOW()
		 WHERE id = $3`,
		blockedColID, blockedReason, task.ID)
	require.NoError(t, err)

	// Verify task is in blocked column.
	slug := columnSlugForTask(t, token, projectID, task.ID)
	require.Equal(t, "blocked", slug)

	// Unblock the task.
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/unblock", projectID, task.ID),
		token, nil,
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify task moved back to todo and is no longer blocked.
	slug = columnSlugForTask(t, token, projectID, task.ID)
	require.Equal(t, "todo", slug)

	got := getTask(t, token, projectID, task.ID)
	require.False(t, got.IsBlocked, "task should no longer be blocked after unblock")
}

func TestTasks_Unblock_NotBlocked(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	task := createTask(t, token, projectID, nil)

	// Try to unblock a task that is in todo (not blocked) — should fail.
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/unblock", projectID, task.ID),
		token, nil,
	)
	require.NotEqual(t, http.StatusOK, resp.StatusCode,
		"unblocking a task that is not in blocked column should fail")
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// 7. Approve Won't Do
// ---------------------------------------------------------------------------

func TestTasks_ApproveWontDo(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)
	pool := testPool(t)

	task := createTask(t, token, projectID, nil)
	moveTask(t, token, projectID, task.ID, "in_progress")

	// Simulate an agent's RequestWontDo: move to blocked with wont_do_requested.
	blockedColID := queryString(t, pool,
		"SELECT id FROM columns WHERE project_id = $1 AND slug = 'blocked'", projectID)

	wontDoReason := strings.Repeat("This task is infeasible because the upstream API is deprecated. ", 2)
	_, err := pool.Exec(context.Background(),
		`UPDATE tasks SET column_id = $1, is_blocked = 1,
		 wont_do_requested = 1, wont_do_reason = $2,
		 wont_do_requested_by = 'e2e-agent', updated_at = NOW()
		 WHERE id = $3`,
		blockedColID, wontDoReason, task.ID)
	require.NoError(t, err)

	// Approve the won't-do.
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/approve-wont-do", projectID, task.ID),
		token, nil,
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify task moved to done and retains wont_do state.
	slug := columnSlugForTask(t, token, projectID, task.ID)
	require.Equal(t, "done", slug)

	got := getTask(t, token, projectID, task.ID)
	require.True(t, got.WontDoRequested, "wont_do_requested should still be true after approval")
	require.NotNil(t, got.CompletedAt, "completed_at should be set after approve-wont-do")
}

func TestTasks_ApproveWontDo_NotRequested(t *testing.T) {
	token := adminToken(t)
	projectID := createTestProject(t, token)

	task := createTask(t, token, projectID, nil)

	// Try to approve-wont-do on a task that has no pending request — should fail.
	resp := doAuth(t, "POST",
		fmt.Sprintf("/api/projects/%s/tasks/%s/approve-wont-do", projectID, task.ID),
		token, nil,
	)
	require.NotEqual(t, http.StatusOK, resp.StatusCode,
		"approving wont-do on a task without a pending request should fail")
	resp.Body.Close()
}
