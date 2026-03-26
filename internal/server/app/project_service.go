package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	agentsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/sirupsen/logrus"
)

type ProjectService struct {
	projects projects.ProjectRepository
	agents   agentsrepo.AgentRepository
	logger   *logrus.Logger
}

func newProjectService(projects projects.ProjectRepository, agents agentsrepo.AgentRepository, logger *logrus.Logger) *ProjectService {
	return &ProjectService{
		projects: projects,
		agents:   agents,
		logger:   logger,
	}
}

func (s *ProjectService) CreateProject(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
	logger := s.logger.WithContext(ctx)

	if name == "" {
		return domain.Project{}, domain.ErrProjectNameRequired
	}

	if parentID != nil {
		parent, err := s.projects.FindByID(ctx, *parentID)
		if err != nil {
			logger.WithError(err).Error("failed to find parent project")
			return domain.Project{}, errors.Join(domain.ErrProjectNotFound, err)
		}
		if parent == nil {
			return domain.Project{}, domain.ErrProjectNotFound
		}
	}

	project := domain.Project{
		ID:             domain.NewProjectID(),
		ParentID:       parentID,
		Name:           name,
		Description:    description,
		GitURL:         gitURL,
		CreatedByRole:  createdByRole,
		CreatedByAgent: createdByAgent,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.projects.Create(ctx, project); err != nil {
		logger.WithError(err).Error("failed to create project")
		return domain.Project{}, err
	}

	if err := s.agents.CopyGlobalRolesToProject(ctx, project.ID); err != nil {
		logger.WithError(err).Warn("failed to copy global roles to project")
	}

	logger.WithField("projectID", project.ID).Info("project created successfully")
	return project, nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string, gitURL, defaultRole *string) error {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.ErrProjectNotFound
	}

	if name != "" {
		project.Name = name
	}
	if description != "" {
		project.Description = description
	}
	if gitURL != nil {
		project.GitURL = *gitURL
	}
	if defaultRole != nil {
		project.DefaultRole = *defaultRole
	}
	project.UpdatedAt = time.Now()

	if err := s.projects.Update(ctx, *project); err != nil {
		logger.WithError(err).Error("failed to update project")
		return err
	}

	logger.Info("project updated successfully")
	return nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, projectID domain.ProjectID) error {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.ErrProjectNotFound
	}

	deletedIDs, err := s.projects.Delete(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to delete project")
		return err
	}

	logger.WithField("deletedCount", len(deletedIDs)).Info("project deleted successfully")
	return nil
}

func (s *ProjectService) GetProject(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	return project, nil
}

func (s *ProjectService) ListProjects(ctx context.Context) ([]domain.Project, error) {
	logger := s.logger.WithContext(ctx)

	projects, err := s.projects.List(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("failed to list projects")
		return nil, err
	}

	return projects, nil
}

func (s *ProjectService) ListSubProjects(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error) {
	logger := s.logger.WithContext(ctx).WithField("parentID", parentID)

	parent, err := s.projects.FindByID(ctx, parentID)
	if err != nil {
		logger.WithError(err).Error("failed to find parent project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if parent == nil {
		return nil, domain.ErrProjectNotFound
	}

	subProjects, err := s.projects.List(ctx, &parentID)
	if err != nil {
		logger.WithError(err).Error("failed to list sub-projects")
		return nil, err
	}

	return subProjects, nil
}

func (s *ProjectService) GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	summary, err := s.projects.GetSummary(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get project summary")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}

	return summary, nil
}

func (s *ProjectService) ListProjectsWithSummary(ctx context.Context) ([]domain.ProjectWithSummary, error) {
	logger := s.logger.WithContext(ctx)

	projects, err := s.projects.List(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("failed to list projects")
		return nil, err
	}

	result := make([]domain.ProjectWithSummary, 0, len(projects))
	for _, project := range projects {
		pws, err := s.buildProjectWithSummary(ctx, project)
		if err != nil {
			logger.WithError(err).WithField("projectID", project.ID).Warn("failed to build project summary")
			continue
		}
		result = append(result, pws)
	}

	return result, nil
}

func (s *ProjectService) ListSubProjectsWithSummary(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error) {
	logger := s.logger.WithContext(ctx).WithField("parentID", parentID)

	parent, err := s.projects.FindByID(ctx, parentID)
	if err != nil {
		logger.WithError(err).Error("failed to find parent project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if parent == nil {
		return nil, domain.ErrProjectNotFound
	}

	subProjects, err := s.projects.List(ctx, &parentID)
	if err != nil {
		logger.WithError(err).Error("failed to list sub-projects")
		return nil, err
	}

	result := make([]domain.ProjectWithSummary, 0, len(subProjects))
	for _, project := range subProjects {
		pws, err := s.buildProjectWithSummary(ctx, project)
		if err != nil {
			logger.WithError(err).WithField("projectID", project.ID).Warn("failed to build project summary")
			continue
		}
		result = append(result, pws)
	}

	return result, nil
}

func (s *ProjectService) GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	taskSummary, err := s.GetProjectSummary(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get project summary")
		return nil, err
	}

	children, err := s.ListSubProjectsWithSummary(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to list sub-projects")
		return nil, err
	}

	breadcrumb, err := s.buildBreadcrumb(ctx, *project)
	if err != nil {
		logger.WithError(err).Error("failed to build breadcrumb")
		return nil, err
	}

	info := &domain.ProjectInfo{
		Project:     *project,
		TaskSummary: *taskSummary,
		Children:    children,
		Breadcrumb:  breadcrumb,
	}

	return info, nil
}

func (s *ProjectService) buildProjectWithSummary(ctx context.Context, project domain.Project) (domain.ProjectWithSummary, error) {
	taskSummary, err := s.projects.GetSummary(ctx, project.ID)
	if err != nil {
		return domain.ProjectWithSummary{}, err
	}

	childrenCount, err := s.projects.CountChildren(ctx, project.ID)
	if err != nil {
		return domain.ProjectWithSummary{}, err
	}

	return domain.ProjectWithSummary{
		Project:       project,
		ChildrenCount: childrenCount,
		TaskSummary:   *taskSummary,
	}, nil
}

func (s *ProjectService) buildBreadcrumb(ctx context.Context, project domain.Project) ([]domain.Project, error) {
	breadcrumb := []domain.Project{}

	current := project
	for {
		breadcrumb = append([]domain.Project{current}, breadcrumb...)

		if current.ParentID == nil {
			break
		}

		parent, err := s.projects.FindByID(ctx, *current.ParentID)
		if err != nil || parent == nil {
			return nil, errors.Join(domain.ErrProjectNotFound, err)
		}

		current = *parent
	}

	return breadcrumb, nil
}

func (s *ProjectService) resolveRootProjectID(ctx context.Context, projectID domain.ProjectID) (domain.ProjectID, error) {
	current := projectID
	for {
		project, err := s.projects.FindByID(ctx, current)
		if err != nil {
			return "", err
		}
		if project.ParentID == nil {
			return current, nil
		}
		current = *project.ParentID
	}
}
