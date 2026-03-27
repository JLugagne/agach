package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
)

type ProjectQueries interface {
	GetProject(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error)
	ListProjects(ctx context.Context) ([]domain.Project, error)
	ListProjectsWithSummary(ctx context.Context) ([]domain.ProjectWithSummary, error)
	ListSubProjects(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error)
	ListSubProjectsWithSummary(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error)
	GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error)
	GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error)
}

type TaskQueries interface {
	GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error)
	ListTasks(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error)
	GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, featureID *domain.ProjectID) (*domain.Task, error)
	GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int, featureID *domain.ProjectID) ([]domain.Task, error)
	GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error)
}

type AgentQueries interface {
	GetAgent(ctx context.Context, agentID domain.AgentID) (*domain.Agent, error)
	GetAgentBySlug(ctx context.Context, slug string) (*domain.Agent, error)
	ListAgents(ctx context.Context) ([]domain.Agent, error)
	ListProjectAgents(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)
	GetProjectAgentBySlug(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error)
}

type ColumnQueries interface {
	GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error)
	GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error)
	ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error)
}

type CommentQueries interface {
	GetComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error)
	ListComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error)
	CountComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error)
}

type DependencyQueries interface {
	ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)
	GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
	GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
}

type SkillQueries interface {
	GetSkill(ctx context.Context, skillID domain.SkillID) (*domain.Skill, error)
	GetSkillBySlug(ctx context.Context, slug string) (*domain.Skill, error)
	ListSkills(ctx context.Context) ([]domain.Skill, error)
	ListAgentSkills(ctx context.Context, agentSlug string) ([]domain.Skill, error)
	GetProjectTasksByAgent(ctx context.Context, projectID domain.ProjectID, agentSlug string) ([]domain.Task, error)
}

type FeatureQueries interface {
	GetFeature(ctx context.Context, featureID domain.FeatureID) (*domain.Feature, error)
	ListFeatures(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error)
	GetFeatureStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error)
	ListFeatureTaskSummaries(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error)
}

type DockerfileQueries interface {
	GetDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) (*domain.Dockerfile, error)
	GetDockerfileBySlug(ctx context.Context, slug string) (*domain.Dockerfile, error)
	GetDockerfileBySlugAndVersion(ctx context.Context, slug, version string) (*domain.Dockerfile, error)
	ListDockerfiles(ctx context.Context) ([]domain.Dockerfile, error)
	GetProjectDockerfile(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error)
}

type NotificationQueries interface {
	ListNotifications(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string, unreadOnly bool, limit, offset int) ([]domain.Notification, error)
	GetNotificationUnreadCount(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string) (int, error)
}

type StatsQueries interface {
	GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)
	GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error)
	GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.AgentColdStartStat, error)
	GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error)
	ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error)
}

type SpecializedAgentQueries interface {
	ListSpecializedAgents(ctx context.Context, parentSlug string) ([]domain.SpecializedAgent, error)
	GetSpecializedAgent(ctx context.Context, slug string) (*domain.SpecializedAgent, error)
	ListSpecializedAgentSkills(ctx context.Context, slug string) ([]domain.Skill, error)
	CountSpecializedByParent(ctx context.Context, parentSlug string) (int, error)
}

type ProjectAccessQueries interface {
	ListProjectUserAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectUserAccess, error)
	ListProjectTeamAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectTeamAccess, error)
	HasProjectAccess(ctx context.Context, projectID domain.ProjectID, userID string, teamIDs []string) (bool, error)
	ListAccessibleProjectIDs(ctx context.Context, userID string, teamIDs []string) ([]domain.ProjectID, error)
}

type Queries interface {
	ProjectQueries
	TaskQueries
	AgentQueries
	ColumnQueries
	CommentQueries
	DependencyQueries
	SkillQueries
	FeatureQueries
	DockerfileQueries
	NotificationQueries
	StatsQueries
	SpecializedAgentQueries
	ProjectAccessQueries
}
