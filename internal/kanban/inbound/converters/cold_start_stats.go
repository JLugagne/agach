package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicColdStartStat converts domain.RoleColdStartStat to pkgkanban.ColdStartStatResponse
func ToPublicColdStartStat(stat domain.RoleColdStartStat) pkgkanban.ColdStartStatResponse {
	return pkgkanban.ColdStartStatResponse{
		AssignedRole:       stat.AssignedRole,
		Count:              stat.Count,
		MinInputTokens:     stat.MinInputTokens,
		MaxInputTokens:     stat.MaxInputTokens,
		AvgInputTokens:     stat.AvgInputTokens,
		MinOutputTokens:    stat.MinOutputTokens,
		MaxOutputTokens:    stat.MaxOutputTokens,
		AvgOutputTokens:    stat.AvgOutputTokens,
		MinCacheReadTokens: stat.MinCacheReadTokens,
		MaxCacheReadTokens: stat.MaxCacheReadTokens,
		AvgCacheReadTokens: stat.AvgCacheReadTokens,
	}
}

// ToPublicColdStartStats converts []domain.RoleColdStartStat to []pkgkanban.ColdStartStatResponse
func ToPublicColdStartStats(stats []domain.RoleColdStartStat) []pkgkanban.ColdStartStatResponse {
	result := make([]pkgkanban.ColdStartStatResponse, len(stats))
	for i, s := range stats {
		result[i] = ToPublicColdStartStat(s)
	}
	return result
}
