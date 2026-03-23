package toolusage

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// ToolUsageRepository defines operations for tracking MCP tool execution counts
type ToolUsageRepository interface {
	// IncrementToolUsage increments the execution count for a tool (UPSERT)
	IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error

	// ListToolUsage returns all tool usage stats ordered by execution_count DESC
	ListToolUsage(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)
}
