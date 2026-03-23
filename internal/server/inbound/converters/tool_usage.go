package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicToolUsageStat converts domain.ToolUsageStat to pkgserver.ToolUsageStatResponse
func ToPublicToolUsageStat(stat domain.ToolUsageStat) pkgserver.ToolUsageStatResponse {
	return pkgserver.ToolUsageStatResponse{
		ToolName:       stat.ToolName,
		ExecutionCount: stat.ExecutionCount,
		LastExecutedAt: stat.LastExecutedAt,
	}
}

// ToPublicToolUsageStats converts []domain.ToolUsageStat to []pkgserver.ToolUsageStatResponse
func ToPublicToolUsageStats(stats []domain.ToolUsageStat) []pkgserver.ToolUsageStatResponse {
	result := make([]pkgserver.ToolUsageStatResponse, len(stats))
	for i, s := range stats {
		result[i] = ToPublicToolUsageStat(s)
	}
	return result
}
