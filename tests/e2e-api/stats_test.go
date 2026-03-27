package e2eapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Inline response types for stats / info endpoints
// ---------------------------------------------------------------------------

type projectInfoResp struct {
	Project     projectResponse        `json:"project"`
	TaskSummary projectSummaryResponse `json:"task_summary"`
	Children    []projectWithSummary   `json:"children"`
	Breadcrumb  []projectResponse      `json:"breadcrumb"`
}

type timelineEntryResp struct {
	Date           string `json:"date"`
	TasksCreated   int    `json:"tasks_created"`
	TasksCompleted int    `json:"tasks_completed"`
}

type coldStartStatResp struct {
	AssignedRole       string  `json:"assigned_role"`
	Count              int     `json:"count"`
	MinInputTokens     int     `json:"min_input_tokens"`
	MaxInputTokens     int     `json:"max_input_tokens"`
	AvgInputTokens     float64 `json:"avg_input_tokens"`
	MinOutputTokens    int     `json:"min_output_tokens"`
	MaxOutputTokens    int     `json:"max_output_tokens"`
	AvgOutputTokens    float64 `json:"avg_output_tokens"`
	MinCacheReadTokens int     `json:"min_cache_read_tokens"`
	MaxCacheReadTokens int     `json:"max_cache_read_tokens"`
	AvgCacheReadTokens float64 `json:"avg_cache_read_tokens"`
}

type modelTokenStatResp struct {
	Model            string `json:"model"`
	TaskCount        int    `json:"task_count"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	CacheReadTokens  int    `json:"cache_read_tokens"`
	CacheWriteTokens int    `json:"cache_write_tokens"`
}

type modelPricingResp struct {
	ID                   string  `json:"id"`
	ModelID              string  `json:"model_id"`
	InputPricePer1M      float64 `json:"input_price_per_1m"`
	OutputPricePer1M     float64 `json:"output_price_per_1m"`
	CacheReadPricePer1M  float64 `json:"cache_read_price_per_1m"`
	CacheWritePricePer1M float64 `json:"cache_write_price_per_1m"`
}

type toolUsageStatResp struct {
	ToolName       string `json:"tool_name"`
	ExecutionCount int    `json:"execution_count"`
}

type bulkReassignResp struct {
	UpdatedCount int `json:"updated_count"`
}

// ---------------------------------------------------------------------------
// 1. TestProjectInfo
// ---------------------------------------------------------------------------

func TestProjectInfo(t *testing.T) {
	tok := adminToken(t)

	proj := createProject(t, tok, map[string]any{
		"name":        "E2E ProjectInfo",
		"description": "project info test",
	})
	t.Cleanup(func() { deleteProject(t, tok, proj.ID) })

	info := getAndDecode[projectInfoResp](t,
		fmt.Sprintf("/api/projects/%s/info", proj.ID), tok)

	require.Equal(t, proj.ID, info.Project.ID)
	require.Equal(t, "E2E ProjectInfo", info.Project.Name)
	require.NotNil(t, info.Children, "children should be non-nil (possibly empty)")
	require.NotNil(t, info.Breadcrumb, "breadcrumb should be non-nil")
}

// ---------------------------------------------------------------------------
// 2. TestTimeline
// ---------------------------------------------------------------------------

func TestTimeline(t *testing.T) {
	tok := adminToken(t)

	proj := createProject(t, tok, map[string]any{
		"name": "E2E Timeline",
	})
	t.Cleanup(func() { deleteProject(t, tok, proj.ID) })

	// Create a task so the project has some activity.
	createTask(t, tok, proj.ID, nil)

	// GET timeline — may be empty array, just verify 200.
	resp := doAuth(t, "GET",
		fmt.Sprintf("/api/projects/%s/stats/timeline", proj.ID), tok, nil)
	requireStatus(t, resp, http.StatusOK)

	var entries []timelineEntryResp
	env := decode[json.RawMessage](t, resp)
	require.NoError(t, json.Unmarshal(env, &entries))
}

// ---------------------------------------------------------------------------
// 3. TestColdStartStats
// ---------------------------------------------------------------------------

func TestColdStartStats(t *testing.T) {
	tok := adminToken(t)

	proj := createProject(t, tok, map[string]any{
		"name": "E2E ColdStart",
	})
	t.Cleanup(func() { deleteProject(t, tok, proj.ID) })

	// GET cold-start stats — empty project returns empty array, just check 200.
	resp := doAuth(t, "GET",
		fmt.Sprintf("/api/projects/%s/stats/cold-start", proj.ID), tok, nil)
	requireStatus(t, resp, http.StatusOK)

	var stats []coldStartStatResp
	env := decode[json.RawMessage](t, resp)
	require.NoError(t, json.Unmarshal(env, &stats))
}

// ---------------------------------------------------------------------------
// 4. TestModelTokenStats
// ---------------------------------------------------------------------------

func TestModelTokenStats(t *testing.T) {
	tok := adminToken(t)

	proj := createProject(t, tok, map[string]any{
		"name": "E2E ModelTokens",
	})
	t.Cleanup(func() { deleteProject(t, tok, proj.ID) })

	// GET model-tokens stats — empty project returns empty array, just check 200.
	resp := doAuth(t, "GET",
		fmt.Sprintf("/api/projects/%s/stats/model-tokens", proj.ID), tok, nil)
	requireStatus(t, resp, http.StatusOK)

	var stats []modelTokenStatResp
	env := decode[json.RawMessage](t, resp)
	require.NoError(t, json.Unmarshal(env, &stats))
}

// ---------------------------------------------------------------------------
// 5. TestModelPricing
// ---------------------------------------------------------------------------

func TestModelPricing(t *testing.T) {
	tok := adminToken(t)

	// GET model-pricing — no project needed.
	resp := doAuth(t, "GET", "/api/model-pricing", tok, nil)
	requireStatus(t, resp, http.StatusOK)

	var pricing []modelPricingResp
	env := decode[json.RawMessage](t, resp)
	require.NoError(t, json.Unmarshal(env, &pricing))
}

// ---------------------------------------------------------------------------
// 6. TestToolUsage
// ---------------------------------------------------------------------------

func TestToolUsage(t *testing.T) {
	tok := adminToken(t)

	proj := createProject(t, tok, map[string]any{
		"name": "E2E ToolUsage",
	})
	t.Cleanup(func() { deleteProject(t, tok, proj.ID) })

	// GET tool-usage — empty project returns empty array, just check 200.
	resp := doAuth(t, "GET",
		fmt.Sprintf("/api/projects/%s/tool-usage", proj.ID), tok, nil)
	requireStatus(t, resp, http.StatusOK)

	var stats []toolUsageStatResp
	env := decode[json.RawMessage](t, resp)
	require.NoError(t, json.Unmarshal(env, &stats))
}

// ---------------------------------------------------------------------------
// 7. TestSpecializedAgentSkills
// ---------------------------------------------------------------------------

func TestSpecializedAgentSkills(t *testing.T) {
	tok := adminToken(t)

	parentSlug := uniqueSlug("stats-agent")
	skillSlug := uniqueSlug("stats-skill")
	specSlug := uniqueSlug("stats-spec")

	// Create parent agent.
	createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":       parentSlug,
		"name":       "Stats Parent Agent",
		"sort_order": 1,
	})
	t.Cleanup(func() { deleteResource(t, "/api/agents/"+parentSlug, tok) })

	// Create skill.
	skill := createAndDecode[skillResp](t, "/api/skills", tok, map[string]any{
		"slug":       skillSlug,
		"name":       "Stats Skill",
		"content":    "skill content for stats test",
		"sort_order": 1,
	})
	t.Cleanup(func() { deleteResource(t, "/api/skills/"+skillSlug, tok) })

	// Create specialized agent with the skill.
	createAndDecode[specializedResp](t,
		fmt.Sprintf("/api/agents/%s/specialized", parentSlug), tok, map[string]any{
			"slug":        specSlug,
			"name":        "Stats Specialized",
			"skill_slugs": []string{skillSlug},
			"sort_order":  1,
		})
	t.Cleanup(func() {
		deleteResource(t, fmt.Sprintf("/api/agents/%s/specialized/%s", parentSlug, specSlug), tok)
	})

	// GET specialized agent skills.
	skills := getAndDecode[[]skillResp](t,
		fmt.Sprintf("/api/agents/%s/specialized/%s/skills", parentSlug, specSlug), tok)

	require.GreaterOrEqual(t, len(skills), 1, "specialized agent should have at least 1 skill")
	found := false
	for _, s := range skills {
		if s.Slug == skillSlug {
			found = true
			assert.Equal(t, skill.Name, s.Name)
			break
		}
	}
	require.True(t, found, "expected skill %q in specialized agent skills list", skillSlug)
}

// ---------------------------------------------------------------------------
// 8. TestBulkReassignAgents
// ---------------------------------------------------------------------------

func TestBulkReassignAgents(t *testing.T) {
	tok := adminToken(t)

	// Create project.
	proj := createProject(t, tok, map[string]any{
		"name": "E2E BulkReassign",
	})
	t.Cleanup(func() { deleteProject(t, tok, proj.ID) })

	// Create two agents.
	agent1Slug := uniqueSlug("bulk-a1")
	agent2Slug := uniqueSlug("bulk-a2")

	createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":       agent1Slug,
		"name":       "Bulk Agent 1",
		"sort_order": 1,
	})
	t.Cleanup(func() { deleteResource(t, "/api/agents/"+agent1Slug, tok) })

	createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":       agent2Slug,
		"name":       "Bulk Agent 2",
		"sort_order": 2,
	})
	t.Cleanup(func() { deleteResource(t, "/api/agents/"+agent2Slug, tok) })

	// Assign both agents to the project.
	resp := doAuth(t, "POST", fmt.Sprintf("/api/projects/%s/agents", proj.ID), tok, map[string]any{
		"agent_slug": agent1Slug,
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	resp = doAuth(t, "POST", fmt.Sprintf("/api/projects/%s/agents", proj.ID), tok, map[string]any{
		"agent_slug": agent2Slug,
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Create a task assigned to agent1.
	task := createTask(t, tok, proj.ID, map[string]any{
		"assigned_role": agent1Slug,
	})

	// Verify task is assigned to agent1.
	got := getTask(t, tok, proj.ID, task.ID)
	require.Equal(t, agent1Slug, got.AssignedRole)

	// Bulk reassign from agent1 to agent2.
	result := createAndDecode[bulkReassignResp](t,
		fmt.Sprintf("/api/projects/%s/agents/bulk-reassign", proj.ID), tok, map[string]any{
			"old_slug": agent1Slug,
			"new_slug": agent2Slug,
		})
	require.GreaterOrEqual(t, result.UpdatedCount, 1, "at least 1 task should be reassigned")

	// Verify the task is now assigned to agent2.
	reassigned := getTask(t, tok, proj.ID, task.ID)
	require.Equal(t, agent2Slug, reassigned.AssignedRole)
}
