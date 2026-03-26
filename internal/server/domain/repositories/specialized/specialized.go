package specialized

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

type SpecializedAgentRepository interface {
	Create(ctx context.Context, agent domain.SpecializedAgent) error
	FindByID(ctx context.Context, id domain.SpecializedAgentID) (*domain.SpecializedAgent, error)
	FindBySlug(ctx context.Context, slug string) (*domain.SpecializedAgent, error)
	ListByParent(ctx context.Context, parentID domain.AgentID) ([]domain.SpecializedAgent, error)
	CountByParent(ctx context.Context, parentID domain.AgentID) (int, error)
	Update(ctx context.Context, agent domain.SpecializedAgent) error
	Delete(ctx context.Context, id domain.SpecializedAgentID) error
	ListSkills(ctx context.Context, specializedAgentID domain.SpecializedAgentID) ([]domain.Skill, error)
	SetSkills(ctx context.Context, specializedAgentID domain.SpecializedAgentID, skillIDs []domain.SkillID) error
}
