package converters

import (
	"math"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// safeFloat normalises non-finite and negative float64 values to 0.0.
func safeFloat(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
		return 0.0
	}
	return f
}

// clampInt normalises negative int values to 0.
func clampInt(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// ToPublicColdStartStat converts domain.RoleColdStartStat to pkgkanban.ColdStartStatResponse
func ToPublicColdStartStat(stat domain.RoleColdStartStat) pkgkanban.ColdStartStatResponse {
	return pkgkanban.ColdStartStatResponse{
		AssignedRole:       stat.AssignedRole,
		Count:              stat.Count,
		MinInputTokens:     clampInt(stat.MinInputTokens),
		MaxInputTokens:     clampInt(stat.MaxInputTokens),
		AvgInputTokens:     safeFloat(stat.AvgInputTokens),
		MinOutputTokens:    clampInt(stat.MinOutputTokens),
		MaxOutputTokens:    clampInt(stat.MaxOutputTokens),
		AvgOutputTokens:    safeFloat(stat.AvgOutputTokens),
		MinCacheReadTokens: clampInt(stat.MinCacheReadTokens),
		MaxCacheReadTokens: clampInt(stat.MaxCacheReadTokens),
		AvgCacheReadTokens: safeFloat(stat.AvgCacheReadTokens),
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
