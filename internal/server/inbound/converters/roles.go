package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicAgent converts domain.Agent to pkgserver.AgentResponse
func ToPublicAgent(agent domain.Agent) pkgserver.AgentResponse {
	return pkgserver.AgentResponse{
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

// ToPublicAgents converts []domain.Agent to []pkgserver.AgentResponse
func ToPublicAgents(agents []domain.Agent) []pkgserver.AgentResponse {
	result := make([]pkgserver.AgentResponse, len(agents))
	for i, a := range agents {
		result[i] = ToPublicAgent(a)
	}
	return result
}
