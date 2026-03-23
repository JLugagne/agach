package projects

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// ProjectRepository defines operations for managing projects
type ProjectRepository interface {
	// Create creates a new project and its SQLite database
	Create(ctx context.Context, project domain.Project) error

	// FindByID retrieves a project by ID
	FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error)

	// List retrieves all root projects (parent_id IS NULL) or children of a parent
	List(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error)

	// GetTree retrieves a project and all its sub-projects recursively
	GetTree(ctx context.Context, id domain.ProjectID) ([]domain.Project, error)

	// Update updates a project's name or description
	Update(ctx context.Context, project domain.Project) error

	// Delete deletes a project and all sub-projects in cascade
	// Returns the list of deleted project IDs (including descendants)
	Delete(ctx context.Context, id domain.ProjectID) ([]domain.ProjectID, error)

	// GetSummary returns task counts per column for a project
	GetSummary(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error)

	// CountChildren returns the number of direct children
	CountChildren(ctx context.Context, id domain.ProjectID) (int, error)

	// ListModelPricing returns all model pricing records.
	ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error)
}
