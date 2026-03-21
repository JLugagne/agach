package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicRole converts domain.Role to pkgkanban.RoleResponse
func ToPublicRole(role domain.Role) pkgkanban.RoleResponse {
	return pkgkanban.RoleResponse{
		ID:             string(role.ID),
		Slug:           role.Slug,
		Name:           role.Name,
		Icon:           role.Icon,
		Color:          role.Color,
		Description:    role.Description,
		TechStack:      role.TechStack,
		PromptHint:     role.PromptHint,
		PromptTemplate: role.PromptTemplate,
		Content:        role.Content,
		SortOrder:      role.SortOrder,
		CreatedAt:      role.CreatedAt,
	}
}

// ToPublicRoles converts []domain.Role to []pkgkanban.RoleResponse
func ToPublicRoles(roles []domain.Role) []pkgkanban.RoleResponse {
	result := make([]pkgkanban.RoleResponse, len(roles))
	for i, r := range roles {
		result[i] = ToPublicRole(r)
	}
	return result
}
