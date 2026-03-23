package agents

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// AgentRepository defines operations for managing agents
type AgentRepository interface {
	// Create creates a new agent
	Create(ctx context.Context, agent domain.Agent) error

	// FindByID retrieves an agent by ID
	FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error)

	// FindBySlug retrieves an agent by slug
	FindBySlug(ctx context.Context, slug string) (*domain.Agent, error)

	// List retrieves all agents ordered by sort_order
	List(ctx context.Context) ([]domain.Agent, error)

	// Update updates an agent
	Update(ctx context.Context, agent domain.Agent) error

	// Delete deletes an agent
	// Returns error if agent is still in use by tasks
	Delete(ctx context.Context, id domain.AgentID) error

	// IsInUse checks if an agent is referenced by any tasks
	IsInUse(ctx context.Context, slug string) (bool, error)

	// Clone creates a new agent as a deep copy of an existing agent with a new ID and slug.
	// Returns ErrAgentAlreadyExists if newSlug is already taken.
	Clone(ctx context.Context, sourceID domain.AgentID, newSlug, newName string) (domain.Agent, error)

	// AssignToProject creates a project_agents row linking projectID and agentID.
	// Returns ErrAgentAlreadyInProject if the pair already exists.
	AssignToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error

	// RemoveFromProject deletes the project_agents row.
	// Returns ErrAgentNotInProject if the pair does not exist.
	RemoveFromProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error

	// ListByProject returns all agents assigned to a project, ordered by sort_order then name.
	ListByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)

	// IsAssignedToProject checks whether an agent is assigned to a given project.
	IsAssignedToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) (bool, error)

	// Deprecated: use AssignToProject
	CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error

	// CreateInProject creates an agent in the per-project database
	CreateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error

	// FindBySlugInProject retrieves an agent by slug from the per-project database
	FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error)

	// FindByIDInProject retrieves an agent by ID from the per-project database
	FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) (*domain.Agent, error)

	// ListInProject retrieves all agents from the per-project database
	ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)

	// UpdateInProject updates an agent in the per-project database
	UpdateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error

	// DeleteInProject deletes an agent from the per-project database
	DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) error
}

// RoleRepository is an alias for AgentRepository for backward compatibility
type RoleRepository = AgentRepository
