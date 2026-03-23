package columns

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// ColumnRepository defines operations for managing columns within a project
type ColumnRepository interface {
	// FindByID retrieves a column by ID from the specified project's DB
	FindByID(ctx context.Context, projectID domain.ProjectID, id domain.ColumnID) (*domain.Column, error)

	// FindBySlug retrieves a column by slug from the specified project's DB
	FindBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error)

	// List retrieves all columns for a project ordered by position
	List(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error)

	// EnsureBacklog creates the backlog column if it does not exist and returns it.
	EnsureBacklog(ctx context.Context, projectID domain.ProjectID) (*domain.Column, error)
}
