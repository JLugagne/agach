package roles

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// RoleRepository defines operations for managing roles
type RoleRepository interface {
	// Create creates a new role
	Create(ctx context.Context, role domain.Role) error

	// FindByID retrieves a role by ID
	FindByID(ctx context.Context, id domain.RoleID) (*domain.Role, error)

	// FindBySlug retrieves a role by slug
	FindBySlug(ctx context.Context, slug string) (*domain.Role, error)

	// List retrieves all roles ordered by sort_order
	List(ctx context.Context) ([]domain.Role, error)

	// Update updates a role
	Update(ctx context.Context, role domain.Role) error

	// Delete deletes a role
	// Returns error if role is still in use by tasks
	Delete(ctx context.Context, id domain.RoleID) error

	// IsInUse checks if a role is referenced by any tasks
	IsInUse(ctx context.Context, slug string) (bool, error)

	// CopyGlobalRolesToProject copies all global roles into the per-project database
	CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error

	// CreateInProject creates a role in the per-project database
	CreateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error

	// FindBySlugInProject retrieves a role by slug from the per-project database
	FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error)

	// FindByIDInProject retrieves a role by ID from the per-project database
	FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) (*domain.Role, error)

	// ListInProject retrieves all roles from the per-project database
	ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error)

	// UpdateInProject updates a role in the per-project database
	UpdateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error

	// DeleteInProject deletes a role from the per-project database
	DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) error
}
