package tasks

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
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

	// GetNextTasks returns up to count tasks for a given role with the same filters as GetNextTask
	GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int) ([]domain.Task, error)

	// HasUnresolvedDependencies checks if a task has dependencies not in "done" or "wont_do"
	HasUnresolvedDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (bool, error)

	// GetDependentsNotDone gets tasks that depend on this task and are not in "done" or "wont_do"
	GetDependentsNotDone(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)

	// MarkTaskSeen sets seen_at = NOW() for the given task (idempotent — only sets if currently NULL)
	MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error

	// ReorderTask changes the position of a task within its current column,
	// shifting other tasks in the same column to make room.
	ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error

	// GetTimeline returns daily task creation and completion counts for the last N days.
	GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error)

	// UpdateSessionID sets the session_id field for a task (for Claude Code session resumption).
	UpdateSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error

	// GetColdStartStats returns aggregated cold-start token statistics grouped by assigned role.
	GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.AgentColdStartStat, error)

	// BulkCreate creates multiple tasks atomically within a single transaction.
	// If any insert fails, no tasks are created.
	BulkCreate(ctx context.Context, projectID domain.ProjectID, tasks []domain.Task) error

	// BulkReassignInProject updates assigned_role from oldSlug to newSlug
	// for all tasks in a project. If newSlug is empty string, clears assigned_role.
	// Returns the number of tasks updated.
	BulkReassignInProject(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error)

	// ListByAssignedRole returns all tasks in a project that have assigned_role == slug.
	// Used to check whether an agent can be safely removed.
	ListByAssignedRole(ctx context.Context, projectID domain.ProjectID, slug string) ([]domain.Task, error)

	// GetModelTokenStats returns aggregated token usage grouped by model for a project.
	GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error)
}

// TaskFilters defines optional filters for listing tasks
type TaskFilters struct {
	ColumnSlug      *domain.ColumnSlug
	FeatureID       *domain.FeatureID
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
