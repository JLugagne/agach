package security_test

// Security tests for numeric value handling across converters.
//
// Vulnerability 6 — ToPublicColdStartStat sanitises non-finite float64 values
//   (NaN, +Inf, -Inf) by replacing them with 0.0, ensuring the response is always
//   JSON-serialisable. The AvgInputTokens, AvgOutputTokens, and AvgCacheReadTokens
//   fields are bounded before being written to ColdStartStatResponse.
//
// Vulnerability 7 — ToPublicTask and ToPublicColdStartStat normalise negative
//   token counts to 0. Token counts are semantically non-negative; negative values
//   indicate corrupted or malicious input and must not appear in public responses.

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Vulnerability 6: non-finite float64 values in ColdStartStatResponse
// ---------------------------------------------------------------------------

// TestToPublicColdStartStat_NaNPropagates verifies that math.NaN() and math.Inf()
// stored in domain.RoleColdStartStat are sanitised to 0.0 by ToPublicColdStartStat,
// ensuring the resulting ColdStartStatResponse is always JSON-serialisable.
func TestToPublicColdStartStat_NaNPropagates(t *testing.T) {
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

// TestToPublicColdStartStat_NaNNormalisedToZero verifies that NaN/Inf values in
// domain.RoleColdStartStat are normalised to 0.0 (not any other sentinel) by
// ToPublicColdStartStat.
func TestToPublicColdStartStat_NaNNormalisedToZero(t *testing.T) {
	stat := domain.RoleColdStartStat{
		AvgInputTokens:     math.NaN(),
		AvgOutputTokens:    math.Inf(1),
		AvgCacheReadTokens: math.Inf(-1),
	}

	result := converters.ToPublicColdStartStat(stat)

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

// TestToPublicTask_NegativeTokenCountsPropagates verifies that negative token
// counts stored in a domain.Task are clamped to 0 by ToPublicTask. Token counts
// are semantically non-negative; negative values must not appear in public responses.
func TestToPublicTask_NegativeTokenCountsPropagates(t *testing.T) {
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

// TestToPublicColdStartStat_NegativeMinMaxCounts verifies that negative min/max
// token count integers in domain.RoleColdStartStat are clamped to 0 by
// ToPublicColdStartStat rather than propagated to the public response.
func TestToPublicColdStartStat_NegativeMinMaxCounts(t *testing.T) {
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
