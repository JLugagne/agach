package toolusagetest

import (
	"context"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/toolusage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockToolUsageRepository is a function-based mock implementation of the ToolUsageRepository interface.
//
// Example usage:
//
//	mock := &MockToolUsageRepository{
//		IncrementToolUsageFunc: func(ctx context.Context, projectID domain.ProjectID, toolName string) error {
//			return nil
//		},
//	}
type MockToolUsageRepository struct {
	IncrementToolUsageFunc func(ctx context.Context, projectID domain.ProjectID, toolName string) error
	ListToolUsageFunc      func(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)
}

func (m *MockToolUsageRepository) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	if m.IncrementToolUsageFunc == nil {
		panic("called not defined IncrementToolUsageFunc")
	}
	return m.IncrementToolUsageFunc(ctx, projectID, toolName)
}

func (m *MockToolUsageRepository) ListToolUsage(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	if m.ListToolUsageFunc == nil {
		panic("called not defined ListToolUsageFunc")
	}
	return m.ListToolUsageFunc(ctx, projectID)
}

// ToolUsageContractTesting runs all contract tests for a ToolUsageRepository implementation.
//
// Parameters:
//   - t: testing.T instance
//   - repo: the ToolUsageRepository implementation to test
//   - projectID: a valid project ID with an existing project database
//
// Example usage in implementation tests:
//
//	func TestSQLiteToolUsageRepository(t *testing.T) {
//		repo := setupTestRepo(t)
//		projectID, _, _, _ := setupTestProject(t, repo)
//		toolusagetest.ToolUsageContractTesting(t, repo.ToolUsage, projectID)
//	}
func ToolUsageContractTesting(t *testing.T, repo toolusage.ToolUsageRepository, projectID domain.ProjectID) {
	ctx := context.Background()

	t.Run("Contract: IncrementToolUsage creates entry on first call", func(t *testing.T) {
		err := repo.IncrementToolUsage(ctx, projectID, "test_tool_create")
		require.NoError(t, err, "IncrementToolUsage should succeed")

		stats, err := repo.ListToolUsage(ctx, projectID)
		require.NoError(t, err)

		var found *domain.ToolUsageStat
		for i := range stats {
			if stats[i].ToolName == "test_tool_create" {
				found = &stats[i]
				break
			}
		}
		require.NotNil(t, found, "Tool should exist after increment")
		assert.Equal(t, 1, found.ExecutionCount, "Execution count should be 1 after first call")
		assert.NotNil(t, found.LastExecutedAt, "LastExecutedAt should be set")
	})

	t.Run("Contract: IncrementToolUsage increments existing entry", func(t *testing.T) {
		err := repo.IncrementToolUsage(ctx, projectID, "test_tool_increment")
		require.NoError(t, err)
		err = repo.IncrementToolUsage(ctx, projectID, "test_tool_increment")
		require.NoError(t, err)
		err = repo.IncrementToolUsage(ctx, projectID, "test_tool_increment")
		require.NoError(t, err)

		stats, err := repo.ListToolUsage(ctx, projectID)
		require.NoError(t, err)

		var found *domain.ToolUsageStat
		for i := range stats {
			if stats[i].ToolName == "test_tool_increment" {
				found = &stats[i]
				break
			}
		}
		require.NotNil(t, found, "Tool should exist")
		assert.Equal(t, 3, found.ExecutionCount, "Execution count should be 3 after three calls")
	})

	t.Run("Contract: ListToolUsage returns results ordered by execution_count DESC", func(t *testing.T) {
		err := repo.IncrementToolUsage(ctx, projectID, "test_tool_low")
		require.NoError(t, err)

		for range 5 {
			err = repo.IncrementToolUsage(ctx, projectID, "test_tool_high")
			require.NoError(t, err)
		}

		stats, err := repo.ListToolUsage(ctx, projectID)
		require.NoError(t, err)

		// Find positions
		highIdx, lowIdx := -1, -1
		for i, s := range stats {
			if s.ToolName == "test_tool_high" {
				highIdx = i
			}
			if s.ToolName == "test_tool_low" {
				lowIdx = i
			}
		}
		require.NotEqual(t, -1, highIdx, "test_tool_high should exist")
		require.NotEqual(t, -1, lowIdx, "test_tool_low should exist")
		assert.Less(t, highIdx, lowIdx, "Higher count tool should appear before lower count tool")
	})
}
