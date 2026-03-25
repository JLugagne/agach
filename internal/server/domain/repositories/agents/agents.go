package agents

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// GlobalAgentRepository handles global agent CRUD operations.
type GlobalAgentRepository interface {
	Create(ctx context.Context, agent domain.Agent) error
	FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Agent, error)
	List(ctx context.Context) ([]domain.Agent, error)
	Update(ctx context.Context, agent domain.Agent) error
	Delete(ctx context.Context, id domain.AgentID) error
	IsInUse(ctx context.Context, slug string) (bool, error)
	Clone(ctx context.Context, sourceID domain.AgentID, newSlug, newName string) (domain.Agent, error)
}

// ProjectAgentRepository handles project assignment operations.
type ProjectAgentRepository interface {
	AssignToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error
	RemoveFromProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error
	ListByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)
	IsAssignedToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) (bool, error)
	// Deprecated: use AssignToProject
	CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error
}

// ScopedAgentRepository handles per-project agent CRUD operations.
type ScopedAgentRepository interface {
	CreateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error
	FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error)
	FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) (*domain.Agent, error)
	ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)
	UpdateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error
	DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) error
}

// AgentRepository composes all sub-interfaces.
type AgentRepository interface {
	GlobalAgentRepository
	ProjectAgentRepository
	ScopedAgentRepository
}
