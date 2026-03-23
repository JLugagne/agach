package columnstest

import (
	"context"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockColumnRepository is a function-based mock implementation of the ColumnRepository interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockColumnRepository{
//		FindByIDFunc: func(ctx context.Context, projectID domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
//			return &domain.Column{ID: id, Slug: domain.ColumnTodo, Name: "To Do"}, nil
//		},
//	}
type MockColumnRepository struct {
	FindByIDFunc       func(ctx context.Context, projectID domain.ProjectID, id domain.ColumnID) (*domain.Column, error)
	FindBySlugFunc     func(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error)
	ListFunc           func(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error)
	EnsureBacklogFunc func(ctx context.Context, projectID domain.ProjectID) (*domain.Column, error)
}

func (m *MockColumnRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
	if m.FindByIDFunc == nil {
		return nil, nil
	}
	return m.FindByIDFunc(ctx, projectID, id)
}

func (m *MockColumnRepository) FindBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	if m.FindBySlugFunc == nil {
		panic("called not defined FindBySlugFunc")
	}
	return m.FindBySlugFunc(ctx, projectID, slug)
}

func (m *MockColumnRepository) List(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, projectID)
}

func (m *MockColumnRepository) EnsureBacklog(ctx context.Context, projectID domain.ProjectID) (*domain.Column, error) {
	if m.EnsureBacklogFunc == nil {
		panic("called not defined EnsureBacklogFunc")
	}
	return m.EnsureBacklogFunc(ctx, projectID)
}

// ColumnsContractTesting runs all contract tests for a ColumnRepository implementation.
// Use this function to verify that your implementation adheres to the ColumnRepository contract.
//
// Parameters:
//   - t: testing.T instance
//   - repo: the ColumnRepository implementation to test
//   - projectID: a valid project ID to use for testing (columns should be initialized for this project)
//
// Example usage in implementation tests:
//
//	func TestSQLiteColumnRepository(t *testing.T) {
//		repo, projectID := setupTestRepo(t)
//		defer cleanupTestRepo(t, repo)
//		columnstest.ColumnsContractTesting(t, repo, projectID)
//	}
func ColumnsContractTesting(t *testing.T, repo columns.ColumnRepository, projectID domain.ProjectID) {
	ctx := context.Background()

	t.Run("Contract: List returns all 4 fixed columns ordered by position", func(t *testing.T) {
		retrieved, err := repo.List(ctx, projectID)
		require.NoError(t, err, "List should succeed")
		require.Len(t, retrieved, 4, "Should return exactly 4 columns")

		// Verify ordering by position
		assert.Equal(t, domain.ColumnTodo, retrieved[0].Slug, "First column should be todo")
		assert.Equal(t, 0, retrieved[0].Position)

		assert.Equal(t, domain.ColumnInProgress, retrieved[1].Slug, "Second column should be in_progress")
		assert.Equal(t, 1, retrieved[1].Position)

		assert.Equal(t, domain.ColumnDone, retrieved[2].Slug, "Third column should be done")
		assert.Equal(t, 2, retrieved[2].Position)

		assert.Equal(t, domain.ColumnBlocked, retrieved[3].Slug, "Fourth column should be blocked")
		assert.Equal(t, 3, retrieved[3].Position)
	})

	t.Run("Contract: FindBySlug retrieves todo column", func(t *testing.T) {
		column, err := repo.FindBySlug(ctx, projectID, domain.ColumnTodo)
		require.NoError(t, err, "FindBySlug should succeed for todo")
		require.NotNil(t, column, "Retrieved column must not be nil")
		assert.Equal(t, domain.ColumnTodo, column.Slug)
		assert.Equal(t, "To Do", column.Name)
		assert.Equal(t, 0, column.Position)
	})

	t.Run("Contract: FindBySlug retrieves in_progress column", func(t *testing.T) {
		column, err := repo.FindBySlug(ctx, projectID, domain.ColumnInProgress)
		require.NoError(t, err, "FindBySlug should succeed for in_progress")
		require.NotNil(t, column, "Retrieved column must not be nil")
		assert.Equal(t, domain.ColumnInProgress, column.Slug)
		assert.Equal(t, "In Progress", column.Name)
		assert.Equal(t, 1, column.Position)
	})

	t.Run("Contract: FindBySlug retrieves done column", func(t *testing.T) {
		column, err := repo.FindBySlug(ctx, projectID, domain.ColumnDone)
		require.NoError(t, err, "FindBySlug should succeed for done")
		require.NotNil(t, column, "Retrieved column must not be nil")
		assert.Equal(t, domain.ColumnDone, column.Slug)
		assert.Equal(t, "Done", column.Name)
		assert.Equal(t, 2, column.Position)
	})

	t.Run("Contract: FindBySlug retrieves blocked column", func(t *testing.T) {
		column, err := repo.FindBySlug(ctx, projectID, domain.ColumnBlocked)
		require.NoError(t, err, "FindBySlug should succeed for blocked")
		require.NotNil(t, column, "Retrieved column must not be nil")
		assert.Equal(t, domain.ColumnBlocked, column.Slug)
		assert.Equal(t, "Blocked", column.Name)
		assert.Equal(t, 3, column.Position)
	})

	t.Run("Contract: FindBySlug returns error for invalid slug", func(t *testing.T) {
		_, err := repo.FindBySlug(ctx, projectID, domain.ColumnSlug("invalid-slug"))
		assert.Error(t, err, "FindBySlug should return error for invalid slug")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrColumnNotFound)
	})

	t.Run("Contract: FindByID retrieves column by ID", func(t *testing.T) {
		// First get column by slug to get its ID
		todoColumn, err := repo.FindBySlug(ctx, projectID, domain.ColumnTodo)
		require.NoError(t, err, "FindBySlug should succeed")

		// Then retrieve by ID
		column, err := repo.FindByID(ctx, projectID, todoColumn.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, column, "Retrieved column must not be nil")
		assert.Equal(t, todoColumn.ID, column.ID)
		assert.Equal(t, domain.ColumnTodo, column.Slug)
	})

	t.Run("Contract: FindByID returns error for non-existent ID", func(t *testing.T) {
		nonExistentID := domain.NewColumnID()
		_, err := repo.FindByID(ctx, projectID, nonExistentID)
		assert.Error(t, err, "FindByID should return error for non-existent ID")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrColumnNotFound)
	})

	t.Run("Contract: FindByID returns error for different project", func(t *testing.T) {
		// Get column ID from current project
		todoColumn, err := repo.FindBySlug(ctx, projectID, domain.ColumnTodo)
		require.NoError(t, err, "FindBySlug should succeed")

		// Try to find it using a different project ID (non-existent project)
		differentProjectID := domain.NewProjectID()
		_, err = repo.FindByID(ctx, differentProjectID, todoColumn.ID)
		assert.Error(t, err, "FindByID should return error when using different project ID")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		// Should return ErrProjectNotFound since the project doesn't exist
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)
	})
}
