package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/comments"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/dependencies"
	dockerfilesrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/dockerfiles"
	featuresrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/features"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects"
	agentsrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/agents"
	skillsrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/skills"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/toolusage"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/sirupsen/logrus"
)

// App implements both Commands and Queries service interfaces
type App struct {
	projects     projects.ProjectRepository
	agents       agentsrepo.AgentRepository
	features     featuresrepo.FeatureRepository
	tasks        tasks.TaskRepository
	columns      columns.ColumnRepository
	comments     comments.CommentRepository
	dependencies dependencies.DependencyRepository
	toolUsage    toolusage.ToolUsageRepository
	skills       skillsrepo.SkillRepository
	dockerfiles  dockerfilesrepo.DockerfileRepository
	logger       *logrus.Logger
}

// Config holds the dependencies for the App
type Config struct {
	Projects     projects.ProjectRepository
	Agents       agentsrepo.AgentRepository
	Features     featuresrepo.FeatureRepository
	Tasks        tasks.TaskRepository
	Columns      columns.ColumnRepository
	Comments     comments.CommentRepository
	Dependencies dependencies.DependencyRepository
	ToolUsage    toolusage.ToolUsageRepository
	Skills       skillsrepo.SkillRepository
	Dockerfiles  dockerfilesrepo.DockerfileRepository
	Logger       *logrus.Logger
}

// NewApp creates a new App instance
func NewApp(cfg Config) *App {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	return &App{
		projects:     cfg.Projects,
		agents:       cfg.Agents,
		features:     cfg.Features,
		tasks:        cfg.Tasks,
		columns:      cfg.Columns,
		comments:     cfg.Comments,
		dependencies: cfg.Dependencies,
		toolUsage:    cfg.ToolUsage,
		skills:       cfg.Skills,
		dockerfiles:  cfg.Dockerfiles,
		logger:       cfg.Logger,
	}
}

// Verify that App implements both service interfaces
var (
	_ service.Commands = (*App)(nil)
	_ service.Queries  = (*App)(nil)
)

// Project Commands

func (a *App) CreateProject(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
	logger := a.logger.WithContext(ctx)

	if name == "" {
		return domain.Project{}, domain.ErrProjectNameRequired
	}

	// If parentID is provided, verify parent exists
	if parentID != nil {
		parent, err := a.projects.FindByID(ctx, *parentID)
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

	if err := a.projects.Create(ctx, project); err != nil {
		logger.WithError(err).Error("failed to create project")
		return domain.Project{}, err
	}

	if err := a.agents.CopyGlobalRolesToProject(ctx, project.ID); err != nil {
		logger.WithError(err).Warn("failed to copy global roles to project")
	}

	logger.WithField("projectID", project.ID).Info("project created successfully")
	return project, nil
}

func (a *App) UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string, gitURL, defaultRole *string) error {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := a.projects.FindByID(ctx, projectID)
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

	if err := a.projects.Update(ctx, *project); err != nil {
		logger.WithError(err).Error("failed to update project")
		return err
	}

	logger.Info("project updated successfully")
	return nil
}

func (a *App) DeleteProject(ctx context.Context, projectID domain.ProjectID) error {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	// Verify project exists
	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.ErrProjectNotFound
	}

	deletedIDs, err := a.projects.Delete(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to delete project")
		return err
	}

	logger.WithField("deletedCount", len(deletedIDs)).Info("project deleted successfully")
	return nil
}

// Project Queries

func (a *App) GetProject(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	return project, nil
}

func (a *App) ListProjects(ctx context.Context) ([]domain.Project, error) {
	logger := a.logger.WithContext(ctx)

	projects, err := a.projects.List(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("failed to list projects")
		return nil, err
	}

	return projects, nil
}

func (a *App) ListSubProjects(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error) {
	logger := a.logger.WithContext(ctx).WithField("parentID", parentID)

	// Verify parent exists
	parent, err := a.projects.FindByID(ctx, parentID)
	if err != nil {
		logger.WithError(err).Error("failed to find parent project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if parent == nil {
		return nil, domain.ErrProjectNotFound
	}

	subProjects, err := a.projects.List(ctx, &parentID)
	if err != nil {
		logger.WithError(err).Error("failed to list sub-projects")
		return nil, err
	}

	return subProjects, nil
}

func (a *App) GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	summary, err := a.projects.GetSummary(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get project summary")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}

	return summary, nil
}

func (a *App) ListProjectsWithSummary(ctx context.Context) ([]domain.ProjectWithSummary, error) {
	logger := a.logger.WithContext(ctx)

	projects, err := a.projects.List(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("failed to list projects")
		return nil, err
	}

	result := make([]domain.ProjectWithSummary, 0, len(projects))
	for _, project := range projects {
		pws, err := a.buildProjectWithSummary(ctx, project)
		if err != nil {
			logger.WithError(err).WithField("projectID", project.ID).Warn("failed to build project summary")
			continue
		}
		result = append(result, pws)
	}

	return result, nil
}

// Feature Commands

func (a *App) CreateFeature(ctx context.Context, projectID domain.ProjectID, name, description, createdByRole, createdByAgent string) (domain.Feature, error) {
	if name == "" {
		return domain.Feature{}, domain.ErrFeatureNameRequired
	}

	// Verify project exists
	if _, err := a.projects.FindByID(ctx, projectID); err != nil {
		return domain.Feature{}, errors.Join(domain.ErrProjectNotFound, err)
	}

	feature := domain.Feature{
		ID:             domain.NewFeatureID(),
		ProjectID:      projectID,
		Name:           name,
		Description:    description,
		Status:         domain.FeatureStatusDraft,
		CreatedByRole:  createdByRole,
		CreatedByAgent: createdByAgent,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := a.features.Create(ctx, feature); err != nil {
		return domain.Feature{}, err
	}

	return feature, nil
}

func (a *App) UpdateFeature(ctx context.Context, featureID domain.FeatureID, name, description string) error {
	feature, err := a.features.FindByID(ctx, featureID)
	if err != nil {
		return errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return domain.ErrFeatureNotFound
	}

	feature.Name = name
	feature.Description = description
	feature.UpdatedAt = time.Now()

	return a.features.Update(ctx, *feature)
}

func (a *App) UpdateFeatureStatus(ctx context.Context, featureID domain.FeatureID, status domain.FeatureStatus) error {
	if !domain.ValidFeatureStatuses[status] {
		return domain.ErrInvalidFeatureStatus
	}

	feature, err := a.features.FindByID(ctx, featureID)
	if err != nil {
		return errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return domain.ErrFeatureNotFound
	}

	return a.features.UpdateStatus(ctx, featureID, status)
}

func (a *App) DeleteFeature(ctx context.Context, featureID domain.FeatureID) error {
	feature, err := a.features.FindByID(ctx, featureID)
	if err != nil {
		return errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return domain.ErrFeatureNotFound
	}

	return a.features.Delete(ctx, featureID)
}

// Feature Queries

func (a *App) GetFeature(ctx context.Context, featureID domain.FeatureID) (*domain.Feature, error) {
	feature, err := a.features.FindByID(ctx, featureID)
	if err != nil {
		return nil, errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return nil, domain.ErrFeatureNotFound
	}
	return feature, nil
}

func (a *App) ListFeatures(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
	return a.features.List(ctx, projectID, statusFilter)
}

func (a *App) ListSubProjectsWithSummary(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error) {
	logger := a.logger.WithContext(ctx).WithField("parentID", parentID)

	// Verify parent exists
	parent, err := a.projects.FindByID(ctx, parentID)
	if err != nil {
		logger.WithError(err).Error("failed to find parent project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if parent == nil {
		return nil, domain.ErrProjectNotFound
	}

	subProjects, err := a.projects.List(ctx, &parentID)
	if err != nil {
		logger.WithError(err).Error("failed to list sub-projects")
		return nil, err
	}

	result := make([]domain.ProjectWithSummary, 0, len(subProjects))
	for _, project := range subProjects {
		pws, err := a.buildProjectWithSummary(ctx, project)
		if err != nil {
			logger.WithError(err).WithField("projectID", project.ID).Warn("failed to build project summary")
			continue
		}
		result = append(result, pws)
	}

	return result, nil
}

func (a *App) GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	// Get project
	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	// Get task summary
	taskSummary, err := a.GetProjectSummary(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get project summary")
		return nil, err
	}

	// Get direct children with summaries
	children, err := a.ListSubProjectsWithSummary(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to list sub-projects")
		return nil, err
	}

	// Build breadcrumb
	breadcrumb, err := a.buildBreadcrumb(ctx, *project)
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

// Helper functions

func (a *App) buildProjectWithSummary(ctx context.Context, project domain.Project) (domain.ProjectWithSummary, error) {
	// Get task summary using efficient repository JOIN query
	taskSummary, err := a.projects.GetSummary(ctx, project.ID)
	if err != nil {
		return domain.ProjectWithSummary{}, err
	}

	// Count children without fetching full records
	childrenCount, err := a.projects.CountChildren(ctx, project.ID)
	if err != nil {
		return domain.ProjectWithSummary{}, err
	}

	return domain.ProjectWithSummary{
		Project:       project,
		ChildrenCount: childrenCount,
		TaskSummary:   *taskSummary,
	}, nil
}

func (a *App) buildBreadcrumb(ctx context.Context, project domain.Project) ([]domain.Project, error) {
	breadcrumb := []domain.Project{}

	// Build path from root to current project
	current := project
	for {
		// Prepend current project to breadcrumb
		breadcrumb = append([]domain.Project{current}, breadcrumb...)

		// If no parent, we're at the root
		if current.ParentID == nil {
			break
		}

		// Get parent
		parent, err := a.projects.FindByID(ctx, *current.ParentID)
		if err != nil || parent == nil {
			return nil, errors.Join(domain.ErrProjectNotFound, err)
		}

		current = *parent
	}

	return breadcrumb, nil
}
