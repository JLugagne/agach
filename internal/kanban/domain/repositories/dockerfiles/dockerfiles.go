package dockerfiles

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// DockerfileRepository defines operations for managing dockerfiles
type DockerfileRepository interface {
	// Create creates a new dockerfile
	Create(ctx context.Context, dockerfile domain.Dockerfile) error

	// FindByID retrieves a dockerfile by ID
	FindByID(ctx context.Context, id domain.DockerfileID) (*domain.Dockerfile, error)

	// FindBySlug retrieves the latest version of a dockerfile by slug
	FindBySlug(ctx context.Context, slug string) (*domain.Dockerfile, error)

	// FindBySlugAndVersion retrieves a specific version of a dockerfile
	FindBySlugAndVersion(ctx context.Context, slug, version string) (*domain.Dockerfile, error)

	// List retrieves all dockerfiles ordered by sort_order, then name
	List(ctx context.Context) ([]domain.Dockerfile, error)

	// Update updates an existing dockerfile (full replace of mutable fields)
	Update(ctx context.Context, dockerfile domain.Dockerfile) error

	// Delete deletes a dockerfile by ID
	// Returns ErrDockerfileInUse if the dockerfile is still referenced by projects
	Delete(ctx context.Context, id domain.DockerfileID) error

	// IsInUse checks whether any project references this dockerfile
	IsInUse(ctx context.Context, id domain.DockerfileID) (bool, error)

	// SetLatest marks a specific dockerfile version as latest, clearing is_latest on others with same slug prefix
	SetLatest(ctx context.Context, id domain.DockerfileID) error

	// GetProjectDockerfile returns the dockerfile assigned to a project, or nil
	GetProjectDockerfile(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error)

	// SetProjectDockerfile assigns a dockerfile to a project (replaces any existing assignment)
	SetProjectDockerfile(ctx context.Context, projectID domain.ProjectID, dockerfileID domain.DockerfileID) error

	// ClearProjectDockerfile removes any dockerfile assignment from a project
	ClearProjectDockerfile(ctx context.Context, projectID domain.ProjectID) error
}
