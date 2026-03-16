package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Dependency Command Tests

func TestApp_AddDependency_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, mockDependencies := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	dependsOnTaskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	callCount := 0
	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		callCount++
		if pid == projectID && (tid == taskID || tid == dependsOnTaskID) {
			return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockDependencies.WouldCreateCycleFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, depTaskID domain.TaskID) (bool, error) {
		return false, nil
	}

	var createdDep domain.TaskDependency
	mockDependencies.CreateFunc = func(ctx context.Context, pid domain.ProjectID, dep domain.TaskDependency) error {
		createdDep = dep
		return nil
	}

	err := a.AddDependency(ctx, projectID, taskID, dependsOnTaskID)

	require.NoError(t, err)
	assert.Equal(t, taskID, createdDep.TaskID)
	assert.Equal(t, dependsOnTaskID, createdDep.DependsOnTaskID)
	assert.NotEmpty(t, createdDep.ID)
}

func TestApp_AddDependency_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	dependsOnTaskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	err := a.AddDependency(ctx, projectID, taskID, dependsOnTaskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_AddDependency_DependsOnTaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	dependsOnTaskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: columnID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	err := a.AddDependency(ctx, projectID, taskID, dependsOnTaskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_AddDependency_CycleDetected_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, mockDependencies := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	dependsOnTaskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "Summary"}, nil
	}

	mockDependencies.WouldCreateCycleFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, depTaskID domain.TaskID) (bool, error) {
		return true, nil
	}

	err := a.AddDependency(ctx, projectID, taskID, dependsOnTaskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrDependencyCycle)
}

func TestApp_RemoveDependency_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, mockDependencies := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	dependsOnTaskID := domain.NewTaskID()

	mockDependencies.DeleteFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, depTaskID domain.TaskID) error {
		return nil
	}

	err := a.RemoveDependency(ctx, projectID, taskID, dependsOnTaskID)

	require.NoError(t, err)
}

func TestApp_RemoveDependency_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, mockDependencies := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	dependsOnTaskID := domain.NewTaskID()

	mockDependencies.DeleteFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, depTaskID domain.TaskID) error {
		return errors.Join(domain.ErrDependencyNotFound, errors.New("not found"))
	}

	err := a.RemoveDependency(ctx, projectID, taskID, dependsOnTaskID)

	assert.Error(t, err)
}

// Dependency Query Tests

func TestApp_ListDependencies_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, mockDependencies := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()
	dep1TaskID := domain.NewTaskID()
	dep2TaskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: columnID, Title: "Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	expectedDeps := []domain.TaskDependency{
		{ID: domain.NewDependencyID(), TaskID: taskID, DependsOnTaskID: dep1TaskID},
		{ID: domain.NewDependencyID(), TaskID: taskID, DependsOnTaskID: dep2TaskID},
	}

	mockDependencies.ListFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) ([]domain.TaskDependency, error) {
		if pid == projectID && tid == taskID {
			return expectedDeps, nil
		}
		return nil, errors.New("not found")
	}

	deps, err := a.ListDependencies(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.Equal(t, expectedDeps, deps)
}

func TestApp_ListDependencies_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	_, err := a.ListDependencies(ctx, projectID, taskID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_GetDependencyContext_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, mockDependencies := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	expectedContext := []domain.DependencyContext{
		{
			TaskID:            domain.NewTaskID(),
			Title:             "Completed dependency",
			CompletionSummary: "Done with this",
			FilesModified:     []string{"file1.go"},
		},
	}

	mockDependencies.GetDependencyContextFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) ([]domain.DependencyContext, error) {
		if pid == projectID && tid == taskID {
			return expectedContext, nil
		}
		return nil, errors.New("not found")
	}

	depCtx, err := a.GetDependencyContext(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.Equal(t, expectedContext, depCtx)
}

func TestApp_GetDependencyContext_ReturnsEmpty_WhenNoDeps(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, mockDependencies := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockDependencies.GetDependencyContextFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) ([]domain.DependencyContext, error) {
		return []domain.DependencyContext{}, nil
	}

	depCtx, err := a.GetDependencyContext(ctx, projectID, taskID)

	require.NoError(t, err)
	assert.Empty(t, depCtx)
}
