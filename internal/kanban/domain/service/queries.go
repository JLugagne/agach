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
	ListProjectsByWorkDir(ctx context.Context, workDir string) ([]domain.ProjectWithSummary, error)

	// Role queries
	GetRole(ctx context.Context, roleID domain.RoleID) (*domain.Role, error)
	GetRoleBySlug(ctx context.Context, slug string) (*domain.Role, error)
	ListRoles(ctx context.Context) ([]domain.Role, error)

	// Task queries
	GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error)
	ListTasks(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error)
	GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error)
	GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error)

	// Column queries
	GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error)
	GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error)
	ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error)

	// Comment queries
	GetComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error)
	ListComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error)

	// Dependency queries
	ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)

	// GetDependencyTasks returns the task objects that this task depends on
	GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)

	// GetDependentTasks returns the task objects that depend on this task
	GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)

	// Tool usage queries
	GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)
}
