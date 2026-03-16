package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicToolUsageStat converts domain.ToolUsageStat to pkgkanban.ToolUsageStatResponse
func ToPublicToolUsageStat(stat domain.ToolUsageStat) pkgkanban.ToolUsageStatResponse {
	return pkgkanban.ToolUsageStatResponse{
		ToolName:       stat.ToolName,
		ExecutionCount: stat.ExecutionCount,
		LastExecutedAt: stat.LastExecutedAt,
	}
}

// ToPublicToolUsageStats converts []domain.ToolUsageStat to []pkgkanban.ToolUsageStatResponse
func ToPublicToolUsageStats(stats []domain.ToolUsageStat) []pkgkanban.ToolUsageStatResponse {
	result := make([]pkgkanban.ToolUsageStatResponse, len(stats))
	for i, s := range stats {
		result[i] = ToPublicToolUsageStat(s)
	}
	return result
}
