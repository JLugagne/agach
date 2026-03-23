package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicAgent converts domain.Agent to pkgkanban.AgentResponse
func ToPublicAgent(agent domain.Agent) pkgkanban.AgentResponse {
	return pkgkanban.AgentResponse{
		ID:             string(agent.ID),
		Slug:           agent.Slug,
		Name:           agent.Name,
		Icon:           agent.Icon,
		Color:          agent.Color,
		Description:    agent.Description,
		TechStack:      agent.TechStack,
		PromptHint:     agent.PromptHint,
		PromptTemplate: agent.PromptTemplate,
		Content:        agent.Content,
		SortOrder:      agent.SortOrder,
		CreatedAt:      agent.CreatedAt,
	}
}

// ToPublicAgents converts []domain.Agent to []pkgkanban.AgentResponse
func ToPublicAgents(agents []domain.Agent) []pkgkanban.AgentResponse {
	result := make([]pkgkanban.AgentResponse, len(agents))
	for i, a := range agents {
		result[i] = ToPublicAgent(a)
	}
	return result
}
