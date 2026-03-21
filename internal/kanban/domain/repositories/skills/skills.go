package skills

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// SkillRepository defines operations for managing skills
type SkillRepository interface {
	// Create creates a new skill
	Create(ctx context.Context, skill domain.Skill) error

	// FindByID retrieves a skill by ID
	FindByID(ctx context.Context, id domain.SkillID) (*domain.Skill, error)

	// FindBySlug retrieves a skill by slug
	FindBySlug(ctx context.Context, slug string) (*domain.Skill, error)

	// List retrieves all skills ordered by sort_order, then name
	List(ctx context.Context) ([]domain.Skill, error)

	// Update updates an existing skill (full replace of mutable fields)
	Update(ctx context.Context, skill domain.Skill) error

	// Delete deletes a skill by ID
	// Returns ErrSkillInUse if the skill is still referenced by agent_skills rows
	Delete(ctx context.Context, id domain.SkillID) error

	// IsInUse checks whether any agent has this skill assigned
	IsInUse(ctx context.Context, id domain.SkillID) (bool, error)

	// ListByAgent returns all skills assigned to a given role ID, ordered by sort_order
	ListByAgent(ctx context.Context, roleID domain.RoleID) ([]domain.Skill, error)

	// AssignToAgent creates an agent_skills row
	// Returns ErrSkillAlreadyExists if the association already exists
	AssignToAgent(ctx context.Context, roleID domain.RoleID, skillID domain.SkillID) error

	// RemoveFromAgent deletes an agent_skills row
	// Returns ErrSkillNotFound if the association does not exist
	RemoveFromAgent(ctx context.Context, roleID domain.RoleID, skillID domain.SkillID) error
}
