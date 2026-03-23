package commentstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommentRepository is a function-based mock implementation of the CommentRepository interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockCommentRepository{
//		CreateFunc: func(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
//			return nil
//		},
//		FindByIDFunc: func(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) (*domain.Comment, error) {
//			return &domain.Comment{ID: id, Content: "Test comment"}, nil
//		},
//	}
type MockCommentRepository struct {
	CreateFunc        func(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error
	FindByIDFunc      func(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) (*domain.Comment, error)
	ListFunc          func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error)
	UpdateFunc        func(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error
	DeleteFunc        func(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) error
	CountFunc         func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error)
	IsLastCommentFunc func(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (bool, error)
}

func (m *MockCommentRepository) Create(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, projectID, comment)
}

func (m *MockCommentRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) (*domain.Comment, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, projectID, id)
}

func (m *MockCommentRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, projectID, taskID, limit, offset)
}

func (m *MockCommentRepository) Update(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, projectID, comment)
}

func (m *MockCommentRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, projectID, id)
}

func (m *MockCommentRepository) Count(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error) {
	if m.CountFunc == nil {
		panic("called not defined CountFunc")
	}
	return m.CountFunc(ctx, projectID, taskID)
}

func (m *MockCommentRepository) IsLastComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (bool, error) {
	if m.IsLastCommentFunc == nil {
		panic("called not defined IsLastCommentFunc")
	}
	return m.IsLastCommentFunc(ctx, projectID, commentID)
}

// CommentsContractTesting runs all contract tests for a CommentRepository implementation.
// Use this function to verify that your implementation adheres to the CommentRepository contract.
//
// Parameters:
//   - t: testing.T instance
//   - repo: the CommentRepository implementation to test
//   - projectID: a valid project ID to use for testing
//   - taskRepo: a task repository to create test tasks
//   - columnID: a valid column ID for creating test tasks
//
// Example usage in implementation tests:
//
//	func TestSQLiteCommentRepository(t *testing.T) {
//		repo, projectID, taskRepo, columnID := setupTestRepo(t)
//		defer cleanupTestRepo(t, repo)
//		commentstest.CommentsContractTesting(t, repo, projectID, taskRepo, columnID)
//	}
func CommentsContractTesting(t *testing.T, repo comments.CommentRepository, projectID domain.ProjectID, taskRepo interface {
	Create(context.Context, domain.ProjectID, domain.Task) error
}, columnID domain.ColumnID) {
	ctx := context.Background()

	// Create a test task for comments
	taskID := domain.NewTaskID()
	task := domain.Task{
		ID:            taskID,
		ColumnID:      columnID,
		Title:         "Test Task for Comments",
		Summary:       "Test task summary",
		Priority:      domain.PriorityMedium,
		PriorityScore: 200,
		CreatedAt:     time.Now(),
	}
	err := taskRepo.Create(ctx, projectID, task)
	require.NoError(t, err, "Failed to create test task for comments")

	t.Run("Contract: Create stores comment and FindByID retrieves it", func(t *testing.T) {
		comment := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     taskID,
			AuthorRole: "developer",
			AuthorName: "Test Agent",
			AuthorType: domain.AuthorTypeAgent,
			Content:    "This is a test comment",
			CreatedAt:  time.Now(),
		}

		err := repo.Create(ctx, projectID, comment)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, comment.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved, "Retrieved comment must not be nil")
		assert.Equal(t, comment.ID, retrieved.ID)
		assert.Equal(t, comment.TaskID, retrieved.TaskID)
		assert.Equal(t, comment.AuthorRole, retrieved.AuthorRole)
		assert.Equal(t, comment.AuthorName, retrieved.AuthorName)
		assert.Equal(t, comment.AuthorType, retrieved.AuthorType)
		assert.Equal(t, comment.Content, retrieved.Content)
		assert.Nil(t, retrieved.EditedAt, "EditedAt should be nil for new comment")
	})

	t.Run("Contract: FindByID returns error for non-existent comment", func(t *testing.T) {
		nonExistentID := domain.NewCommentID()
		_, err := repo.FindByID(ctx, projectID, nonExistentID)
		assert.Error(t, err, "FindByID should return error for non-existent comment")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrCommentNotFound)
	})

	t.Run("Contract: List returns comments ordered by created_at ASC", func(t *testing.T) {
		// Create multiple comments with different timestamps
		now := time.Now()
		comments := []domain.Comment{
			{
				ID:         domain.NewCommentID(),
				TaskID:     taskID,
				AuthorRole: "developer",
				AuthorName: "Agent 1",
				AuthorType: domain.AuthorTypeAgent,
				Content:    "First comment",
				CreatedAt:  now.Add(-2 * time.Hour),
			},
			{
				ID:         domain.NewCommentID(),
				TaskID:     taskID,
				AuthorRole: "human",
				AuthorName: "User 1",
				AuthorType: domain.AuthorTypeHuman,
				Content:    "Second comment",
				CreatedAt:  now.Add(-1 * time.Hour),
			},
			{
				ID:         domain.NewCommentID(),
				TaskID:     taskID,
				AuthorRole: "developer",
				AuthorName: "Agent 2",
				AuthorType: domain.AuthorTypeAgent,
				Content:    "Third comment",
				CreatedAt:  now,
			},
		}

		for _, comment := range comments {
			err := repo.Create(ctx, projectID, comment)
			require.NoError(t, err, "Create should succeed")
		}

		retrieved, err := repo.List(ctx, projectID, taskID, 0, 0)
		require.NoError(t, err, "List should succeed")
		require.GreaterOrEqual(t, len(retrieved), 3, "Should return at least 3 comments")

		// Find our test comments
		var testComments []domain.Comment
		for _, r := range retrieved {
			for _, c := range comments {
				if r.ID == c.ID {
					testComments = append(testComments, r)
					break
				}
			}
		}

		require.Len(t, testComments, 3, "Should find all test comments")

		// Verify ordering by created_at ASC
		assert.Equal(t, "First comment", testComments[0].Content)
		assert.Equal(t, "Second comment", testComments[1].Content)
		assert.Equal(t, "Third comment", testComments[2].Content)
	})

	t.Run("Contract: Update modifies comment content and sets edited_at", func(t *testing.T) {
		comment := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     taskID,
			AuthorRole: "developer",
			AuthorName: "Test Agent",
			AuthorType: domain.AuthorTypeAgent,
			Content:    "Original content",
			CreatedAt:  time.Now(),
		}

		err := repo.Create(ctx, projectID, comment)
		require.NoError(t, err, "Create should succeed")

		// Update the comment
		editedAt := time.Now()
		comment.Content = "Updated content"
		comment.EditedAt = &editedAt

		err = repo.Update(ctx, projectID, comment)
		require.NoError(t, err, "Update should succeed")

		retrieved, err := repo.FindByID(ctx, projectID, comment.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, "Updated content", retrieved.Content)
		assert.NotNil(t, retrieved.EditedAt, "EditedAt should be set after update")
	})

	t.Run("Contract: Update returns error for non-existent comment", func(t *testing.T) {
		nonExistentComment := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     taskID,
			AuthorRole: "developer",
			AuthorName: "Test Agent",
			AuthorType: domain.AuthorTypeAgent,
			Content:    "Non-existent",
			CreatedAt:  time.Now(),
		}

		err := repo.Update(ctx, projectID, nonExistentComment)
		assert.Error(t, err, "Update should return error for non-existent comment")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrCommentNotFound)
	})

	t.Run("Contract: Delete removes comment", func(t *testing.T) {
		comment := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     taskID,
			AuthorRole: "developer",
			AuthorName: "Test Agent",
			AuthorType: domain.AuthorTypeAgent,
			Content:    "Delete me",
			CreatedAt:  time.Now(),
		}

		err := repo.Create(ctx, projectID, comment)
		require.NoError(t, err, "Create should succeed")

		err = repo.Delete(ctx, projectID, comment.ID)
		require.NoError(t, err, "Delete should succeed")

		_, err = repo.FindByID(ctx, projectID, comment.ID)
		assert.Error(t, err, "FindByID should return error for deleted comment")
		assert.ErrorIs(t, err, domain.ErrCommentNotFound)
	})

	t.Run("Contract: Delete returns error for non-existent comment", func(t *testing.T) {
		nonExistentID := domain.NewCommentID()
		err := repo.Delete(ctx, projectID, nonExistentID)
		assert.Error(t, err, "Delete should return error for non-existent comment")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrCommentNotFound)
	})

	t.Run("Contract: Count returns correct number of comments for task", func(t *testing.T) {
		// Create a new task for this test
		newTaskID := domain.NewTaskID()
		newTask := domain.Task{
			ID:            newTaskID,
			ColumnID:      columnID,
			Title:         "Task for Count test",
			Summary:       "Count test task",
			Priority:      domain.PriorityLow,
			PriorityScore: 100,
			CreatedAt:     time.Now(),
		}
		err := taskRepo.Create(ctx, projectID, newTask)
		require.NoError(t, err, "Failed to create task for Count test")

		// Create multiple comments
		comment1 := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     newTaskID,
			AuthorRole: "developer",
			AuthorName: "Agent 1",
			AuthorType: domain.AuthorTypeAgent,
			Content:    "Count comment 1",
			CreatedAt:  time.Now(),
		}

		comment2 := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     newTaskID,
			AuthorRole: "human",
			AuthorName: "User 1",
			AuthorType: domain.AuthorTypeHuman,
			Content:    "Count comment 2",
			CreatedAt:  time.Now(),
		}

		err = repo.Create(ctx, projectID, comment1)
		require.NoError(t, err, "Create should succeed")
		err = repo.Create(ctx, projectID, comment2)
		require.NoError(t, err, "Create should succeed")

		count, err := repo.Count(ctx, projectID, newTaskID)
		require.NoError(t, err, "Count should succeed")
		assert.Equal(t, 2, count, "Should count exactly 2 comments")
	})

	t.Run("Contract: Count returns 0 for task with no comments", func(t *testing.T) {
		emptyTaskID := domain.NewTaskID()
		count, err := repo.Count(ctx, projectID, emptyTaskID)
		require.NoError(t, err, "Count should succeed")
		assert.Equal(t, 0, count, "Should return 0 for task with no comments")
	})

	t.Run("Contract: IsLastComment returns true for last comment", func(t *testing.T) {
		// Create a new task for this test
		newTaskID := domain.NewTaskID()
		newTask := domain.Task{
			ID:            newTaskID,
			ColumnID:      columnID,
			Title:         "Task for IsLastComment test",
			Summary:       "IsLastComment test task",
			Priority:      domain.PriorityLow,
			PriorityScore: 100,
			CreatedAt:     time.Now(),
		}
		err := taskRepo.Create(ctx, projectID, newTask)
		require.NoError(t, err, "Failed to create task for IsLastComment test")

		// Create multiple comments
		now := time.Now()
		comment1 := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     newTaskID,
			AuthorRole: "developer",
			AuthorName: "Agent 1",
			AuthorType: domain.AuthorTypeAgent,
			Content:    "First comment",
			CreatedAt:  now.Add(-1 * time.Hour),
		}

		comment2 := domain.Comment{
			ID:         domain.NewCommentID(),
			TaskID:     newTaskID,
			AuthorRole: "developer",
			AuthorName: "Agent 2",
			AuthorType: domain.AuthorTypeAgent,
			Content:    "Last comment",
			CreatedAt:  now,
		}

		err = repo.Create(ctx, projectID, comment1)
		require.NoError(t, err, "Create should succeed")
		err = repo.Create(ctx, projectID, comment2)
		require.NoError(t, err, "Create should succeed")

		isLast, err := repo.IsLastComment(ctx, projectID, comment2.ID)
		require.NoError(t, err, "IsLastComment should succeed")
		assert.True(t, isLast, "comment2 should be the last comment")

		isLast, err = repo.IsLastComment(ctx, projectID, comment1.ID)
		require.NoError(t, err, "IsLastComment should succeed")
		assert.False(t, isLast, "comment1 should not be the last comment")
	})

	t.Run("Contract: IsLastComment returns error for non-existent comment", func(t *testing.T) {
		nonExistentID := domain.NewCommentID()
		_, err := repo.IsLastComment(ctx, projectID, nonExistentID)
		assert.Error(t, err, "IsLastComment should return error for non-existent comment")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrCommentNotFound)
	})
}
