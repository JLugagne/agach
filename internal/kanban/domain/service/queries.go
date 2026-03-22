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
	ListFeaturesActiveOnly(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error)
	GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error)
	GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error)
	ListProjectsByWorkDir(ctx context.Context, workDir string) ([]domain.ProjectWithSummary, error)

	// Role queries (global)
	GetRole(ctx context.Context, roleID domain.RoleID) (*domain.Role, error)
	GetRoleBySlug(ctx context.Context, slug string) (*domain.Role, error)
	ListRoles(ctx context.Context) ([]domain.Role, error)

	// Role queries (per-project)
	ListProjectRoles(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error)
	GetProjectRoleBySlug(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error)

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

	// GetDependencyTasks returns the task objects that this task depends on
	GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)

	// GetDependentTasks returns the task objects that depend on this task
	GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)

	// Tool usage queries
	GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)

	// Timeline queries
	GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error)

	// Cold start stats queries
	GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.RoleColdStartStat, error)

	// GetWIPSlots returns the current WIP slot availability for the in_progress column.
	// FreeSlots is -1 when the column has no limit (wip_limit == 0).
	GetWIPSlots(ctx context.Context, projectID domain.ProjectID) (*domain.WIPSlotsInfo, error)

	// Skill queries

	// GetSkill retrieves a skill by ID.
	GetSkill(ctx context.Context, skillID domain.SkillID) (*domain.Skill, error)

	// GetSkillBySlug retrieves a skill by slug.
	GetSkillBySlug(ctx context.Context, slug string) (*domain.Skill, error)

	// ListSkills returns all global skills ordered by sort_order.
	ListSkills(ctx context.Context) ([]domain.Skill, error)

	// ListAgentSkills returns all skills assigned to a given agent (by slug).
	ListAgentSkills(ctx context.Context, agentSlug string) ([]domain.Skill, error)

	// GetProjectTasksByAgent returns all tasks in a project whose assigned_role matches agentSlug.
	// Used by the remove-agent dialog to show how many tasks are affected.
	GetProjectTasksByAgent(ctx context.Context, projectID domain.ProjectID, agentSlug string) ([]domain.Task, error)

	// Dockerfile queries

	// GetDockerfile retrieves a dockerfile by ID.
	GetDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) (*domain.Dockerfile, error)

	// GetDockerfileBySlugAndVersion retrieves a specific version of a dockerfile.
	GetDockerfileBySlugAndVersion(ctx context.Context, slug, version string) (*domain.Dockerfile, error)

	// ListDockerfiles returns all dockerfiles ordered by slug, sort_order, version.
	ListDockerfiles(ctx context.Context) ([]domain.Dockerfile, error)

	// GetProjectDockerfile returns the dockerfile currently assigned to a project, or nil.
	GetProjectDockerfile(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error)
}
