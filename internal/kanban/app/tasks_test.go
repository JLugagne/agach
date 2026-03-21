package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Task Command Tests

func TestApp_CreateTask_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	todoColID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnTodo {
			return &domain.Column{ID: todoColID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.CreateFunc = func(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
		return nil
	}

	mockTasks.ListFunc = func(ctx context.Context, pid domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{}, nil
	}

	task, err := a.CreateTask(ctx, projectID, "Test Task", "Test summary", "Task description", domain.PriorityMedium, "architect", "agent1", "", nil, nil, "", false, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, "Test summary", task.Summary)
	assert.Equal(t, "Task description", task.Description)
	assert.Equal(t, domain.PriorityMedium, task.Priority)
	assert.Equal(t, 200, task.PriorityScore)
	assert.Equal(t, todoColID, task.ColumnID)
}

func TestApp_CreateTask_EmptyTitle_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	_, err := a.CreateTask(ctx, projectID, "", "Summary", "Description", domain.PriorityMedium, "architect", "agent1", "", nil, nil, "", false, nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskTitleRequired)
}

func TestApp_CreateTask_EmptySummary_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	_, err := a.CreateTask(ctx, projectID, "Test Task", "", "Description", domain.PriorityMedium, "architect", "agent1", "", nil, nil, "", false, nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrSummaryRequired)
}

func TestApp_CreateTask_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.CreateTask(ctx, projectID, "Test Task", "Summary", "Description", domain.PriorityMedium, "architect", "agent1", "", nil, nil, "", false, nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_UpdateTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	existingTask := &domain.Task{
		ID:          taskID,
		ColumnID:    columnID,
		Title:       "Old Title",
		Summary:     "Old Summary",
		Description: "Old Description",
	}

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return existingTask, nil
		}
		return nil, errors.New("not found")
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	newTitle := "New Title"
	newDescription := "New Description"

	err := a.UpdateTask(ctx, projectID, taskID, &newTitle, &newDescription, nil, nil, nil, nil, nil, nil, nil, nil, nil, false)

	require.NoError(t, err)
	assert.Equal(t, "New Title", updatedTask.Title)
	assert.Equal(t, "New Description", updatedTask.Description)
}

func TestApp_UpdateTask_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	newTitle := "New Title"
	err := a.UpdateTask(ctx, projectID, taskID, &newTitle, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, false)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_UpdateTaskFiles_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	existingTask := &domain.Task{
		ID:            taskID,
		ColumnID:      columnID,
		Title:         "Test Task",
		FilesModified: []string{"old_file.go"},
		ContextFiles:  []string{"old_context.go"},
	}

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return existingTask, nil
		}
		return nil, errors.New("not found")
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	newFilesModified := []string{"internal/kanban/app/tasks.go", "internal/kanban/domain/types.go"}
	newContextFiles := []string{"CLAUDE.md"}

	err := a.UpdateTaskFiles(ctx, projectID, taskID, &newFilesModified, &newContextFiles)

	require.NoError(t, err)
	assert.Equal(t, newFilesModified, updatedTask.FilesModified)
	assert.Equal(t, newContextFiles, updatedTask.ContextFiles)
}

func TestApp_UpdateTaskFiles_OnlyFilesModified(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	originalContextFiles := []string{"original_context.go"}
	existingTask := &domain.Task{
		ID:            taskID,
		ColumnID:      columnID,
		Title:         "Test Task",
		FilesModified: []string{},
		ContextFiles:  originalContextFiles,
	}

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return existingTask, nil
		}
		return nil, errors.New("not found")
	}

	var updatedTask domain.Task
	mockTasks.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, task domain.Task) error {
		updatedTask = task
		return nil
	}

	newFilesModified := []string{"internal/kanban/app/tasks.go"}

	err := a.UpdateTaskFiles(ctx, projectID, taskID, &newFilesModified, nil)

	require.NoError(t, err)
	assert.Equal(t, newFilesModified, updatedTask.FilesModified)
	// context_files should remain unchanged when nil is passed
	assert.Equal(t, originalContextFiles, updatedTask.ContextFiles)
}

func TestApp_UpdateTaskFiles_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	newFilesModified := []string{"some_file.go"}
	err := a.UpdateTaskFiles(ctx, projectID, taskID, &newFilesModified, nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_DeleteTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, Title: "Test Task"}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.DeleteFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) error {
		return nil
	}

	err := a.DeleteTask(ctx, projectID, taskID)

	require.NoError(t, err)
}

func TestApp_ReorderTask_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, Title: "Test Task"}, nil
		}
		return nil, errors.New("not found")
	}

	var calledWithPosition int
	mockTasks.ReorderTaskFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, newPosition int) error {
		calledWithPosition = newPosition
		return nil
	}

	err := a.ReorderTask(ctx, projectID, taskID, 3)

	require.NoError(t, err)
	assert.Equal(t, 3, calledWithPosition)
}

func TestApp_ReorderTask_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.ReorderTask(ctx, projectID, taskID, 2)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_GetNextTasks_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	expectedTasks := []domain.Task{
		{ID: domain.NewTaskID(), Title: "Task 1"},
		{ID: domain.NewTaskID(), Title: "Task 2"},
	}

	mockTasks.GetNextTasksFunc = func(ctx context.Context, pid domain.ProjectID, role string, count int) ([]domain.Task, error) {
		return expectedTasks, nil
	}

	result, err := a.GetNextTasks(ctx, projectID, "developer", 2, nil)

	require.NoError(t, err)
	assert.Equal(t, expectedTasks, result)
}

func TestApp_GetNextTasks_DefaultCount(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	var calledWithCount int
	mockTasks.GetNextTasksFunc = func(ctx context.Context, pid domain.ProjectID, role string, count int) ([]domain.Task, error) {
		calledWithCount = count
		return []domain.Task{}, nil
	}

	_, err := a.GetNextTasks(ctx, projectID, "developer", 0, nil)

	require.NoError(t, err)
	assert.Equal(t, 1, calledWithCount, "count <= 0 should default to 1")
}

func TestApp_MarkTaskSeen_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, Title: "Test Task"}, nil
		}
		return nil, errors.New("not found")
	}

	seenCalled := false
	mockTasks.MarkTaskSeenFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) error {
		seenCalled = true
		return nil
	}

	err := a.MarkTaskSeen(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.True(t, seenCalled)
}

func TestApp_MarkTaskSeen_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.MarkTaskSeen(ctx, projectID, taskID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_GetTimeline_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	expectedEntries := []domain.TimelineEntry{
		{Date: "2026-03-17", TasksCreated: 3, TasksCompleted: 1},
		{Date: "2026-03-18", TasksCreated: 2, TasksCompleted: 5},
	}

	mockTasks.GetTimelineFunc = func(ctx context.Context, pid domain.ProjectID, days int) ([]domain.TimelineEntry, error) {
		if pid == projectID && days == 7 {
			return expectedEntries, nil
		}
		return nil, errors.New("not found")
	}

	entries, err := a.GetTimeline(ctx, projectID, 7)

	require.NoError(t, err)
	assert.Equal(t, expectedEntries, entries)
}

func TestApp_UpdateTaskSessionID_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	var calledWithSessionID string
	mockTasks.UpdateSessionIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, sessionID string) error {
		calledWithSessionID = sessionID
		return nil
	}

	err := a.UpdateTaskSessionID(ctx, projectID, taskID, "session-abc-123")

	require.NoError(t, err)
	assert.Equal(t, "session-abc-123", calledWithSessionID)
}
