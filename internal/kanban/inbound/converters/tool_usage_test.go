package converters_test

import (
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/stretchr/testify/assert"
)

func TestToPublicToolUsageStat(t *testing.T) {
	t.Run("Converts stat with last executed at", func(t *testing.T) {
		now := time.Now()
		stat := domain.ToolUsageStat{
			ToolName:       "get_board",
			ExecutionCount: 42,
			LastExecutedAt: &now,
		}

		result := converters.ToPublicToolUsageStat(stat)

		assert.Equal(t, "get_board", result.ToolName)
		assert.Equal(t, 42, result.ExecutionCount)
		assert.NotNil(t, result.LastExecutedAt)
		assert.Equal(t, now, *result.LastExecutedAt)
	})

	t.Run("Converts stat with nil last executed at", func(t *testing.T) {
		stat := domain.ToolUsageStat{
			ToolName:       "create_task",
			ExecutionCount: 0,
			LastExecutedAt: nil,
		}

		result := converters.ToPublicToolUsageStat(stat)

		assert.Equal(t, "create_task", result.ToolName)
		assert.Equal(t, 0, result.ExecutionCount)
		assert.Nil(t, result.LastExecutedAt)
	})
}

func TestToPublicToolUsageStats(t *testing.T) {
	t.Run("Converts empty slice", func(t *testing.T) {
		stats := []domain.ToolUsageStat{}

		result := converters.ToPublicToolUsageStats(stats)

		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})

	t.Run("Converts multiple stats", func(t *testing.T) {
		now := time.Now()
		stats := []domain.ToolUsageStat{
			{
				ToolName:       "get_board",
				ExecutionCount: 10,
				LastExecutedAt: &now,
			},
			{
				ToolName:       "move_task",
				ExecutionCount: 5,
				LastExecutedAt: nil,
			},
		}

		result := converters.ToPublicToolUsageStats(stats)

		assert.Len(t, result, 2)
		assert.Equal(t, "get_board", result[0].ToolName)
		assert.Equal(t, 10, result[0].ExecutionCount)
		assert.NotNil(t, result[0].LastExecutedAt)
		assert.Equal(t, "move_task", result[1].ToolName)
		assert.Equal(t, 5, result[1].ExecutionCount)
		assert.Nil(t, result[1].LastExecutedAt)
	})
}
