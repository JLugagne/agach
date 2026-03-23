package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MoveTask Tests

func TestApp_MoveTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColID := domain.NewColumnID()
	inProgressColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: todoColID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == todoColID {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return &domain.Column{ID: inProgressColID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return false, nil
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnInProgress)

	require.NoError(t, err)
	assert.Equal(t, inProgressColID, updatedTask.ColumnID)
}

func TestApp_MoveTask_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnDone)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_MoveTask_FromInProgressToTodo_AppendsResolution(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColID := domain.NewColumnID()
	todoColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:         taskID,
				ColumnID:   inProgressColID,
				Title:      "Task",
				Summary:    "Summary",
				Resolution: "",
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == inProgressColID {
			return &domain.Column{ID: inProgressColID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnTodo {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnTodo)

	require.NoError(t, err)
	assert.Contains(t, updatedTask.Resolution, "Moved back to Todo by human")
	assert.Contains(t, updatedTask.Resolution, "task was not completed")
}

func TestApp_MoveTask_FromInProgressToTodo_PreservesExistingResolution(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColID := domain.NewColumnID()
	todoColID := domain.NewColumnID()

	existingResolution := "Agent stopped work: needs more info"

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:         taskID,
				ColumnID:   inProgressColID,
				Title:      "Task",
				Summary:    "Summary",
				Resolution: existingResolution,
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == inProgressColID {
			return &domain.Column{ID: inProgressColID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnTodo {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnTodo)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(updatedTask.Resolution, existingResolution))
	assert.Contains(t, updatedTask.Resolution, "Moved back to Todo by human")
}

func TestApp_MoveTask_ToBlocked_SetsBlockedFlag(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColID := domain.NewColumnID()
	blockedColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: todoColID, Title: "Task", Summary: "Summary", BlockedReason: "waiting for dependency"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == todoColID {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnBlocked {
			return &domain.Column{ID: blockedColID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnBlocked)

	require.NoError(t, err)
	assert.True(t, updatedTask.IsBlocked)
	assert.NotNil(t, updatedTask.BlockedAt)
}

func TestApp_MoveTask_FromBlocked_ClearsBlockedFlags(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	blockedColID := domain.NewColumnID()
	todoColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:              taskID,
				ColumnID:        blockedColID,
				Title:           "Blocked Task",
				Summary:         "Summary",
				IsBlocked:       true,
				BlockedReason:   "Some reason",
				WontDoRequested: true,
				WontDoReason:    "Won't do reason",
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == blockedColID {
			return &domain.Column{ID: blockedColID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnTodo {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnTodo)

	require.NoError(t, err)
	assert.False(t, updatedTask.IsBlocked)
	assert.Empty(t, updatedTask.BlockedReason)
	assert.False(t, updatedTask.WontDoRequested)
	assert.Empty(t, updatedTask.WontDoReason)
}

// MoveTask - Dependency Checks

func TestApp_MoveTask_ToInProgress_WithUnresolvedDeps_Fails(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColID := domain.NewColumnID()
	inProgressColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: todoColID, Title: "Dependent Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == todoColID {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return &domain.Column{ID: inProgressColID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, errors.New("not found")
	}

	// Signal that there are unresolved dependencies
	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return true, nil
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnInProgress)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrUnresolvedDependencies)
}

func TestApp_MoveTask_ToInProgress_WithResolvedDeps_Succeeds(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColID := domain.NewColumnID()
	inProgressColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: todoColID, Title: "Task With Resolved Deps", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == todoColID {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return &domain.Column{ID: inProgressColID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, errors.New("not found")
	}

	// All dependencies are resolved
	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return false, nil
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnInProgress)

	require.NoError(t, err)
	assert.Equal(t, inProgressColID, updatedTask.ColumnID)
}

func TestApp_MoveTask_ToTodo_WithUnresolvedDeps_Succeeds(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColID := domain.NewColumnID()
	todoColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:       taskID,
				ColumnID: inProgressColID,
				Title:    "Task Moving Back",
				Summary:  "Summary",
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == inProgressColID {
			return &domain.Column{ID: inProgressColID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnTodo {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	// HasUnresolvedDependencies should NOT be called when moving to todo — we verify by not setting it.
	// If it were called with nil func, the mock would panic.

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	// Moving to todo should succeed even without setting HasUnresolvedDependenciesFunc,
	// proving the dependency check is only performed when targeting in_progress.
	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnTodo)

	require.NoError(t, err)
	assert.Equal(t, todoColID, updatedTask.ColumnID)
	assert.Contains(t, updatedTask.Resolution, "Moved back to Todo by human")
}

// StartTask Tests

func TestApp_StartTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColID := domain.NewColumnID()
	inProgressColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: todoColID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == todoColID {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return &domain.Column{ID: inProgressColID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return false, nil
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.StartTask(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.Equal(t, inProgressColID, updatedTask.ColumnID)
}

// CompleteTask Tests

func TestApp_CompleteTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColID := domain.NewColumnID()
	doneColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: inProgressColID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnDone {
			return &domain.Column{ID: doneColID, Slug: domain.ColumnDone, Name: "Done"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.CompleteTask(ctx, projectID, taskID, "Task completed successfully with all requirements met", []string{"main.go", "handler.go"}, "agent1", nil)

	require.NoError(t, err)
	assert.Equal(t, doneColID, updatedTask.ColumnID)
	assert.Equal(t, "Task completed successfully with all requirements met", updatedTask.CompletionSummary)
	assert.Equal(t, []string{"main.go", "handler.go"}, updatedTask.FilesModified)
	assert.Equal(t, "agent1", updatedTask.CompletedByAgent)
	assert.NotNil(t, updatedTask.CompletedAt)
}

func TestApp_CompleteTask_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.CompleteTask(ctx, projectID, taskID, "Completion summary", nil, "agent1", nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_CompleteTask_DoneColumnNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	colID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: colID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return nil, errors.New("not found")
	}

	err := a.CompleteTask(ctx, projectID, taskID, "Completion summary", nil, "agent1", nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

// BlockTask Tests

func TestApp_BlockTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColID := domain.NewColumnID()
	blockedColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: inProgressColID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnBlocked {
			return &domain.Column{ID: blockedColID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	// Comments are auto-created; mock CreateComment dependency chain
	mockComments.CreateFunc = func(ctx context.Context, pid domain.ProjectID, comment domain.Comment) error {
		return nil
	}

	err := a.BlockTask(ctx, projectID, taskID, "Waiting for external API to be ready", "agent1")

	require.NoError(t, err)
	assert.Equal(t, blockedColID, updatedTask.ColumnID)
	assert.True(t, updatedTask.IsBlocked)
	assert.Equal(t, "Waiting for external API to be ready", updatedTask.BlockedReason)
	assert.Equal(t, "agent1", updatedTask.BlockedByAgent)
	assert.NotNil(t, updatedTask.BlockedAt)
}

func TestApp_BlockTask_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.BlockTask(ctx, projectID, taskID, "Some reason", "agent1")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// UnblockTask Tests

func TestApp_UnblockTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	blockedColID := domain.NewColumnID()
	todoColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:            taskID,
				ColumnID:      blockedColID,
				Title:         "Blocked Task",
				Summary:       "Summary",
				IsBlocked:     true,
				BlockedReason: "Waiting for approval",
			}, nil
		}
		return nil, errors.New("not found")
	}

	callCount := 0
	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == blockedColID {
			return &domain.Column{ID: blockedColID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnTodo {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		callCount++
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	err := a.UnblockTask(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.Equal(t, todoColID, updatedTask.ColumnID)
	assert.False(t, updatedTask.IsBlocked)
}

func TestApp_UnblockTask_TaskNotInBlocked_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: todoColID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == todoColID {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	err := a.UnblockTask(ctx, projectID, taskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotInBlocked)
}

func TestApp_UnblockTask_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.UnblockTask(ctx, projectID, taskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// RequestWontDo Tests

func TestApp_RequestWontDo_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColID := domain.NewColumnID()
	blockedColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: todoColID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnBlocked {
			return &domain.Column{ID: blockedColID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	mockComments.CreateFunc = func(ctx context.Context, pid domain.ProjectID, comment domain.Comment) error {
		return nil
	}

	err := a.RequestWontDo(ctx, projectID, taskID, "This task is no longer relevant to the project scope", "agent1")

	require.NoError(t, err)
	assert.Equal(t, blockedColID, updatedTask.ColumnID)
	assert.True(t, updatedTask.IsBlocked)
	assert.True(t, updatedTask.WontDoRequested)
	assert.Equal(t, "This task is no longer relevant to the project scope", updatedTask.WontDoReason)
	assert.Equal(t, "agent1", updatedTask.WontDoRequestedBy)
	assert.NotNil(t, updatedTask.WontDoRequestedAt)
}

func TestApp_RequestWontDo_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.RequestWontDo(ctx, projectID, taskID, "Some reason", "agent1")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// ApproveWontDo Tests

func TestApp_ApproveWontDo_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	blockedColID := domain.NewColumnID()
	doneColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:              taskID,
				ColumnID:        blockedColID,
				Title:           "Task",
				Summary:         "Summary",
				IsBlocked:       true,
				WontDoRequested: true,
				WontDoReason:    "Not needed",
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnDone {
			return &domain.Column{ID: doneColID, Slug: domain.ColumnDone, Name: "Done"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	mockComments.CreateFunc = func(ctx context.Context, pid domain.ProjectID, comment domain.Comment) error {
		return nil
	}

	err := a.ApproveWontDo(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.Equal(t, doneColID, updatedTask.ColumnID)
	assert.True(t, updatedTask.WontDoRequested, "wont_do_requested should remain true as state marker")
	assert.Equal(t, "Not needed", updatedTask.WontDoReason)
	assert.False(t, updatedTask.IsBlocked)
	assert.NotNil(t, updatedTask.CompletedAt)
}

func TestApp_ApproveWontDo_WontDoNotRequested_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	colID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:              taskID,
				ColumnID:        colID,
				Title:           "Task",
				Summary:         "Summary",
				WontDoRequested: false, // No won't-do requested
			}, nil
		}
		return nil, errors.New("not found")
	}

	err := a.ApproveWontDo(ctx, projectID, taskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrWontDoNotRequested)
}

func TestApp_ApproveWontDo_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.ApproveWontDo(ctx, projectID, taskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// RejectWontDo Tests

func TestApp_RejectWontDo_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	blockedColID := domain.NewColumnID()
	todoColID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:              taskID,
				ColumnID:        blockedColID,
				Title:           "Task",
				Summary:         "Summary",
				WontDoRequested: true,
				WontDoReason:    "Some reason",
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnTodo {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	mockComments.CreateFunc = func(ctx context.Context, pid domain.ProjectID, comment domain.Comment) error {
		return nil
	}

	err := a.RejectWontDo(ctx, projectID, taskID, "This task is still needed")

	require.NoError(t, err)
	assert.Equal(t, todoColID, updatedTask.ColumnID)
	assert.False(t, updatedTask.WontDoRequested)
	assert.Empty(t, updatedTask.WontDoReason)
	assert.Empty(t, updatedTask.WontDoRequestedBy)
	assert.Nil(t, updatedTask.WontDoRequestedAt)
	assert.False(t, updatedTask.IsBlocked)
}

func TestApp_RejectWontDo_WontDoNotRequested_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	colID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{
				ID:              taskID,
				ColumnID:        colID,
				Title:           "Task",
				Summary:         "Summary",
				WontDoRequested: false,
			}, nil
		}
		return nil, errors.New("not found")
	}

	err := a.RejectWontDo(ctx, projectID, taskID, "Rejection reason")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrWontDoNotRequested)
}

func TestApp_RejectWontDo_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.RejectWontDo(ctx, projectID, taskID, "Rejection reason")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// Task Query Tests

func TestApp_GetTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()
	expectedTask := &domain.Task{
		ID:       taskID,
		ColumnID: columnID,
		Title:    "Test Task",
		Summary:  "Summary",
		Priority: domain.PriorityHigh,
	}

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return expectedTask, nil
		}
		return nil, errors.New("not found")
	}

	task, err := a.GetTask(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.Equal(t, expectedTask, task)
}

func TestApp_GetTask_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetTask(ctx, projectID, taskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_ListTasks_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	expectedTasks := []domain.TaskWithDetails{
		{Task: domain.Task{ID: domain.NewTaskID(), ColumnID: columnID, Title: "Task 1", Summary: "Summary 1"}},
		{Task: domain.Task{ID: domain.NewTaskID(), ColumnID: columnID, Title: "Task 2", Summary: "Summary 2"}},
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		if pid == projectID {
			return expectedTasks, nil
		}
		return nil, errors.New("not found")
	}

	taskList, err := a.ListTasks(ctx, projectID, tasks.TaskFilters{})

	require.NoError(t, err)
	assert.Equal(t, expectedTasks, taskList)
}

func TestApp_ListTasks_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.ListTasks(ctx, projectID, tasks.TaskFilters{})

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_ListTasks_WithFilters_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()
	todoSlug := domain.ColumnTodo

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	expectedTasks := []domain.TaskWithDetails{
		{Task: domain.Task{ID: domain.NewTaskID(), ColumnID: columnID, Title: "Todo Task", Summary: "Summary"}},
	}

	var capturedFilters tasks.TaskFilters
	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		capturedFilters = filters
		return expectedTasks, nil
	}

	taskList, err := a.ListTasks(ctx, projectID, tasks.TaskFilters{ColumnSlug: &todoSlug})

	require.NoError(t, err)
	assert.Equal(t, expectedTasks, taskList)
	require.NotNil(t, capturedFilters.ColumnSlug)
	assert.Equal(t, domain.ColumnTodo, *capturedFilters.ColumnSlug)
}

func TestApp_GetNextTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, mockDeps := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()
	role := "developer"

	expectedTask := &domain.Task{
		ID:            domain.NewTaskID(),
		ColumnID:      columnID,
		Title:         "Next Task",
		Summary:       "Summary",
		Priority:      domain.PriorityHigh,
		PriorityScore: 300,
		AssignedRole:  role,
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		if pid == projectID {
			return []domain.TaskWithDetails{
				{Task: *expectedTask},
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return false, nil
	}

	_ = mockDeps

	task, err := a.GetNextTask(ctx, projectID, role, nil)

	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, expectedTask.ID, task.ID)
	assert.Equal(t, expectedTask.Title, task.Title)
}

func TestApp_GetNextTask_NoTasksAvailable_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	_, err := a.GetNextTask(ctx, projectID, "developer", nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrNoAvailableTasks)
}

func TestApp_GetNextTask_FiltersCorrectly(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	role := "architect"
	columnID := domain.NewColumnID()

	var capturedFilters tasks.TaskFilters
	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		capturedFilters = filters
		return []domain.TaskWithDetails{
			{Task: domain.Task{ID: domain.NewTaskID(), ColumnID: columnID, Title: "Task", Summary: "Summary", AssignedRole: role, PriorityScore: 200}},
		}, nil
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return false, nil
	}

	_, err := a.GetNextTask(ctx, projectID, role, nil)

	require.NoError(t, err)
	require.NotNil(t, capturedFilters.ColumnSlug)
	assert.Equal(t, domain.ColumnTodo, *capturedFilters.ColumnSlug)
	require.NotNil(t, capturedFilters.AssignedRole)
	assert.Equal(t, role, *capturedFilters.AssignedRole)
	require.NotNil(t, capturedFilters.IsBlocked)
	assert.False(t, *capturedFilters.IsBlocked)
	require.NotNil(t, capturedFilters.WontDoRequested)
	assert.False(t, *capturedFilters.WontDoRequested)
}

func TestApp_GetNextTask_SkipsTasksWithUnresolvedDependencies(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	role := "developer"
	columnID := domain.NewColumnID()

	blockedTaskID := domain.NewTaskID()
	readyTaskID := domain.NewTaskID()

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{
			{Task: domain.Task{ID: blockedTaskID, ColumnID: columnID, Title: "Blocked by dep", Summary: "S", PriorityScore: 400}},
			{Task: domain.Task{ID: readyTaskID, ColumnID: columnID, Title: "Ready", Summary: "S", PriorityScore: 200}},
		}, nil
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		if tid == blockedTaskID {
			return true, nil
		}
		return false, nil
	}

	task, err := a.GetNextTask(ctx, projectID, role, nil)

	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, readyTaskID, task.ID, "should return the task without unresolved dependencies")
}

func TestApp_GetNextTask_AllTasksHaveUnresolvedDeps_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{
			{Task: domain.Task{ID: domain.NewTaskID(), ColumnID: columnID, Title: "Task", Summary: "S", PriorityScore: 400}},
		}, nil
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return true, nil
	}

	_, err := a.GetNextTask(ctx, projectID, "developer", nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNoAvailableTasks)
}

func TestApp_GetNextTask_WithSubProject(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, _, _ := setupTestApp()

	rootProjectID := domain.NewProjectID()
	subProjectID := domain.NewProjectID()
	role := "developer"
	columnID := domain.NewColumnID()
	taskID := domain.NewTaskID()

	mockProjects.GetTreeFunc = func(ctx context.Context, id domain.ProjectID) ([]domain.Project, error) {
		if id == subProjectID {
			return []domain.Project{
				{ID: subProjectID, Name: "Sub Project", ParentID: &rootProjectID},
			}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		if pid == subProjectID {
			return []domain.TaskWithDetails{
				{Task: domain.Task{ID: taskID, ColumnID: columnID, Title: "Sub task", Summary: "S", PriorityScore: 300}},
			}, nil
		}
		return []domain.TaskWithDetails{}, nil
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (bool, error) {
		return false, nil
	}

	task, err := a.GetNextTask(ctx, rootProjectID, role, &subProjectID)

	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, taskID, task.ID)
}
