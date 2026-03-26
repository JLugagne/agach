package e2eapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Inline response structs
// ---------------------------------------------------------------------------

type agentResp struct {
	ID               string  `json:"id"`
	Slug             string  `json:"slug"`
	Name             string  `json:"name"`
	Icon             *string `json:"icon"`
	Color            *string `json:"color"`
	Description      *string `json:"description"`
	TechStack        *string `json:"tech_stack"`
	PromptHint       *string `json:"prompt_hint"`
	PromptTemplate   *string `json:"prompt_template"`
	Model            *string `json:"model"`
	Thinking         *bool   `json:"thinking"`
	SkillCount       int     `json:"skill_count"`
	SpecializedCount int     `json:"specialized_count"`
	SortOrder        int     `json:"sort_order"`
	CreatedAt        string  `json:"created_at"`
}

type skillResp struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Content     *string `json:"content"`
	Icon        *string `json:"icon"`
	Color       *string `json:"color"`
	SortOrder   int     `json:"sort_order"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type specializedResp struct {
	ID            string `json:"id"`
	ParentAgentID string `json:"parent_agent_id"`
	ParentSlug    string `json:"parent_slug"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	SkillCount    int    `json:"skill_count"`
	SortOrder     int    `json:"sort_order"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// 1. TestAgents_CRUD
// ---------------------------------------------------------------------------

func TestAgents_CRUD(t *testing.T) {
	tok := adminToken(t)
	slug := uniqueSlug("agent-crud")

	// CREATE
	created := createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":        slug,
		"name":        "CRUD Agent",
		"icon":        "robot",
		"color":       "#FF0000",
		"description": "An agent for CRUD testing",
		"tech_stack":  "Go",
		"sort_order":  10,
	})
	require.NotEmpty(t, created.ID)
	assert.Equal(t, slug, created.Slug)
	assert.Equal(t, "CRUD Agent", created.Name)

	// GET by slug
	fetched := getAndDecode[agentResp](t, "/api/agents/"+slug, tok)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, slug, fetched.Slug)
	assert.Equal(t, "CRUD Agent", fetched.Name)

	// LIST — must contain the created agent
	agents := getAndDecode[[]agentResp](t, "/api/agents", tok)
	found := false
	for _, a := range agents {
		if a.Slug == slug {
			found = true
			break
		}
	}
	require.True(t, found, "created agent not found in list")

	// UPDATE name
	updated := patchAndDecode[agentResp](t, "/api/agents/"+slug, tok, map[string]any{
		"name": ptr("Updated CRUD Agent"),
	})
	assert.Equal(t, "Updated CRUD Agent", updated.Name)
	assert.Equal(t, slug, updated.Slug) // slug unchanged

	// Verify update via GET
	refetched := getAndDecode[agentResp](t, "/api/agents/"+slug, tok)
	assert.Equal(t, "Updated CRUD Agent", refetched.Name)

	// DELETE
	deleteResource(t, "/api/agents/"+slug, tok)

	// Verify gone — expect 404
	resp := doAuth(t, "GET", "/api/agents/"+slug, tok, nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// 2. TestAgents_Clone
// ---------------------------------------------------------------------------

func TestAgents_Clone(t *testing.T) {
	tok := adminToken(t)
	srcSlug := uniqueSlug("agent-clone-src")
	dstSlug := uniqueSlug("agent-clone-dst")

	// Create source agent
	createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":        srcSlug,
		"name":        "Source Agent",
		"icon":        "copy",
		"color":       "#00FF00",
		"description": "Source for cloning",
		"sort_order":  1,
	})

	// Clone
	cloned := createAndDecode[agentResp](t, fmt.Sprintf("/api/agents/%s/clone", srcSlug), tok, map[string]any{
		"new_slug": dstSlug,
		"new_name": "Cloned Agent",
	})
	require.NotEmpty(t, cloned.ID)
	assert.Equal(t, dstSlug, cloned.Slug)
	assert.Equal(t, "Cloned Agent", cloned.Name)

	// Verify clone exists independently
	fetched := getAndDecode[agentResp](t, "/api/agents/"+dstSlug, tok)
	assert.Equal(t, dstSlug, fetched.Slug)
	assert.Equal(t, "Cloned Agent", fetched.Name)

	// Cleanup
	deleteResource(t, "/api/agents/"+srcSlug, tok)
	deleteResource(t, "/api/agents/"+dstSlug, tok)
}

// ---------------------------------------------------------------------------
// 3. TestSkills_CRUD
// ---------------------------------------------------------------------------

func TestSkills_CRUD(t *testing.T) {
	tok := adminToken(t)
	slug := uniqueSlug("skill-crud")

	// CREATE
	created := createAndDecode[skillResp](t, "/api/skills", tok, map[string]any{
		"slug":        slug,
		"name":        "CRUD Skill",
		"description": "A skill for CRUD testing",
		"content":     "Some content here",
		"icon":        "wrench",
		"color":       "#0000FF",
		"sort_order":  5,
	})
	require.NotEmpty(t, created.ID)
	assert.Equal(t, slug, created.Slug)
	assert.Equal(t, "CRUD Skill", created.Name)

	// GET by slug
	fetched := getAndDecode[skillResp](t, "/api/skills/"+slug, tok)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, slug, fetched.Slug)

	// LIST — must contain the created skill
	skills := getAndDecode[[]skillResp](t, "/api/skills", tok)
	found := false
	for _, s := range skills {
		if s.Slug == slug {
			found = true
			break
		}
	}
	require.True(t, found, "created skill not found in list")

	// UPDATE
	updated := patchAndDecode[skillResp](t, "/api/skills/"+slug, tok, map[string]any{
		"name": ptr("Updated Skill"),
	})
	assert.Equal(t, "Updated Skill", updated.Name)

	// DELETE
	deleteResource(t, "/api/skills/"+slug, tok)

	// Verify gone
	resp := doAuth(t, "GET", "/api/skills/"+slug, tok, nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// 4. TestAgentSkills_AssignAndRemove
// ---------------------------------------------------------------------------

func TestAgentSkills_AssignAndRemove(t *testing.T) {
	tok := adminToken(t)
	agentSlug := uniqueSlug("agent-sk")
	skillSlug := uniqueSlug("skill-sk")

	// Create agent
	createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":       agentSlug,
		"name":       "Agent With Skills",
		"sort_order": 1,
	})

	// Create skill
	createAndDecode[skillResp](t, "/api/skills", tok, map[string]any{
		"slug":       skillSlug,
		"name":       "Assignable Skill",
		"content":    "skill content",
		"sort_order": 1,
	})

	// Assign skill to agent
	resp := doAuth(t, "POST", fmt.Sprintf("/api/agents/%s/skills", agentSlug), tok, map[string]any{
		"skill_slug": skillSlug,
	})
	requireStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// List agent skills — should contain the skill
	agentSkills := getAndDecode[[]skillResp](t, fmt.Sprintf("/api/agents/%s/skills", agentSlug), tok)
	found := false
	for _, s := range agentSkills {
		if s.Slug == skillSlug {
			found = true
			break
		}
	}
	require.True(t, found, "assigned skill not found in agent's skill list")

	// Verify agent skill_count increased
	agent := getAndDecode[agentResp](t, "/api/agents/"+agentSlug, tok)
	assert.GreaterOrEqual(t, agent.SkillCount, 1)

	// Remove skill from agent
	deleteResource(t, fmt.Sprintf("/api/agents/%s/skills/%s", agentSlug, skillSlug), tok)

	// Verify skill removed from agent
	agentSkillsAfter := getAndDecode[[]skillResp](t, fmt.Sprintf("/api/agents/%s/skills", agentSlug), tok)
	for _, s := range agentSkillsAfter {
		assert.NotEqual(t, skillSlug, s.Slug, "skill should have been removed from agent")
	}

	// Cleanup
	deleteResource(t, "/api/agents/"+agentSlug, tok)
	deleteResource(t, "/api/skills/"+skillSlug, tok)
}

// ---------------------------------------------------------------------------
// 5. TestSpecializedAgents_CRUD
// ---------------------------------------------------------------------------

func TestSpecializedAgents_CRUD(t *testing.T) {
	tok := adminToken(t)
	parentSlug := uniqueSlug("agent-spec")
	skillSlug := uniqueSlug("skill-spec")
	specSlug := uniqueSlug("spec")

	// Create parent agent
	parent := createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":       parentSlug,
		"name":       "Parent Agent",
		"sort_order": 1,
	})

	// Create skill
	createAndDecode[skillResp](t, "/api/skills", tok, map[string]any{
		"slug":       skillSlug,
		"name":       "Spec Skill",
		"content":    "specialized content",
		"sort_order": 1,
	})

	// Create specialized agent with skill
	spec := createAndDecode[specializedResp](t,
		fmt.Sprintf("/api/agents/%s/specialized", parentSlug), tok, map[string]any{
			"slug":        specSlug,
			"name":        "Specialized Agent",
			"skill_slugs": []string{skillSlug},
			"sort_order":  1,
		})
	require.NotEmpty(t, spec.ID)
	assert.Equal(t, specSlug, spec.Slug)
	assert.Equal(t, "Specialized Agent", spec.Name)
	assert.Equal(t, parent.ID, spec.ParentAgentID)
	assert.Equal(t, parentSlug, spec.ParentSlug)

	// GET specialized
	fetched := getAndDecode[specializedResp](t,
		fmt.Sprintf("/api/agents/%s/specialized/%s", parentSlug, specSlug), tok)
	assert.Equal(t, spec.ID, fetched.ID)
	assert.Equal(t, specSlug, fetched.Slug)

	// LIST specialized — must contain the created one
	specList := getAndDecode[[]specializedResp](t,
		fmt.Sprintf("/api/agents/%s/specialized", parentSlug), tok)
	found := false
	for _, s := range specList {
		if s.Slug == specSlug {
			found = true
			break
		}
	}
	require.True(t, found, "specialized agent not found in list")

	// UPDATE name
	updated := patchAndDecode[specializedResp](t,
		fmt.Sprintf("/api/agents/%s/specialized/%s", parentSlug, specSlug), tok, map[string]any{
			"name": ptr("Updated Specialized"),
		})
	assert.Equal(t, "Updated Specialized", updated.Name)

	// DELETE specialized
	deleteResource(t, fmt.Sprintf("/api/agents/%s/specialized/%s", parentSlug, specSlug), tok)

	// Verify gone
	resp := doAuth(t, "GET",
		fmt.Sprintf("/api/agents/%s/specialized/%s", parentSlug, specSlug), tok, nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()

	// Cleanup
	deleteResource(t, "/api/agents/"+parentSlug, tok)
	deleteResource(t, "/api/skills/"+skillSlug, tok)
}

// ---------------------------------------------------------------------------
// 6. TestAgents_DB_Verification
// ---------------------------------------------------------------------------

func TestAgents_DB_Verification(t *testing.T) {
	tok := adminToken(t)
	pool := testPool(t)

	agentSlug := uniqueSlug("agent-db")
	skillSlug := uniqueSlug("skill-db")

	// Create agent
	agent := createAndDecode[agentResp](t, "/api/agents", tok, map[string]any{
		"slug":       agentSlug,
		"name":       "DB Verified Agent",
		"sort_order": 1,
	})
	require.NotEmpty(t, agent.ID)

	// Verify agent exists in "roles" table
	require.True(t, rowExists(t, pool, "roles", "slug = $1", agentSlug),
		"agent should exist in roles table")

	// Create skill
	skill := createAndDecode[skillResp](t, "/api/skills", tok, map[string]any{
		"slug":       skillSlug,
		"name":       "DB Verified Skill",
		"content":    "db check content",
		"sort_order": 1,
	})
	require.NotEmpty(t, skill.ID)

	// Verify skill exists in "skills" table
	require.True(t, rowExists(t, pool, "skills", "slug = $1", skillSlug),
		"skill should exist in skills table")

	// Cleanup
	deleteResource(t, "/api/agents/"+agentSlug, tok)
	deleteResource(t, "/api/skills/"+skillSlug, tok)

	// Verify rows removed after deletion
	require.False(t, rowExists(t, pool, "roles", "slug = $1", agentSlug),
		"agent should be removed from roles table after delete")
	require.False(t, rowExists(t, pool, "skills", "slug = $1", skillSlug),
		"skill should be removed from skills table after delete")
}
