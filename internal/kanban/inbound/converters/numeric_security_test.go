package converters_test

// Security tests for numeric value handling across converters.
//
// Vulnerability 6 (RED)  — float64 fields in ColdStartStatResponse can carry
//   math.NaN() or math.Inf(), which are not valid JSON values. Go's encoding/json
//   marshal fails with an UnsupportedValueError for these, breaking downstream
//   serialisation silently or causing a 500 error. The converters pass these values
//   through without sanitisation.
//   cold_start_stats.go: AvgInputTokens, AvgOutputTokens, AvgCacheReadTokens are
//   copied directly from domain.RoleColdStartStat without bounds checking.
//
// Vulnerability 6 (GREEN) — ToPublicColdStartStat sanitises non-finite float64
//   values (NaN, +Inf, -Inf) by replacing them with 0.0, ensuring the response
//   is always JSON-serialisable.
//
// Vulnerability 7 (RED)  — Negative token counts (int fields) are passed through
//   without validation in tasks.go and cold_start_stats.go. Token counts are
//   semantically non-negative; negative values indicate corrupted or malicious input
//   but are silently propagated to public responses.
//
// Vulnerability 7 (GREEN) — ToPublicTask and ToPublicColdStartStat normalise
//   negative token counts to 0.

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Vulnerability 6: non-finite float64 values in ColdStartStatResponse
// ---------------------------------------------------------------------------

// TestToPublicColdStartStat_RED_NaNPropagates demonstrates that math.NaN() stored
// in domain.RoleColdStartStat propagates to ColdStartStatResponse, which then
// causes json.Marshal to fail with an UnsupportedValueError.
//
// This test is expected to FAIL against the current implementation (red test):
// the NaN value is preserved and json.Marshal returns an error.
func TestToPublicColdStartStat_RED_NaNPropagates(t *testing.T) {
	stat := domain.RoleColdStartStat{
		AssignedRole:       "backend",
		Count:              1,
		MinInputTokens:     100,
		MaxInputTokens:     200,
		AvgInputTokens:     math.NaN(),  // crafted NaN
		MinOutputTokens:    50,
		MaxOutputTokens:    100,
		AvgOutputTokens:    math.Inf(1), // crafted +Inf
		MinCacheReadTokens: 0,
		MaxCacheReadTokens: 0,
		AvgCacheReadTokens: math.Inf(-1), // crafted -Inf
	}

	result := converters.ToPublicColdStartStat(stat)

	// RED assertion: after a fix, the result should be JSON-serialisable.
	// Currently NaN/Inf propagate and json.Marshal fails, so this assertion fails.
	_, err := json.Marshal(result)
	assert.NoError(t, err,
		"ColdStartStatResponse with NaN/Inf must be JSON-serialisable after sanitisation")
}

// TestToPublicColdStartStat_GREEN_FiniteValuesSerialise verifies that finite
// float64 values in the domain struct serialise correctly.
func TestToPublicColdStartStat_GREEN_FiniteValuesSerialise(t *testing.T) {
	stat := domain.RoleColdStartStat{
		AssignedRole:       "frontend",
		Count:              5,
		MinInputTokens:     100,
		MaxInputTokens:     500,
		AvgInputTokens:     300.5,
		MinOutputTokens:    50,
		MaxOutputTokens:    250,
		AvgOutputTokens:    150.0,
		MinCacheReadTokens: 0,
		MaxCacheReadTokens: 1000,
		AvgCacheReadTokens: 500.0,
	}

	result := converters.ToPublicColdStartStat(stat)

	data, err := json.Marshal(result)
	require.NoError(t, err, "finite float64 values must serialise without error")
	assert.NotEmpty(t, data)
	assert.Equal(t, 300.5, result.AvgInputTokens)
	assert.Equal(t, 150.0, result.AvgOutputTokens)
	assert.Equal(t, 500.0, result.AvgCacheReadTokens)
}

// TestToPublicColdStartStat_RED_NaNNormalisedToZero is the companion RED assertion
// checking the actual normalised value after a fix is applied:
// NaN/Inf must become 0.0, not any other sentinel.
//
// This test is expected to FAIL against the current implementation (red test):
// the function returns NaN, not 0.0.
func TestToPublicColdStartStat_RED_NaNNormalisedToZero(t *testing.T) {
	stat := domain.RoleColdStartStat{
		AvgInputTokens:     math.NaN(),
		AvgOutputTokens:    math.Inf(1),
		AvgCacheReadTokens: math.Inf(-1),
	}

	result := converters.ToPublicColdStartStat(stat)

	// RED assertion: these should all be 0.0 after sanitisation.
	assert.Equal(t, 0.0, result.AvgInputTokens,
		"NaN AvgInputTokens must be normalised to 0.0")
	assert.Equal(t, 0.0, result.AvgOutputTokens,
		"+Inf AvgOutputTokens must be normalised to 0.0")
	assert.Equal(t, 0.0, result.AvgCacheReadTokens,
		"-Inf AvgCacheReadTokens must be normalised to 0.0")
}

// ---------------------------------------------------------------------------
// Vulnerability 7: negative token counts in task responses
// ---------------------------------------------------------------------------

// TestToPublicTask_RED_NegativeTokenCountsPropagates demonstrates that negative
// token counts stored in a domain.Task pass through ToPublicTask unchecked.
// Token counts are semantically non-negative; negative values signal either
// corrupted data or a malicious manipulation.
//
// This test is expected to FAIL against the current implementation (red test):
// the function preserves negative values.
func TestToPublicTask_RED_NegativeTokenCountsPropagates(t *testing.T) {
	now := time.Now()
	task := domain.Task{
		ID:               domain.TaskID("task-neg"),
		ColumnID:         domain.ColumnID("col-1"),
		Title:            "negative tokens",
		Summary:          "summary",
		InputTokens:      -1,
		OutputTokens:     -99999,
		CacheReadTokens:  -1,
		CacheWriteTokens: -1,
		DurationSeconds:  -500,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	result := converters.ToPublicTask(task)

	// RED assertions: after a fix, negative counts must be clamped to 0.
	assert.GreaterOrEqual(t, result.InputTokens, 0,
		"InputTokens must not be negative in public response")
	assert.GreaterOrEqual(t, result.OutputTokens, 0,
		"OutputTokens must not be negative in public response")
	assert.GreaterOrEqual(t, result.CacheReadTokens, 0,
		"CacheReadTokens must not be negative in public response")
	assert.GreaterOrEqual(t, result.CacheWriteTokens, 0,
		"CacheWriteTokens must not be negative in public response")
	assert.GreaterOrEqual(t, result.DurationSeconds, 0,
		"DurationSeconds must not be negative in public response")
}

// TestToPublicTask_GREEN_ZeroAndPositiveTokenCountsPassThrough verifies that
// zero and positive token counts are preserved correctly.
func TestToPublicTask_GREEN_ZeroAndPositiveTokenCountsPassThrough(t *testing.T) {
	now := time.Now()
	task := domain.Task{
		ID:               domain.TaskID("task-pos"),
		ColumnID:         domain.ColumnID("col-1"),
		Title:            "positive tokens",
		Summary:          "summary",
		InputTokens:      1000,
		OutputTokens:     500,
		CacheReadTokens:  200,
		CacheWriteTokens: 100,
		DurationSeconds:  3600,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	result := converters.ToPublicTask(task)

	assert.Equal(t, 1000, result.InputTokens)
	assert.Equal(t, 500, result.OutputTokens)
	assert.Equal(t, 200, result.CacheReadTokens)
	assert.Equal(t, 100, result.CacheWriteTokens)
	assert.Equal(t, 3600, result.DurationSeconds)
}

// TestToPublicColdStartStat_RED_NegativeMinMaxCounts demonstrates that negative
// min/max token count integers propagate from domain struct to public response.
//
// This test is expected to FAIL against the current implementation (red test).
func TestToPublicColdStartStat_RED_NegativeMinMaxCounts(t *testing.T) {
	stat := domain.RoleColdStartStat{
		AssignedRole:       "backend",
		Count:              3,
		MinInputTokens:     -100,
		MaxInputTokens:     -50,
		AvgInputTokens:     -75.0,
		MinOutputTokens:    -200,
		MaxOutputTokens:    -100,
		AvgOutputTokens:    -150.0,
		MinCacheReadTokens: -10,
		MaxCacheReadTokens: -5,
		AvgCacheReadTokens: -7.5,
	}

	result := converters.ToPublicColdStartStat(stat)

	// RED assertions: negative token counts must be clamped to 0.
	assert.GreaterOrEqual(t, result.MinInputTokens, 0,
		"MinInputTokens must not be negative")
	assert.GreaterOrEqual(t, result.MaxInputTokens, 0,
		"MaxInputTokens must not be negative")
	assert.GreaterOrEqual(t, result.AvgInputTokens, 0.0,
		"AvgInputTokens must not be negative")
	assert.GreaterOrEqual(t, result.MinOutputTokens, 0,
		"MinOutputTokens must not be negative")
	assert.GreaterOrEqual(t, result.MaxOutputTokens, 0,
		"MaxOutputTokens must not be negative")
	assert.GreaterOrEqual(t, result.AvgOutputTokens, 0.0,
		"AvgOutputTokens must not be negative")
	assert.GreaterOrEqual(t, result.MinCacheReadTokens, 0,
		"MinCacheReadTokens must not be negative")
	assert.GreaterOrEqual(t, result.MaxCacheReadTokens, 0,
		"MaxCacheReadTokens must not be negative")
	assert.GreaterOrEqual(t, result.AvgCacheReadTokens, 0.0,
		"AvgCacheReadTokens must not be negative")
}

// TestToPublicColdStartStat_GREEN_ZeroCounts verifies that all-zero counts
// produce a valid, JSON-serialisable response.
func TestToPublicColdStartStat_GREEN_ZeroCounts(t *testing.T) {
	stat := domain.RoleColdStartStat{
		AssignedRole: "ops",
		Count:        0,
	}

	result := converters.ToPublicColdStartStat(stat)

	_, err := json.Marshal(result)
	require.NoError(t, err, "zero-count stat must serialise without error")
	assert.Equal(t, "ops", result.AssignedRole)
	assert.Equal(t, 0, result.Count)
}
