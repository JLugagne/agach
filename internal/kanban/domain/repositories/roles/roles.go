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
}
