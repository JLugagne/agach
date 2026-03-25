package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

func ToPublicSkill(skill domain.Skill) pkgserver.SkillResponse {
	return pkgserver.SkillResponse{
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

func ToPublicSkills(skills []domain.Skill) []pkgserver.SkillResponse {
	return MapSlice(skills, ToPublicSkill)
}
