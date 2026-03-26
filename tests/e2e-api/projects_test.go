package e2eapi

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Inline types for decoding project-related API responses.
// ---------------------------------------------------------------------------

type projectResponse struct {
	ID             string    `json:"id"`
	ParentID       *string   `json:"parent_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	GitURL         string    `json:"git_url"`
	CreatedByRole  string    `json:"created_by_role"`
	CreatedByAgent string    `json:"created_by_agent"`
	DefaultRole    string    `json:"default_role"`
	DockerfileID   *string   `json:"dockerfile_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type projectWithSummary struct {
	projectResponse
	ChildrenCount int                    `json:"children_count"`
	TaskSummary   projectSummaryResponse `json:"task_summary"`
}

type projectSummaryResponse struct {
	BacklogCount    int `json:"backlog_count"`
	TodoCount       int `json:"todo_count"`
	InProgressCount int `json:"in_progress_count"`
	DoneCount       int `json:"done_count"`
	BlockedCount    int `json:"blocked_count"`
}

type columnResponse struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type columnWithTasks struct {
	columnResponse
	Tasks []taskBrief `json:"tasks"`
}

type taskBrief struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type boardResponse struct {
	Columns []columnWithTasks `json:"columns"`
}

type dockerfileResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Content     string `json:"content"`
}

// ---------------------------------------------------------------------------
// Helpers: POST/PUT/DELETE that expect 200 (all endpoints use SendSuccess).
// ---------------------------------------------------------------------------

func createProject(t *testing.T, token string, body map[string]any) projectResponse {
	t.Helper()
	resp := doAuth(t, "POST", "/api/projects", token, body)
	requireStatus(t, resp, http.StatusOK)
	return decode[projectResponse](t, resp)
}

func deleteProject(t *testing.T, token, id string) {
	t.Helper()
	resp := doAuth(t, "DELETE", fmt.Sprintf("/api/projects/%s", id), token, nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestProjects_CRUD(t *testing.T) {
	token := adminToken(t)

	// Create.
	proj := createProject(t, token, map[string]any{
		"name":        "E2E CRUD Project",
		"description": "Created by e2e test",
		"git_url":     "https://github.com/test/crud",
	})
	require.NotEmpty(t, proj.ID)
	require.Equal(t, "E2E CRUD Project", proj.Name)
	require.Equal(t, "Created by e2e test", proj.Description)
	require.Equal(t, "https://github.com/test/crud", proj.GitURL)

	// Get by ID.
	got := getAndDecode[projectResponse](t, fmt.Sprintf("/api/projects/%s", proj.ID), token)
	require.Equal(t, proj.ID, got.ID)
	require.Equal(t, proj.Name, got.Name)

	// List: project should appear.
	list := getAndDecode[[]projectWithSummary](t, "/api/projects", token)
	found := false
	for _, p := range list {
		if p.ID == proj.ID {
			found = true
			break
		}
	}
	require.True(t, found, "created project should appear in list")

	// Update name.
	resp := doAuth(t, "PATCH", fmt.Sprintf("/api/projects/%s", proj.ID), token, map[string]any{
		"name": "E2E CRUD Updated",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify update.
	updated := getAndDecode[projectResponse](t, fmt.Sprintf("/api/projects/%s", proj.ID), token)
	require.Equal(t, "E2E CRUD Updated", updated.Name)
	require.Equal(t, proj.Description, updated.Description) // unchanged

	// Delete.
	deleteProject(t, token, proj.ID)

	// Verify gone from list.
	listAfter := getAndDecode[[]projectWithSummary](t, "/api/projects", token)
	for _, p := range listAfter {
		require.NotEqual(t, proj.ID, p.ID, "deleted project should not appear in list")
	}
}

func TestProjects_SubProject(t *testing.T) {
	token := adminToken(t)

	// Create parent.
	parent := createProject(t, token, map[string]any{
		"name":        "E2E Parent Project",
		"description": "parent",
	})
	t.Cleanup(func() { deleteProject(t, token, parent.ID) })

	// Create child.
	child := createProject(t, token, map[string]any{
		"name":        "E2E Child Project",
		"description": "child",
		"parent_id":   parent.ID,
	})
	require.NotNil(t, child.ParentID)
	require.Equal(t, parent.ID, *child.ParentID)

	// Verify GET children returns child.
	children := getAndDecode[[]projectWithSummary](t, fmt.Sprintf("/api/projects/%s/children", parent.ID), token)
	require.GreaterOrEqual(t, len(children), 1)
	found := false
	for _, c := range children {
		if c.ID == child.ID {
			found = true
			require.Equal(t, "E2E Child Project", c.Name)
			break
		}
	}
	require.True(t, found, "child project should appear in parent's children")

	// Cleanup child first (cascade would handle it, but be explicit).
	deleteProject(t, token, child.ID)
}

func TestProjects_Summary(t *testing.T) {
	token := adminToken(t)

	proj := createProject(t, token, map[string]any{
		"name": "E2E Summary Project",
	})
	t.Cleanup(func() { deleteProject(t, token, proj.ID) })

	// Create a few tasks in the project.
	for i := range 3 {
		resp := doAuth(t, "POST", fmt.Sprintf("/api/projects/%s/tasks", proj.ID), token, map[string]any{
			"title":    fmt.Sprintf("Summary task %d", i),
			"summary":  fmt.Sprintf("Summary for task %d", i),
			"priority": "medium",
		})
		requireStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	}

	// Check summary counts.
	summary := getAndDecode[projectSummaryResponse](t, fmt.Sprintf("/api/projects/%s/summary", proj.ID), token)
	total := summary.BacklogCount + summary.TodoCount + summary.InProgressCount + summary.DoneCount + summary.BlockedCount
	require.GreaterOrEqual(t, total, 3, "summary should reflect at least 3 tasks")
}

func TestProjects_Board(t *testing.T) {
	token := adminToken(t)

	proj := createProject(t, token, map[string]any{
		"name": "E2E Board Project",
	})
	t.Cleanup(func() { deleteProject(t, token, proj.ID) })

	// Create a task so there is something on the board.
	resp := doAuth(t, "POST", fmt.Sprintf("/api/projects/%s/tasks", proj.ID), token, map[string]any{
		"title":    "Board task",
		"summary":  "Board task summary",
		"priority": "high",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Get board.
	board := getAndDecode[boardResponse](t, fmt.Sprintf("/api/projects/%s/board", proj.ID), token)
	require.GreaterOrEqual(t, len(board.Columns), 4, "board should have at least 4 columns")

	// Verify expected column slugs exist (project creation seeds: todo, in_progress, done, blocked).
	expectedSlugs := map[string]bool{
		"todo":        false,
		"in_progress": false,
		"done":        false,
		"blocked":     false,
	}
	for _, col := range board.Columns {
		if _, ok := expectedSlugs[col.Slug]; ok {
			expectedSlugs[col.Slug] = true
		}
	}
	for slug, found := range expectedSlugs {
		require.True(t, found, "expected column %q not found on board", slug)
	}

	// Verify at least one column has tasks.
	totalTasks := 0
	for _, col := range board.Columns {
		totalTasks += len(col.Tasks)
	}
	require.GreaterOrEqual(t, totalTasks, 1, "board should contain at least 1 task")

	// Also verify GET columns endpoint.
	columns := getAndDecode[[]columnResponse](t, fmt.Sprintf("/api/projects/%s/columns", proj.ID), token)
	require.GreaterOrEqual(t, len(columns), 4)
}

func TestProjects_Agents(t *testing.T) {
	token := adminToken(t)

	// Create a project first.
	proj := createProject(t, token, map[string]any{
		"name": "E2E Agents Project",
	})
	t.Cleanup(func() { deleteProject(t, token, proj.ID) })

	// Create a global agent.
	agentSlug := uniqueSlug("e2e-agent")
	resp := doAuth(t, "POST", "/api/agents", token, map[string]any{
		"slug": agentSlug,
		"name": "E2E Test Agent",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()
	t.Cleanup(func() {
		r := doAuth(t, "DELETE", fmt.Sprintf("/api/agents/%s", agentSlug), token, nil)
		r.Body.Close()
	})

	// Assign agent to project (writes to project_agents, which ListByProject reads).
	resp = doAuth(t, "POST", fmt.Sprintf("/api/projects/%s/agents", proj.ID), token, map[string]any{
		"agent_slug": agentSlug,
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify agent was assigned by checking the DB directly.
	pool := testPool(t)
	require.True(t, rowExists(t, pool,
		"project_agents", "project_id = $1", proj.ID),
		"project_agents should have at least one row after assignment")

	// Remove agent from project.
	resp = doAuth(t, "DELETE", fmt.Sprintf("/api/projects/%s/agents/%s", proj.ID, agentSlug), token, map[string]any{
		"clear_assignment": true,
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify agent removed from DB.
	require.False(t, rowExists(t, pool,
		"project_agents", "project_id = $1", proj.ID),
		"project_agents should be empty after removal")
}

func TestProjects_Dockerfile(t *testing.T) {
	token := adminToken(t)

	// Create a project.
	proj := createProject(t, token, map[string]any{
		"name": "E2E Dockerfile Project",
	})
	t.Cleanup(func() { deleteProject(t, token, proj.ID) })

	// Create a dockerfile.
	dfSlug := uniqueSlug("e2e-df")
	dfResp := doAuth(t, "POST", "/api/dockerfiles", token, map[string]any{
		"slug":        dfSlug,
		"name":        "E2E Dockerfile",
		"description": "test dockerfile",
		"version":     "1.0.0",
		"content":     "FROM ubuntu:22.04",
		"is_latest":   true,
	})
	requireStatus(t, dfResp, http.StatusOK)
	df := decode[dockerfileResponse](t, dfResp)
	require.NotEmpty(t, df.ID)
	t.Cleanup(func() {
		r := doAuth(t, "DELETE", fmt.Sprintf("/api/dockerfiles/%s", df.ID), token, nil)
		r.Body.Close()
	})

	// Assign dockerfile to project.
	resp := doAuth(t, "PUT", fmt.Sprintf("/api/projects/%s/dockerfile", proj.ID), token, map[string]any{
		"dockerfile_id": df.ID,
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify GET project dockerfile.
	gotDf := getAndDecode[dockerfileResponse](t, fmt.Sprintf("/api/projects/%s/dockerfile", proj.ID), token)
	require.Equal(t, df.ID, gotDf.ID)
	require.Equal(t, dfSlug, gotDf.Slug)

	// Also verify the project itself has dockerfile_id set.
	gotProj := getAndDecode[projectResponse](t, fmt.Sprintf("/api/projects/%s", proj.ID), token)
	require.NotNil(t, gotProj.DockerfileID)
	require.Equal(t, df.ID, *gotProj.DockerfileID)

	// Clear dockerfile.
	resp = doAuth(t, "DELETE", fmt.Sprintf("/api/projects/%s/dockerfile", proj.ID), token, nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify cleared.
	gotProjAfter := getAndDecode[projectResponse](t, fmt.Sprintf("/api/projects/%s", proj.ID), token)
	require.Nil(t, gotProjAfter.DockerfileID)
}

func TestProjects_DB_Verification(t *testing.T) {
	token := adminToken(t)
	pool := testPool(t)

	proj := createProject(t, token, map[string]any{
		"name":        "E2E DB Verification Project",
		"description": "verify row in pg",
		"git_url":     "https://github.com/test/dbcheck",
	})
	t.Cleanup(func() { deleteProject(t, token, proj.ID) })

	// Verify project exists in the projects table.
	require.True(t, rowExists(t, pool, "projects", "id = $1", proj.ID),
		"project should exist in projects table")

	// Verify name matches.
	name := queryString(t, pool, "SELECT name FROM projects WHERE id = $1", proj.ID)
	require.Equal(t, "E2E DB Verification Project", name)

	// Verify description.
	desc := queryString(t, pool, "SELECT description FROM projects WHERE id = $1", proj.ID)
	require.Equal(t, "verify row in pg", desc)

	// Verify git_url.
	gitURL := queryString(t, pool, "SELECT git_url FROM projects WHERE id = $1", proj.ID)
	require.Equal(t, "https://github.com/test/dbcheck", gitURL)
}
