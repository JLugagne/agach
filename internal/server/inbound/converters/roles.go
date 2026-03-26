package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicAgent converts domain.Agent to pkgserver.AgentResponse
func ToPublicAgent(agent domain.Agent) pkgserver.AgentResponse {
	return ToPublicAgentWithCount(agent, 0, 0)
}

// ToPublicAgentWithCount converts domain.Agent to pkgserver.AgentResponse with counts
func ToPublicAgentWithCount(agent domain.Agent, skillCount, specializedCount int) pkgserver.AgentResponse {
	return pkgserver.AgentResponse{
		ID:               string(agent.ID),
		Slug:             agent.Slug,
		Name:             agent.Name,
		Icon:             agent.Icon,
		Color:            agent.Color,
		Description:      agent.Description,
		TechStack:        agent.TechStack,
		PromptHint:       agent.PromptHint,
		PromptTemplate:   agent.PromptTemplate,
		Content:          agent.Content,
		Model:            agent.Model,
		Thinking:         agent.Thinking,
		SkillCount:       skillCount,
		SpecializedCount: specializedCount,
		SortOrder:        agent.SortOrder,
		CreatedAt:        agent.CreatedAt,
	}
}

// ToPublicAgents converts []domain.Agent to []pkgserver.AgentResponse
func ToPublicAgents(agents []domain.Agent) []pkgserver.AgentResponse {
	return MapSlice(agents, ToPublicAgent)
}
