package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	dockerfilesrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dockerfiles"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/sirupsen/logrus"
)

type DockerfileService struct {
	dockerfiles dockerfilesrepo.DockerfileRepository
	projects    projects.ProjectRepository
	logger      *logrus.Logger
}

func newDockerfileService(dockerfiles dockerfilesrepo.DockerfileRepository, projects projects.ProjectRepository, logger *logrus.Logger) *DockerfileService {
	return &DockerfileService{
		dockerfiles: dockerfiles,
		projects:    projects,
		logger:      logger,
	}
}

func (s *DockerfileService) CreateDockerfile(ctx context.Context, slug, name, description, version, content string, isLatest bool, sortOrder int) (domain.Dockerfile, error) {
	logger := s.logger.WithContext(ctx)

	if slug == "" {
		return domain.Dockerfile{}, domain.ErrDockerfileSlugRequired
	}
	if name == "" {
		return domain.Dockerfile{}, domain.ErrDockerfileNameRequired
	}
	if version == "" {
		return domain.Dockerfile{}, domain.ErrDockerfileVersionRequired
	}

	existing, err := s.dockerfiles.FindBySlugAndVersion(ctx, slug, version)
	if err == nil && existing != nil {
		return domain.Dockerfile{}, domain.ErrDockerfileAlreadyExists
	}

	dockerfile := domain.Dockerfile{
		ID:          domain.NewDockerfileID(),
		Slug:        slug,
		Name:        name,
		Description: description,
		Version:     version,
		Content:     content,
		IsLatest:    isLatest,
		SortOrder:   sortOrder,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.dockerfiles.Create(ctx, dockerfile); err != nil {
		logger.WithError(err).Error("failed to create dockerfile")
		return domain.Dockerfile{}, err
	}

	if isLatest {
		if err := s.dockerfiles.SetLatest(ctx, dockerfile.ID); err != nil {
			logger.WithError(err).Warn("failed to set dockerfile as latest")
		}
	}

	logger.WithField("dockerfileID", dockerfile.ID).Info("dockerfile created")
	return dockerfile, nil
}

func (s *DockerfileService) UpdateDockerfile(ctx context.Context, dockerfileID domain.DockerfileID, name, description, content *string, isLatest *bool, sortOrder *int) error {
	logger := s.logger.WithContext(ctx).WithField("dockerfileID", dockerfileID)

	dockerfile, err := s.dockerfiles.FindByID(ctx, dockerfileID)
	if err != nil {
		return domain.ErrDockerfileNotFound
	}

	if name != nil {
		dockerfile.Name = *name
	}
	if description != nil {
		dockerfile.Description = *description
	}
	if content != nil {
		dockerfile.Content = *content
	}
	if isLatest != nil {
		dockerfile.IsLatest = *isLatest
	}
	if sortOrder != nil {
		dockerfile.SortOrder = *sortOrder
	}
	dockerfile.UpdatedAt = time.Now()

	if err := s.dockerfiles.Update(ctx, *dockerfile); err != nil {
		logger.WithError(err).Error("failed to update dockerfile")
		return err
	}

	if isLatest != nil && *isLatest {
		if err := s.dockerfiles.SetLatest(ctx, dockerfileID); err != nil {
			logger.WithError(err).Warn("failed to set dockerfile as latest")
		}
	}

	logger.Info("dockerfile updated")
	return nil
}

func (s *DockerfileService) DeleteDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) error {
	logger := s.logger.WithContext(ctx).WithField("dockerfileID", dockerfileID)

	if _, err := s.dockerfiles.FindByID(ctx, dockerfileID); err != nil {
		return domain.ErrDockerfileNotFound
	}

	inUse, err := s.dockerfiles.IsInUse(ctx, dockerfileID)
	if err != nil {
		return err
	}
	if inUse {
		return domain.ErrDockerfileInUse
	}

	if err := s.dockerfiles.Delete(ctx, dockerfileID); err != nil {
		logger.WithError(err).Error("failed to delete dockerfile")
		return err
	}

	logger.Info("dockerfile deleted")
	return nil
}

func (s *DockerfileService) SetProjectDockerfile(ctx context.Context, projectID domain.ProjectID, dockerfileID domain.DockerfileID) error {
	logger := s.logger.WithContext(ctx)

	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		return domain.ErrProjectNotFound
	}
	if _, err := s.dockerfiles.FindByID(ctx, dockerfileID); err != nil {
		return domain.ErrDockerfileNotFound
	}

	if err := s.dockerfiles.SetProjectDockerfile(ctx, projectID, dockerfileID); err != nil {
		logger.WithError(err).Error("failed to set project dockerfile")
		return err
	}

	logger.WithField("projectID", projectID).WithField("dockerfileID", dockerfileID).Info("project dockerfile set")
	return nil
}

func (s *DockerfileService) ClearProjectDockerfile(ctx context.Context, projectID domain.ProjectID) error {
	logger := s.logger.WithContext(ctx)

	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		return domain.ErrProjectNotFound
	}

	if err := s.dockerfiles.ClearProjectDockerfile(ctx, projectID); err != nil {
		logger.WithError(err).Error("failed to clear project dockerfile")
		return err
	}

	logger.WithField("projectID", projectID).Info("project dockerfile cleared")
	return nil
}

func (s *DockerfileService) GetDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) (*domain.Dockerfile, error) {
	d, err := s.dockerfiles.FindByID(ctx, dockerfileID)
	if err != nil {
		return nil, domain.ErrDockerfileNotFound
	}
	return d, nil
}

func (s *DockerfileService) GetDockerfileBySlug(ctx context.Context, slug string) (*domain.Dockerfile, error) {
	d, err := s.dockerfiles.FindBySlug(ctx, slug)
	if err != nil {
		return nil, domain.ErrDockerfileNotFound
	}
	return d, nil
}

func (s *DockerfileService) GetDockerfileBySlugAndVersion(ctx context.Context, slug, version string) (*domain.Dockerfile, error) {
	d, err := s.dockerfiles.FindBySlugAndVersion(ctx, slug, version)
	if err != nil {
		return nil, domain.ErrDockerfileNotFound
	}
	return d, nil
}

func (s *DockerfileService) ListDockerfiles(ctx context.Context) ([]domain.Dockerfile, error) {
	return s.dockerfiles.List(ctx)
}

func (s *DockerfileService) GetProjectDockerfile(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error) {
	return s.dockerfiles.GetProjectDockerfile(ctx, projectID)
}
