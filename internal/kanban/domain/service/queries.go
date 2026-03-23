package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
)

// Queries defines read operations for the Kanban system
type Queries interface {
	// Project queries
	GetProject(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error)
	ListProjects(ctx context.Context) ([]domain.Project, error)
	ListProjectsWithSummary(ctx context.Context) ([]domain.ProjectWithSummary, error)
	ListSubProjects(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error)
	ListSubProjectsWithSummary(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error)
	GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error)
	GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error)
	// Agent queries (global)
	GetAgent(ctx context.Context, agentID domain.AgentID) (*domain.Agent, error)
	GetAgentBySlug(ctx context.Context, slug string) (*domain.Agent, error)
	ListAgents(ctx context.Context) ([]domain.Agent, error)

	// Agent queries (per-project)
	ListProjectAgents(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)
	GetProjectAgentBySlug(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error)

	// Task queries
	GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error)
	ListTasks(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error)
	GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, featureID *domain.ProjectID) (*domain.Task, error)
	GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int, featureID *domain.ProjectID) ([]domain.Task, error)
	GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error)

	// Column queries
	GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error)
	GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error)
	ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error)

	// Comment queries
	GetComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error)
	ListComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error)
	CountComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error)

	// Dependency queries
	ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)
	GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
	GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)

	// Tool usage queries
	GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)

	// Timeline queries
	GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error)

	// Cold start stats queries
	GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.AgentColdStartStat, error)

	// Model token stats queries
	GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error)

	// Model pricing queries
	ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error)

	// Feature queries
	GetFeature(ctx context.Context, featureID domain.FeatureID) (*domain.Feature, error)
	ListFeatures(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error)
	GetFeatureStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error)

	// Skill queries
	GetSkill(ctx context.Context, skillID domain.SkillID) (*domain.Skill, error)
	GetSkillBySlug(ctx context.Context, slug string) (*domain.Skill, error)
	ListSkills(ctx context.Context) ([]domain.Skill, error)
	ListAgentSkills(ctx context.Context, agentSlug string) ([]domain.Skill, error)
	GetProjectTasksByAgent(ctx context.Context, projectID domain.ProjectID, agentSlug string) ([]domain.Task, error)

	// Dockerfile queries
	GetDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) (*domain.Dockerfile, error)
	GetDockerfileBySlugAndVersion(ctx context.Context, slug, version string) (*domain.Dockerfile, error)
	ListDockerfiles(ctx context.Context) ([]domain.Dockerfile, error)
	GetProjectDockerfile(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error)

	// Notification queries
	ListNotifications(ctx context.Context, projectID domain.ProjectID, unreadOnly bool, limit, offset int) ([]domain.Notification, error)
	GetNotificationUnreadCount(ctx context.Context, projectID domain.ProjectID) (int, error)
}
