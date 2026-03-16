package tasks

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// TaskRepository defines operations for managing tasks within a project
type TaskRepository interface {
	// Create creates a new task in the specified project's DB
	Create(ctx context.Context, projectID domain.ProjectID, task domain.Task) error

	// FindByID retrieves a task by ID from the specified project's DB
	FindByID(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) (*domain.Task, error)

	// List retrieves tasks with optional filters
	List(ctx context.Context, projectID domain.ProjectID, filters TaskFilters) ([]domain.TaskWithDetails, error)

	// Update updates a task
	Update(ctx context.Context, projectID domain.ProjectID, task domain.Task) error

	// Delete deletes a task
	Delete(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) error

	// Move moves a task to a different column
	Move(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnID domain.ColumnID) error

	// CountByColumn returns the number of tasks in a column
	CountByColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (int, error)

	// GetNextTask returns the next task for a given role
	// Filters: column=todo, assigned_role=role, is_blocked=0, wont_do_requested=0, all deps resolved
	// Sorts by priority_score DESC, created_at ASC
	GetNextTask(ctx context.Context, projectID domain.ProjectID, role string) (*domain.Task, error)

	// HasUnresolvedDependencies checks if a task has dependencies not in "done" or "wont_do"
	HasUnresolvedDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (bool, error)

	// GetDependentsNotDone gets tasks that depend on this task and are not in "done" or "wont_do"
	GetDependentsNotDone(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)

	// MarkTaskSeen sets seen_at = NOW() for the given task (idempotent — only sets if currently NULL)
	MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
}

// TaskFilters defines optional filters for listing tasks
type TaskFilters struct {
	ColumnSlug      *domain.ColumnSlug
	AssignedRole    *string
	Tag             *string
	Priority        *domain.Priority
	IsBlocked       *bool
	WontDoRequested *bool
	UpdatedSince    *time.Time
	Search          string // Full-text search query (matches title, summary, description, tags)
	Limit           int    // 0 means no limit
	Offset          int
}
