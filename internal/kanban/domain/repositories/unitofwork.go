package repositories

import (
	"context"

	agentsrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/agents"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/comments"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/dependencies"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
)

// Repositories groups all repository interfaces for use within a transaction
type Repositories struct {
	Projects     projects.ProjectRepository
	Agents       agentsrepo.AgentRepository
	Tasks        tasks.TaskRepository
	Columns      columns.ColumnRepository
	Comments     comments.CommentRepository
	Dependencies dependencies.DependencyRepository
}

// UnitOfWork provides transaction management across repositories
type UnitOfWork interface {
	// Do executes a function within a transaction
	// If fn returns an error, the transaction is rolled back
	// Otherwise, the transaction is committed
	Do(ctx context.Context, fn func(repos Repositories) error) error
}
