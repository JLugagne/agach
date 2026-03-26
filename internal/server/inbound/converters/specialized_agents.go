package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

func ToPublicSpecializedAgent(sa domain.SpecializedAgent, parentSlug string, skillCount int) pkgserver.SpecializedAgentResponse {
	return pkgserver.SpecializedAgentResponse{
		ID:            string(sa.ID),
		ParentAgentID: string(sa.ParentAgentID),
		ParentSlug:    parentSlug,
		Slug:          sa.Slug,
		Name:          sa.Name,
		SkillCount:    skillCount,
		SortOrder:     sa.SortOrder,
		CreatedAt:     sa.CreatedAt,
		UpdatedAt:     sa.UpdatedAt,
	}
}

func ToPublicSpecializedAgents(agents []domain.SpecializedAgent, parentSlug string) []pkgserver.SpecializedAgentResponse {
	result := make([]pkgserver.SpecializedAgentResponse, len(agents))
	for i, a := range agents {
		result[i] = ToPublicSpecializedAgent(a, parentSlug, 0)
	}
	return result
}
