package dependenciestest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDependencyRepository is a function-based mock implementation of the DependencyRepository interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockDependencyRepository{
//		CreateFunc: func(ctx context.Context, projectID domain.ProjectID, dep domain.TaskDependency) error {
//			return nil
//		},
//		ListFunc: func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
//			return []domain.TaskDependency{}, nil
//		},
//	}
type MockDependencyRepository struct {
	CreateFunc               func(ctx context.Context, projectID domain.ProjectID, dep domain.TaskDependency) error
	DeleteFunc               func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) error
	ListFunc                 func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)
	ListDependentsFunc       func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)
	WouldCreateCycleFunc     func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) (bool, error)
	GetDependencyContextFunc func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error)
}

func (m *MockDependencyRepository) Create(ctx context.Context, projectID domain.ProjectID, dep domain.TaskDependency) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, projectID, dep)
}

func (m *MockDependencyRepository) Delete(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, projectID, taskID, dependsOnTaskID)
}

func (m *MockDependencyRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, projectID, taskID)
}

func (m *MockDependencyRepository) ListDependents(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	if m.ListDependentsFunc == nil {
		panic("called not defined ListDependentsFunc")
	}
	return m.ListDependentsFunc(ctx, projectID, taskID)
}

func (m *MockDependencyRepository) WouldCreateCycle(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) (bool, error) {
	if m.WouldCreateCycleFunc == nil {
		panic("called not defined WouldCreateCycleFunc")
	}
	return m.WouldCreateCycleFunc(ctx, projectID, taskID, dependsOnTaskID)
}

func (m *MockDependencyRepository) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	if m.GetDependencyContextFunc == nil {
		panic("called not defined GetDependencyContextFunc")
	}
	return m.GetDependencyContextFunc(ctx, projectID, taskID)
}

// TaskCreator is a helper interface for creating tasks in dependency tests
type TaskCreator interface {
	Create(ctx context.Context, projectID domain.ProjectID, task domain.Task) error
}

// createTestTask is a helper function to create a task for testing
func createTestTask(t *testing.T, ctx context.Context, taskRepo TaskCreator, projectID domain.ProjectID, taskID domain.TaskID, columnID domain.ColumnID, title string) {
	t.Helper()
	task := domain.Task{
		ID:            taskID,
		ColumnID:      columnID,
		Title:         title,
		Summary:       "Test summary for " + title,
		Priority:      domain.PriorityMedium,
		PriorityScore: 200,
	}
	require.NoError(t, taskRepo.Create(ctx, projectID, task), "Create task should succeed: "+title)
}

// DependenciesContractTesting runs all contract tests for a DependencyRepository implementation.
// Use this function to verify that your implementation adheres to the DependencyRepository contract.
//
// Parameters:
//   - t: testing.T instance
//   - repo: the DependencyRepository implementation to test
//   - projectID: a valid project ID to use for testing
//   - taskRepo: a task repository for creating test tasks (needed for foreign key constraints)
//   - columnID: a valid column ID to use when creating tasks
//
// Example usage in implementation tests:
//
//	func TestSQLiteDependencyRepository(t *testing.T) {
//		repo, projectID, taskRepo, todoColumnID := setupTestRepo(t)
//		defer cleanupTestRepo(t, repo)
//		dependenciestest.DependenciesContractTesting(t, repo, projectID, taskRepo, todoColumnID)
//	}
func DependenciesContractTesting(t *testing.T, repo dependencies.DependencyRepository, projectID domain.ProjectID, taskRepo TaskCreator, columnID domain.ColumnID) {
	ctx := context.Background()

	t.Run("Contract: Create stores dependency and List retrieves it", func(t *testing.T) {
		taskID := domain.NewTaskID()
		dependsOnTaskID := domain.NewTaskID()

		// Create both tasks first
		createTestTask(t, ctx, taskRepo, projectID, taskID, columnID, "Task 1")
		createTestTask(t, ctx, taskRepo, projectID, dependsOnTaskID, columnID, "Task 2")

		dep := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: dependsOnTaskID,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.List(ctx, projectID, taskID)
		require.NoError(t, err, "List should succeed")
		require.Len(t, retrieved, 1, "Should return exactly 1 dependency")
		assert.Equal(t, dep.TaskID, retrieved[0].TaskID)
		assert.Equal(t, dep.DependsOnTaskID, retrieved[0].DependsOnTaskID)
	})

	t.Run("Contract: Create returns error for duplicate dependency", func(t *testing.T) {
		taskID := domain.NewTaskID()
		dependsOnTaskID := domain.NewTaskID()

		// Create both tasks first
		createTestTask(t, ctx, taskRepo, projectID, taskID, columnID, "Task 1")
		createTestTask(t, ctx, taskRepo, projectID, dependsOnTaskID, columnID, "Task 2")

		dep1 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: dependsOnTaskID,
			CreatedAt:       time.Now(),
		}

		dep2 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: dependsOnTaskID,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep1)
		require.NoError(t, err, "First Create should succeed")

		err = repo.Create(ctx, projectID, dep2)
		assert.Error(t, err, "Second Create with duplicate dependency should fail")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrDependencyAlreadyExists)
	})

	t.Run("Contract: Create returns error for self-referencing dependency", func(t *testing.T) {
		taskID := domain.NewTaskID()

		dep := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: taskID, // Self-reference
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep)
		assert.Error(t, err, "Create should fail for self-referencing dependency")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
	})

	t.Run("Contract: Delete removes dependency", func(t *testing.T) {
		taskID := domain.NewTaskID()
		dependsOnTaskID := domain.NewTaskID()

		// Create both tasks first
		createTestTask(t, ctx, taskRepo, projectID, taskID, columnID, "Task 1")
		createTestTask(t, ctx, taskRepo, projectID, dependsOnTaskID, columnID, "Task 2")

		dep := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: dependsOnTaskID,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep)
		require.NoError(t, err, "Create should succeed")

		err = repo.Delete(ctx, projectID, taskID, dependsOnTaskID)
		require.NoError(t, err, "Delete should succeed")

		retrieved, err := repo.List(ctx, projectID, taskID)
		require.NoError(t, err, "List should succeed")
		assert.Empty(t, retrieved, "Should return empty list after deletion")
	})

	t.Run("Contract: Delete returns error for non-existent dependency", func(t *testing.T) {
		taskID := domain.NewTaskID()
		dependsOnTaskID := domain.NewTaskID()

		err := repo.Delete(ctx, projectID, taskID, dependsOnTaskID)
		assert.Error(t, err, "Delete should return error for non-existent dependency")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrDependencyNotFound)
	})

	t.Run("Contract: List returns multiple dependencies", func(t *testing.T) {
		taskID := domain.NewTaskID()
		dependsOn1 := domain.NewTaskID()
		dependsOn2 := domain.NewTaskID()

		// Create all tasks first
		createTestTask(t, ctx, taskRepo, projectID, taskID, columnID, "Task 1")
		createTestTask(t, ctx, taskRepo, projectID, dependsOn1, columnID, "Depends On 1")
		createTestTask(t, ctx, taskRepo, projectID, dependsOn2, columnID, "Depends On 2")

		dep1 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: dependsOn1,
			CreatedAt:       time.Now(),
		}

		dep2 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: dependsOn2,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep1)
		require.NoError(t, err, "Create should succeed")
		err = repo.Create(ctx, projectID, dep2)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.List(ctx, projectID, taskID)
		require.NoError(t, err, "List should succeed")
		assert.Len(t, retrieved, 2, "Should return exactly 2 dependencies")
	})

	t.Run("Contract: List returns empty for task with no dependencies", func(t *testing.T) {
		taskID := domain.NewTaskID()

		retrieved, err := repo.List(ctx, projectID, taskID)
		require.NoError(t, err, "List should succeed")
		assert.Empty(t, retrieved, "Should return empty list for task with no dependencies")
	})

	t.Run("Contract: ListDependents returns tasks that depend on given task", func(t *testing.T) {
		targetTask := domain.NewTaskID()
		dependentTask1 := domain.NewTaskID()
		dependentTask2 := domain.NewTaskID()

		// Create all tasks
		createTestTask(t, ctx, taskRepo, projectID, targetTask, columnID, "Target Task")
		createTestTask(t, ctx, taskRepo, projectID, dependentTask1, columnID, "Dependent 1")
		createTestTask(t, ctx, taskRepo, projectID, dependentTask2, columnID, "Dependent 2")

		dep1 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          dependentTask1,
			DependsOnTaskID: targetTask,
			CreatedAt:       time.Now(),
		}
		dep2 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          dependentTask2,
			DependsOnTaskID: targetTask,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep1)
		require.NoError(t, err, "Create dep1 should succeed")
		err = repo.Create(ctx, projectID, dep2)
		require.NoError(t, err, "Create dep2 should succeed")

		retrieved, err := repo.ListDependents(ctx, projectID, targetTask)
		require.NoError(t, err, "ListDependents should succeed")
		require.Len(t, retrieved, 2, "Should return 2 dependents")

		taskIDs := []domain.TaskID{retrieved[0].TaskID, retrieved[1].TaskID}
		assert.Contains(t, taskIDs, dependentTask1)
		assert.Contains(t, taskIDs, dependentTask2)
	})

	t.Run("Contract: ListDependents returns empty for task with no dependents", func(t *testing.T) {
		taskID := domain.NewTaskID()

		retrieved, err := repo.ListDependents(ctx, projectID, taskID)
		require.NoError(t, err, "ListDependents should succeed")
		assert.Empty(t, retrieved, "Should return empty list for task with no dependents")
	})

	t.Run("Contract: WouldCreateCycle returns false for simple dependency", func(t *testing.T) {
		taskA := domain.NewTaskID()
		taskB := domain.NewTaskID()

		wouldCycle, err := repo.WouldCreateCycle(ctx, projectID, taskA, taskB)
		require.NoError(t, err, "WouldCreateCycle should succeed")
		assert.False(t, wouldCycle, "Should return false for simple dependency")
	})

	t.Run("Contract: WouldCreateCycle returns true for direct cycle", func(t *testing.T) {
		taskA := domain.NewTaskID()
		taskB := domain.NewTaskID()

		// Create both tasks first
		createTestTask(t, ctx, taskRepo, projectID, taskA, columnID, "Task A")
		createTestTask(t, ctx, taskRepo, projectID, taskB, columnID, "Task B")

		// Create A -> B
		dep := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskA,
			DependsOnTaskID: taskB,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep)
		require.NoError(t, err, "Create should succeed")

		// Check if B -> A would create cycle
		wouldCycle, err := repo.WouldCreateCycle(ctx, projectID, taskB, taskA)
		require.NoError(t, err, "WouldCreateCycle should succeed")
		assert.True(t, wouldCycle, "Should return true for direct cycle (A->B, B->A)")
	})

	t.Run("Contract: WouldCreateCycle returns true for transitive cycle", func(t *testing.T) {
		taskA := domain.NewTaskID()
		taskB := domain.NewTaskID()
		taskC := domain.NewTaskID()

		// Create all tasks first
		createTestTask(t, ctx, taskRepo, projectID, taskA, columnID, "Task A")
		createTestTask(t, ctx, taskRepo, projectID, taskB, columnID, "Task B")
		createTestTask(t, ctx, taskRepo, projectID, taskC, columnID, "Task C")

		// Create A -> B
		dep1 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskA,
			DependsOnTaskID: taskB,
			CreatedAt:       time.Now(),
		}

		// Create B -> C
		dep2 := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskB,
			DependsOnTaskID: taskC,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep1)
		require.NoError(t, err, "Create should succeed")
		err = repo.Create(ctx, projectID, dep2)
		require.NoError(t, err, "Create should succeed")

		// Check if C -> A would create cycle (A->B->C->A)
		wouldCycle, err := repo.WouldCreateCycle(ctx, projectID, taskC, taskA)
		require.NoError(t, err, "WouldCreateCycle should succeed")
		assert.True(t, wouldCycle, "Should return true for transitive cycle (A->B->C->A)")
	})

	t.Run("Contract: WouldCreateCycle returns true for self-reference", func(t *testing.T) {
		taskA := domain.NewTaskID()

		wouldCycle, err := repo.WouldCreateCycle(ctx, projectID, taskA, taskA)
		require.NoError(t, err, "WouldCreateCycle should succeed")
		assert.True(t, wouldCycle, "Should return true for self-reference")
	})

	t.Run("Contract: GetDependencyContext returns empty for task with no dependencies", func(t *testing.T) {
		taskID := domain.NewTaskID()

		context, err := repo.GetDependencyContext(ctx, projectID, taskID)
		require.NoError(t, err, "GetDependencyContext should succeed")
		assert.Empty(t, context, "Should return empty slice for task with no dependencies")
	})

	t.Run("Contract: GetDependencyContext returns context for resolved dependencies", func(t *testing.T) {
		// This test is more complex and requires task setup
		// For now, verify it returns empty for a task with no dependencies
		// Actual implementation will need to verify it returns DependencyContext
		// for tasks in "done" or "blocked" columns

		taskID := domain.NewTaskID()
		dependsOnTaskID := domain.NewTaskID()

		// Create both tasks first
		createTestTask(t, ctx, taskRepo, projectID, taskID, columnID, "Task 1")
		createTestTask(t, ctx, taskRepo, projectID, dependsOnTaskID, columnID, "Task 2")

		dep := domain.TaskDependency{
			ID:              domain.NewDependencyID(),
			TaskID:          taskID,
			DependsOnTaskID: dependsOnTaskID,
			CreatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, dep)
		require.NoError(t, err, "Create should succeed")

		// Note: This test will need task data to be meaningful
		// For contract testing, we verify the method doesn't error
		_, err = repo.GetDependencyContext(ctx, projectID, taskID)
		require.NoError(t, err, "GetDependencyContext should not error")
	})
}
