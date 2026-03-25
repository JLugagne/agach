package converters_test

import (
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/stretchr/testify/assert"
)

func TestToPublicAgent(t *testing.T) {
	agent := domain.Agent{
		ID:          domain.AgentID("role-123"),
		Slug:        "architect",
		Name:        "System Architect",
		Icon:        "📐",
		Color:       "#3B82F6",
		Description: "Designs systems",
		TechStack:   []string{"Go", "PostgreSQL"},
		PromptHint:  "Focus on clean code",
		SortOrder:   1,
	}

	result := converters.ToPublicAgent(agent)

	assert.Equal(t, "role-123", result.ID)
	assert.Equal(t, "architect", result.Slug)
	assert.Equal(t, "System Architect", result.Name)
	assert.Equal(t, "📐", result.Icon)
	assert.Equal(t, "#3B82F6", result.Color)
	assert.Equal(t, "Designs systems", result.Description)
	assert.Equal(t, []string{"Go", "PostgreSQL"}, result.TechStack)
	assert.Equal(t, "Focus on clean code", result.PromptHint)
	assert.Equal(t, 1, result.SortOrder)
}

func TestToPublicAgents(t *testing.T) {
	agents := []domain.Agent{
		{ID: domain.AgentID("role-1"), Slug: "dev", Name: "Developer"},
		{ID: domain.AgentID("role-2"), Slug: "arch", Name: "Architect"},
	}

	result := converters.ToPublicAgents(agents)

	assert.Len(t, result, 2)
	assert.Equal(t, "role-1", result[0].ID)
	assert.Equal(t, "role-2", result[1].ID)
}
