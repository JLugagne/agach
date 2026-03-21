package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

func ToPublicSkill(skill domain.Skill) pkgkanban.SkillResponse {
	return pkgkanban.SkillResponse{
		ID:          string(skill.ID),
		Slug:        skill.Slug,
		Name:        skill.Name,
		Description: skill.Description,
		Content:     skill.Content,
		Icon:        skill.Icon,
		Color:       skill.Color,
		SortOrder:   skill.SortOrder,
		CreatedAt:   skill.CreatedAt,
		UpdatedAt:   skill.UpdatedAt,
	}
}

func ToPublicSkills(skills []domain.Skill) []pkgkanban.SkillResponse {
	result := make([]pkgkanban.SkillResponse, len(skills))
	for i, s := range skills {
		result[i] = ToPublicSkill(s)
	}
	return result
}
