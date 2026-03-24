package builds

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
)

// DockerBuildRepository defines operations for managing docker build records.
type DockerBuildRepository interface {
	Create(ctx context.Context, build domain.DockerBuild) error
	FindByID(ctx context.Context, id domain.BuildID) (*domain.DockerBuild, error)
	ListByDockerfile(ctx context.Context, slug string) ([]domain.DockerBuild, error)
	ListAll(ctx context.Context) ([]domain.DockerBuild, error)
	UpdateStatus(ctx context.Context, id domain.BuildID, status domain.BuildStatus, log string) error
	Delete(ctx context.Context, id domain.BuildID) error
	DeleteNonLatest(ctx context.Context, slug string) (int, error)
}
