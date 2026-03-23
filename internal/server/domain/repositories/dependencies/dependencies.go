package dependencies

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// DependencyRepository defines operations for managing task dependencies within a project
type DependencyRepository interface {
	// Create creates a new dependency in the specified project's DB
	Create(ctx context.Context, projectID domain.ProjectID, dep domain.TaskDependency) error

	// Delete deletes a dependency
	Delete(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) error

	// List retrieves all dependencies for a task
	List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)

	// WouldCreateCycle checks if adding a dependency would create a cycle
	WouldCreateCycle(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) (bool, error)

	// ListDependents retrieves all task IDs that depend on the given task
	// (i.e. tasks where depends_on_task_id = taskID)
	ListDependents(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)

	// GetDependencyContext returns context for all resolved dependencies of a task
	// (dependencies in "done" or "wont_do" columns)
	GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error)
}
