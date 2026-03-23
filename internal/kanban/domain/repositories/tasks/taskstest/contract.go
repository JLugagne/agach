package taskstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTaskRepository is a function-based mock implementation of the TaskRepository interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockTaskRepository{
//		CreateFunc: func(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
//			return nil
//		},
//		FindByIDFunc: func(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
//			return &domain.Task{ID: id, Title: "Test Task"}, nil
//		},
//	}
type MockTaskRepository struct {
	BulkCreateFunc                func(ctx context.Context, projectID domain.ProjectID, tasks []domain.Task) error
	BulkReassignInProjectFunc     func(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error)
	ListByAssignedRoleFunc        func(ctx context.Context, projectID domain.ProjectID, slug string) ([]domain.Task, error)
	CreateFunc                    func(ctx context.Context, projectID domain.ProjectID, task domain.Task) error
	FindByIDFunc                  func(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) (*domain.Task, error)
	ListFunc                      func(ctx context.Context, projectID domain.ProjectID, filters tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error)
	UpdateFunc                    func(ctx context.Context, projectID domain.ProjectID, task domain.Task) error
	DeleteFunc                    func(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) error
	MoveFunc                      func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnID domain.ColumnID) error
	CountByColumnFunc             func(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (int, error)
	GetNextTaskFunc               func(ctx context.Context, projectID domain.ProjectID, role string) (*domain.Task, error)
	GetNextTasksFunc              func(ctx context.Context, projectID domain.ProjectID, role string, count int) ([]domain.Task, error)
	HasUnresolvedDependenciesFunc func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (bool, error)
	GetDependentsNotDoneFunc      func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
	MarkTaskSeenFunc              func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	ReorderTaskFunc               func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error
	GetTimelineFunc               func(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error)
	UpdateSessionIDFunc           func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error
	GetColdStartStatsFunc         func(ctx context.Context, projectID domain.ProjectID) ([]domain.RoleColdStartStat, error)
	GetModelTokenStatsFunc        func(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error)
}

func (m *MockTaskRepository) BulkCreate(ctx context.Context, projectID domain.ProjectID, tasks []domain.Task) error {
	if m.BulkCreateFunc == nil {
		panic("called not defined BulkCreateFunc")
	}
	return m.BulkCreateFunc(ctx, projectID, tasks)
}

func (m *MockTaskRepository) BulkReassignInProject(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error) {
	if m.BulkReassignInProjectFunc == nil {
		panic("called not defined BulkReassignInProjectFunc")
	}
	return m.BulkReassignInProjectFunc(ctx, projectID, oldSlug, newSlug)
}

func (m *MockTaskRepository) ListByAssignedRole(ctx context.Context, projectID domain.ProjectID, slug string) ([]domain.Task, error) {
	if m.ListByAssignedRoleFunc == nil {
		panic("called not defined ListByAssignedRoleFunc")
	}
	return m.ListByAssignedRoleFunc(ctx, projectID, slug)
}

func (m *MockTaskRepository) Create(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, projectID, task)
}

func (m *MockTaskRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, projectID, id)
}

func (m *MockTaskRepository) List(ctx context.Context, projectID domain.ProjectID, filters tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, projectID, filters)
}

func (m *MockTaskRepository) Update(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, projectID, task)
}

func (m *MockTaskRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) error {
	if m.DeleteFunc == nil {
		return nil
	}
	return m.DeleteFunc(ctx, projectID, id)
}

func (m *MockTaskRepository) Move(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnID domain.ColumnID) error {
	if m.MoveFunc == nil {
		panic("called not defined MoveFunc")
	}
	return m.MoveFunc(ctx, projectID, taskID, targetColumnID)
}

func (m *MockTaskRepository) CountByColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (int, error) {
	if m.CountByColumnFunc == nil {
		panic("called not defined CountByColumnFunc")
	}
	return m.CountByColumnFunc(ctx, projectID, columnID)
}

func (m *MockTaskRepository) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string) (*domain.Task, error) {
	if m.GetNextTaskFunc == nil {
		panic("called not defined GetNextTaskFunc")
	}
	return m.GetNextTaskFunc(ctx, projectID, role)
}

func (m *MockTaskRepository) GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int) ([]domain.Task, error) {
	if m.GetNextTasksFunc == nil {
		panic("called not defined GetNextTasksFunc")
	}
	return m.GetNextTasksFunc(ctx, projectID, role, count)
}

func (m *MockTaskRepository) HasUnresolvedDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (bool, error) {
	if m.HasUnresolvedDependenciesFunc == nil {
		panic("called not defined HasUnresolvedDependenciesFunc")
	}
	return m.HasUnresolvedDependenciesFunc(ctx, projectID, taskID)
}

func (m *MockTaskRepository) GetDependentsNotDone(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	if m.GetDependentsNotDoneFunc == nil {
		panic("called not defined GetDependentsNotDoneFunc")
	}
	return m.GetDependentsNotDoneFunc(ctx, projectID, taskID)
}

func (m *MockTaskRepository) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.MarkTaskSeenFunc == nil {
		panic("called not defined MarkTaskSeenFunc")
	}
	return m.MarkTaskSeenFunc(ctx, projectID, taskID)
}

func (m *MockTaskRepository) ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error {
	if m.ReorderTaskFunc == nil {
		panic("called not defined ReorderTaskFunc")
	}
	return m.ReorderTaskFunc(ctx, projectID, taskID, newPosition)
}

func (m *MockTaskRepository) GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error) {
	if m.GetTimelineFunc == nil {
		panic("called not defined GetTimelineFunc")
	}
	return m.GetTimelineFunc(ctx, projectID, days)
}

func (m *MockTaskRepository) UpdateSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error {
	if m.UpdateSessionIDFunc == nil {
		panic("called not defined UpdateSessionIDFunc")
	}
	return m.UpdateSessionIDFunc(ctx, projectID, taskID, sessionID)
}

func (m *MockTaskRepository) GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.RoleColdStartStat, error) {
	if m.GetColdStartStatsFunc == nil {
		panic("called not defined GetColdStartStatsFunc")
	}
	return m.GetColdStartStatsFunc(ctx, projectID)
}

func (m *MockTaskRepository) GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error) {
	if m.GetModelTokenStatsFunc == nil {
		panic("called not defined GetModelTokenStatsFunc")
	}
	return m.GetModelTokenStatsFunc(ctx, projectID)
}

// TasksContractTesting runs all contract tests for a TaskRepository implementation.
// Use this function to verify that your implementation adheres to the TaskRepository contract.
//
// Parameters:
//   - t: testing.T instance
//   - repo: the TaskRepository implementation to test
//   - projectID: a valid project ID to use for testing
//   - todoColumnID: the ID of the "todo" column
//   - inProgressColumnID: the ID of the "in_progress" column
//   - doneColumnID: the ID of the "done" column
//   - featureProjectID: a valid project ID (child of projectID) to use as feature_id in tests
//
// Example usage in implementation tests:
//
//	func TestSQLiteTaskRepository(t *testing.T) {
//		repo, projectID, columnIDs, featureProjectID := setupTestRepo(t)
//		defer cleanupTestRepo(t, repo)
//		taskstest.TasksContractTesting(t, repo, projectID, columnIDs.Todo, columnIDs.InProgress, columnIDs.Done, featureProjectID)
//	}
func TasksContractTesting(t *testing.T, repo tasksrepo.TaskRepository, projectID domain.ProjectID, todoColumnID, inProgressColumnID, doneColumnID domain.ColumnID, featureProjectID domain.ProjectID) {
	ctx := context.Background()

	t.Run("Contract: Create stores task and FindByID retrieves it", func(t *testing.T) {
		task := domain.Task{
			ID:              domain.NewTaskID(),
			ColumnID:        todoColumnID,
			Title:           "Test Task",
			Summary:         "This is a test task summary",
			Description:     "This is a test task description",
			Priority:        domain.PriorityHigh,
			PriorityScore:   domain.PriorityHigh.Score(),
			Position:        0,
			CreatedByRole:   "developer",
			CreatedByAgent:  "test-agent",
			AssignedRole:    "developer",
			IsBlocked:       false,
			WontDoRequested: false,
			ContextFiles:    []string{"file1.go", "file2.go"},
			Tags:            []string{"backend", "bug"},
			EstimatedEffort: "2h",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved, "Retrieved task must not be nil")
		assert.Equal(t, task.ID, retrieved.ID)
		assert.Equal(t, task.Title, retrieved.Title)
		assert.Equal(t, task.Summary, retrieved.Summary)
		assert.Equal(t, task.Description, retrieved.Description)
		assert.Equal(t, task.Priority, retrieved.Priority)
		assert.Equal(t, task.PriorityScore, retrieved.PriorityScore)
		assert.Equal(t, task.CreatedByRole, retrieved.CreatedByRole)
		assert.Equal(t, task.AssignedRole, retrieved.AssignedRole)
		assert.Equal(t, task.Tags, retrieved.Tags)
	})

	t.Run("Contract: FindByID returns error for non-existent task", func(t *testing.T) {
		nonExistentID := domain.NewTaskID()
		_, err := repo.FindByID(ctx, projectID, nonExistentID)
		assert.Error(t, err, "FindByID should return error for non-existent task")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("Contract: List returns tasks with filters", func(t *testing.T) {
		// Create multiple tasks
		tasks := []domain.Task{
			{
				ID:            domain.NewTaskID(),
				ColumnID:      todoColumnID,
				Title:         "Task 1",
				Summary:       "Summary 1",
				Priority:      domain.PriorityHigh,
				PriorityScore: domain.PriorityHigh.Score(),
				AssignedRole:  "developer",
				Tags:          []string{"backend"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            domain.NewTaskID(),
				ColumnID:      inProgressColumnID,
				Title:         "Task 2",
				Summary:       "Summary 2",
				Priority:      domain.PriorityMedium,
				PriorityScore: domain.PriorityMedium.Score(),
				AssignedRole:  "designer",
				Tags:          []string{"frontend"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
		}

		for _, task := range tasks {
			err := repo.Create(ctx, projectID, task)
			require.NoError(t, err, "Create should succeed")
		}

		// Test filter by column
		columnSlug := domain.ColumnTodo
		filters := tasksrepo.TaskFilters{ColumnSlug: &columnSlug}
		retrieved, err := repo.List(ctx, projectID, filters)
		require.NoError(t, err, "List should succeed")
		require.NotEmpty(t, retrieved, "List should return tasks")

		// Find our test task
		found := false
		for _, r := range retrieved {
			if r.ID == tasks[0].ID {
				found = true
				assert.Equal(t, "Task 1", r.Title)
				break
			}
		}
		assert.True(t, found, "Should find task in todo column")
	})

	t.Run("Contract: Update modifies task data", func(t *testing.T) {
		task := domain.Task{
			ID:              domain.NewTaskID(),
			ColumnID:        todoColumnID,
			Title:           "Original Title",
			Summary:         "Original summary",
			Description:     "Original description",
			Priority:        domain.PriorityMedium,
			PriorityScore:   domain.PriorityMedium.Score(),
			AssignedRole:    "developer",
			EstimatedEffort: "1h",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		// Update the task
		task.Title = "Updated Title"
		task.Description = "Updated description"
		task.EstimatedEffort = "3h"
		task.UpdatedAt = time.Now()

		err = repo.Update(ctx, projectID, task)
		require.NoError(t, err, "Update should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, "Updated Title", retrieved.Title)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, "3h", retrieved.EstimatedEffort)
		assert.Equal(t, "Original summary", retrieved.Summary, "Summary should remain unchanged")
	})

	t.Run("Contract: Update returns error for non-existent task", func(t *testing.T) {
		nonExistentTask := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Non-existent",
			Summary:       "Non-existent summary",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Update(ctx, projectID, nonExistentTask)
		assert.Error(t, err, "Update should return error for non-existent task")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("Contract: Delete removes task", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Delete Task",
			Summary:       "Delete summary",
			Priority:      domain.PriorityLow,
			PriorityScore: domain.PriorityLow.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		err = repo.Delete(ctx, projectID, task.ID)
		require.NoError(t, err, "Delete should succeed")

		_, err = repo.FindByID(ctx, projectID, task.ID)
		assert.Error(t, err, "FindByID should return error for deleted task")
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("Contract: Delete returns error for non-existent task", func(t *testing.T) {
		nonExistentID := domain.NewTaskID()
		err := repo.Delete(ctx, projectID, nonExistentID)
		assert.Error(t, err, "Delete should return error for non-existent task")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("Contract: Move changes task column", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Move Task",
			Summary:       "Move summary",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		err = repo.Move(ctx, projectID, task.ID, inProgressColumnID)
		require.NoError(t, err, "Move should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, inProgressColumnID, retrieved.ColumnID)
	})

	t.Run("Contract: Move returns error for non-existent task", func(t *testing.T) {
		nonExistentID := domain.NewTaskID()
		err := repo.Move(ctx, projectID, nonExistentID, inProgressColumnID)
		assert.Error(t, err, "Move should return error for non-existent task")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("Contract: CountByColumn returns correct count", func(t *testing.T) {
		// Create tasks in different columns
		task1 := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Count Task 1",
			Summary:       "Summary 1",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		task2 := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Count Task 2",
			Summary:       "Summary 2",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task1)
		require.NoError(t, err, "Create should succeed")
		err = repo.Create(ctx, projectID, task2)
		require.NoError(t, err, "Create should succeed")

		count, err := repo.CountByColumn(ctx, projectID, todoColumnID)
		require.NoError(t, err, "CountByColumn should succeed")
		assert.GreaterOrEqual(t, count, 2, "Should count at least 2 tasks in todo column")
	})

	t.Run("Contract: GetNextTask returns highest priority task", func(t *testing.T) {
		// Create tasks with different priorities
		lowPriorityTask := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Low Priority",
			Summary:       "Low summary",
			Priority:      domain.PriorityLow,
			PriorityScore: domain.PriorityLow.Score(),
			AssignedRole:  "developer",
			IsBlocked:     false,
			CreatedAt:     time.Now().Add(-2 * time.Hour),
			UpdatedAt:     time.Now(),
		}

		highPriorityTask := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "High Priority",
			Summary:       "High summary",
			Priority:      domain.PriorityHigh,
			PriorityScore: domain.PriorityHigh.Score(),
			AssignedRole:  "developer",
			IsBlocked:     false,
			CreatedAt:     time.Now().Add(-1 * time.Hour),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, lowPriorityTask)
		require.NoError(t, err, "Create should succeed")
		err = repo.Create(ctx, projectID, highPriorityTask)
		require.NoError(t, err, "Create should succeed")

		nextTask, err := repo.GetNextTask(ctx, projectID, "developer")
		require.NoError(t, err, "GetNextTask should succeed")
		require.NotNil(t, nextTask, "GetNextTask should return a task")
		assert.Equal(t, highPriorityTask.ID, nextTask.ID, "Should return highest priority task")
	})

	t.Run("Contract: GetNextTask filters blocked tasks", func(t *testing.T) {
		blockedTask := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Blocked Task",
			Summary:       "Blocked summary",
			Priority:      domain.PriorityCritical,
			PriorityScore: domain.PriorityCritical.Score(),
			AssignedRole:  "tester",
			IsBlocked:     true,
			BlockedReason: "Waiting for clarification",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, blockedTask)
		require.NoError(t, err, "Create should succeed")

		nextTask, err := repo.GetNextTask(ctx, projectID, "tester")
		// Should either return nil or a different task, not the blocked one
		if nextTask != nil {
			assert.NotEqual(t, blockedTask.ID, nextTask.ID, "Should not return blocked task")
		}
	})

	t.Run("Contract: HasUnresolvedDependencies returns false for task without dependencies", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			Title:         "Independent Task",
			Summary:       "Independent summary",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		hasUnresolved, err := repo.HasUnresolvedDependencies(ctx, projectID, task.ID)
		require.NoError(t, err, "HasUnresolvedDependencies should succeed")
		assert.False(t, hasUnresolved, "Task without dependencies should return false")
	})

	t.Run("Contract: GetDependentsNotDone returns empty for task without dependents", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      doneColumnID,
			Title:         "Completed Task",
			Summary:       "Completed summary",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		dependents, err := repo.GetDependentsNotDone(ctx, projectID, task.ID)
		require.NoError(t, err, "GetDependentsNotDone should succeed")
		assert.Empty(t, dependents, "Task without dependents should return empty slice")
	})

	t.Run("Contract: MarkTaskSeen sets seen_at on first call", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      doneColumnID,
			Title:         "Seen Task",
			Summary:       "Seen summary",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		// Initially seen_at should be nil
		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Nil(t, retrieved.SeenAt, "seen_at should be nil before first view")

		// Mark as seen
		err = repo.MarkTaskSeen(ctx, projectID, task.ID)
		require.NoError(t, err, "MarkTaskSeen should succeed")

		// Now seen_at should be set
		retrieved, err = repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed after marking seen")
		assert.NotNil(t, retrieved.SeenAt, "seen_at should be set after MarkTaskSeen")
	})

	t.Run("Contract: MarkTaskSeen is idempotent (does not overwrite existing seen_at)", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      doneColumnID,
			Title:         "Idempotent Seen Task",
			Summary:       "Idempotent seen summary",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create should succeed")

		err = repo.MarkTaskSeen(ctx, projectID, task.ID)
		require.NoError(t, err, "First MarkTaskSeen should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved.SeenAt, "seen_at must be set")
		firstSeenAt := *retrieved.SeenAt

		// Call again — should not overwrite
		err = repo.MarkTaskSeen(ctx, projectID, task.ID)
		require.NoError(t, err, "Second MarkTaskSeen should succeed (idempotent)")

		retrieved, err = repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed after second call")
		require.NotNil(t, retrieved.SeenAt, "seen_at must still be set")
		assert.Equal(t, firstSeenAt, *retrieved.SeenAt, "seen_at must not change on second call")
	})

	t.Run("Contract: MarkTaskSeen returns error for non-existent task", func(t *testing.T) {
		nonExistentID := domain.NewTaskID()
		err := repo.MarkTaskSeen(ctx, projectID, nonExistentID)
		assert.Error(t, err, "MarkTaskSeen should return error for non-existent task")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("Contract: Create stores feature_id and FindByID retrieves it", func(t *testing.T) {
		featureID := featureProjectID
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			FeatureID:     &featureID,
			Title:         "Task with Feature",
			Summary:       "Task belonging to a feature",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create with feature_id should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved, "Retrieved task must not be nil")
		require.NotNil(t, retrieved.FeatureID, "FeatureID must not be nil after retrieval")
		assert.Equal(t, featureID, *retrieved.FeatureID, "FeatureID must match the stored value")
	})

	t.Run("Contract: Create with nil feature_id stores NULL", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			FeatureID:     nil,
			Title:         "Task without Feature",
			Summary:       "Root-level task with no feature",
			Priority:      domain.PriorityLow,
			PriorityScore: domain.PriorityLow.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.Create(ctx, projectID, task)
		require.NoError(t, err, "Create with nil feature_id should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved, "Retrieved task must not be nil")
		assert.Nil(t, retrieved.FeatureID, "FeatureID must be nil for root-level task")
	})

	t.Run("Contract: BulkCreate preserves feature_id for each task", func(t *testing.T) {
		featureID := featureProjectID

		tasks := []domain.Task{
			{
				ID:            domain.NewTaskID(),
				ColumnID:      todoColumnID,
				FeatureID:     &featureID,
				Title:         "Bulk Task 1",
				Summary:       "First bulk task with feature",
				Priority:      domain.PriorityMedium,
				PriorityScore: domain.PriorityMedium.Score(),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            domain.NewTaskID(),
				ColumnID:      todoColumnID,
				FeatureID:     nil,
				Title:         "Bulk Task 2",
				Summary:       "Second bulk task without feature",
				Priority:      domain.PriorityLow,
				PriorityScore: domain.PriorityLow.Score(),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
		}

		err := repo.BulkCreate(ctx, projectID, tasks)
		require.NoError(t, err, "BulkCreate should succeed")

		retrieved0, err := repo.FindByID(ctx, projectID, tasks[0].ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved0.FeatureID, "First task should have feature_id set")
		assert.Equal(t, featureID, *retrieved0.FeatureID)

		retrieved1, err := repo.FindByID(ctx, projectID, tasks[1].ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved1.FeatureID, "Second task should have nil feature_id")
	})

	t.Run("Contract: Update can change feature_id", func(t *testing.T) {
		featureID := featureProjectID
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			FeatureID:     nil,
			Title:         "Task to update feature_id",
			Summary:       "Will be assigned a feature",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		require.NoError(t, repo.Create(ctx, projectID, task))

		task.FeatureID = &featureID
		task.UpdatedAt = time.Now()
		require.NoError(t, repo.Update(ctx, projectID, task))

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.FeatureID, "feature_id should be set after update")
		assert.Equal(t, featureID, *retrieved.FeatureID)
	})

	t.Run("Contract: Update can clear feature_id to nil", func(t *testing.T) {
		featureID := featureProjectID
		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			FeatureID:     &featureID,
			Title:         "Task to clear feature_id",
			Summary:       "Feature will be cleared",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		require.NoError(t, repo.Create(ctx, projectID, task))

		task.FeatureID = nil
		task.UpdatedAt = time.Now()
		require.NoError(t, repo.Update(ctx, projectID, task))

		retrieved, err := repo.FindByID(ctx, projectID, task.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved.FeatureID, "feature_id should be nil after clearing")
	})

	t.Run("Contract: List filters by feature_id", func(t *testing.T) {
		featureID := featureProjectID

		// taskA has feature_id set; taskB has no feature (nil), so filtering by featureID excludes it.
		taskA := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			FeatureID:     &featureID,
			Title:         "Feature A Task",
			Summary:       "Belongs to feature A",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		taskB := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      todoColumnID,
			FeatureID:     nil,
			Title:         "No Feature Task",
			Summary:       "Root-level task with no feature",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		require.NoError(t, repo.Create(ctx, projectID, taskA))
		require.NoError(t, repo.Create(ctx, projectID, taskB))

		filters := tasksrepo.TaskFilters{FeatureID: &featureID}
		results, err := repo.List(ctx, projectID, filters)
		require.NoError(t, err)

		foundA, foundB := false, false
		for _, r := range results {
			if r.ID == taskA.ID {
				foundA = true
			}
			if r.ID == taskB.ID {
				foundB = true
			}
		}
		assert.True(t, foundA, "Should find task belonging to feature A")
		assert.False(t, foundB, "Should not find root-level task when filtering by feature_id")
	})
}
