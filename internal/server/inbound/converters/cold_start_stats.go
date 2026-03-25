package converters

import (
	"math"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
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

// ToPublicColdStartStat converts domain.AgentColdStartStat to pkgserver.ColdStartStatResponse
func ToPublicColdStartStat(stat domain.AgentColdStartStat) pkgserver.ColdStartStatResponse {
	return pkgserver.ColdStartStatResponse{
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

// ToPublicColdStartStats converts []domain.AgentColdStartStat to []pkgserver.ColdStartStatResponse
func ToPublicColdStartStats(stats []domain.AgentColdStartStat) []pkgserver.ColdStartStatResponse {
	return MapSlice(stats, ToPublicColdStartStat)
}
