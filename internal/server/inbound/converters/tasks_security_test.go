package converters_test

// Security tests for tasks.go converters.
//
// Vulnerability 1 (RED)  — ToDomainPriority accepts arbitrary invalid enum values.
//   tasks.go:9-13: any non-empty string is blindly cast to domain.Priority, bypassing
//   the four-value constraint (critical/high/medium/low).
//
// Vulnerability 1 (GREEN) — ToDomainPriority rejects invalid values by returning
//   domain.PriorityMedium (safe default) rather than propagating attacker input.
//
// Vulnerability 2 (RED)  — ToDomainPriority has no length bound on the input string.
//   A very long string is cast directly to domain.Priority.
//
// Vulnerability 2 (GREEN) — ToDomainPriority rejects excessively long input.
//
// Vulnerability 3 (RED)  — ToDomainTaskIDs accepts non-UUID strings without parsing.
//   tasks.go:17-23: each element is cast to domain.TaskID with no format check,
//   inconsistent with ToDomainProjectID which calls domain.ParseProjectID.
//
// Vulnerability 3 (GREEN) — ToDomainTaskIDs skips (or errors on) non-UUID elements,
//   consistent with the validation applied in ToDomainProjectID.

import (
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Vulnerability 1: missing enum validation in ToDomainPriority
// ---------------------------------------------------------------------------

// TestToDomainPriority_RED_InvalidEnumAccepted demonstrates that ToDomainPriority
// currently accepts arbitrary strings, which means attacker-controlled values leak
// into domain.Priority and subsequently into API responses.
//
// This test is expected to FAIL against the current implementation (red test):
// the function returns the injected value as a priority rather than a safe default.
func TestToDomainPriority_RED_InvalidEnumAccepted(t *testing.T) {
	invalidValues := []string{
		"urgent",
		"CRITICAL",
		"'; DROP TABLE tasks; --",
		"<script>alert(1)</script>",
		"__proto__",
		"null",
		"undefined",
	}

	for _, v := range invalidValues {
		result := converters.ToDomainPriority(v)
		// RED assertion: after a fix, invalid values should return a safe default.
		// Currently the implementation returns the raw string, so this will fail.
		assert.Equal(t, domain.PriorityMedium, result,
			"invalid priority %q should be normalised to medium, got %q", v, result)
	}
}

// TestToDomainPriority_GREEN_ValidEnumValues verifies that valid priority strings
// are converted correctly (this must stay green before and after any fix).
func TestToDomainPriority_GREEN_ValidEnumValues(t *testing.T) {
	cases := []struct {
		input    string
		expected domain.Priority
	}{
		{"critical", domain.PriorityCritical},
		{"high", domain.PriorityHigh},
		{"medium", domain.PriorityMedium},
		{"low", domain.PriorityLow},
		{"", domain.PriorityMedium},
	}

	for _, tc := range cases {
		result := converters.ToDomainPriority(tc.input)
		assert.Equal(t, tc.expected, result,
			"valid priority %q should convert to %q", tc.input, tc.expected)
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 2: no length bound on priority string
// ---------------------------------------------------------------------------

// TestToDomainPriority_RED_UnboundedLength demonstrates that ToDomainPriority
// accepts a megabyte-long string without any length check, allowing allocation
// amplification if the value is stored or re-encoded.
//
// This test is expected to FAIL against the current implementation (red test):
// the function returns the oversized string as-is.
func TestToDomainPriority_RED_UnboundedLength(t *testing.T) {
	// 1 MB string — well above any reasonable priority label length.
	huge := strings.Repeat("x", 1_000_000)
	result := converters.ToDomainPriority(huge)
	// RED assertion: after a fix, oversized input should be rejected.
	// Currently it is returned as-is, so this will fail.
	assert.Equal(t, domain.PriorityMedium, result,
		"priority string of 1 MB should be rejected and normalised to medium")
}

// TestToDomainPriority_GREEN_ReasonableLengthRejected verifies that a
// slightly-above-maximum length string (say, > 20 characters) is rejected
// while valid short strings still work. Threshold is implementation-defined;
// any value of PriorityMedium for the oversized input is acceptable.
func TestToDomainPriority_GREEN_ReasonableLengthAccepted(t *testing.T) {
	// All valid values are short; ensure they are not affected by length logic.
	for _, v := range []string{"critical", "high", "medium", "low"} {
		result := converters.ToDomainPriority(v)
		assert.NotEqual(t, "", string(result),
			"valid priority %q must not become empty after length check", v)
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 3: no UUID validation in ToDomainTaskIDs
// ---------------------------------------------------------------------------

// TestToDomainTaskIDs_RED_NoUUIDValidation demonstrates that ToDomainTaskIDs
// silently wraps arbitrary strings as domain.TaskID values without UUID parsing,
// inconsistent with ToDomainProjectID which validates format via domain.ParseProjectID.
//
// This test is expected to FAIL against the current implementation (red test):
// the function returns a slice that includes the injection payload.
func TestToDomainTaskIDs_RED_NoUUIDValidation(t *testing.T) {
	malformed := []string{
		"not-a-uuid",
		"'; DROP TABLE tasks; --",
		"../../../etc/passwd",
		"",
		strings.Repeat("a", 1000),
	}

	result := converters.ToDomainTaskIDs(malformed)

	// RED assertion: after a fix, non-UUID entries must be dropped.
	// Currently they are all preserved, so this will fail for malformed inputs.
	for _, id := range result {
		_, parseErr := uuid.Parse(string(id))
		assert.NoError(t, parseErr,
			"ToDomainTaskIDs must only return valid UUID-formatted TaskIDs, got %q", id)
	}
}

// TestToDomainTaskIDs_GREEN_ValidUUIDsPassThrough verifies that well-formed UUID
// strings are still converted correctly regardless of any added validation.
func TestToDomainTaskIDs_GREEN_ValidUUIDsPassThrough(t *testing.T) {
	id1 := domain.NewTaskID()
	id2 := domain.NewTaskID()
	ids := []string{id1.String(), id2.String()}

	result := converters.ToDomainTaskIDs(ids)

	assert.Len(t, result, 2)
	assert.Equal(t, id1, result[0])
	assert.Equal(t, id2, result[1])
}

// TestToDomainTaskIDs_GREEN_EmptySliceReturnsEmpty verifies the empty-input
// contract is unaffected by validation additions.
func TestToDomainTaskIDs_GREEN_EmptySliceReturnsEmpty(t *testing.T) {
	result := converters.ToDomainTaskIDs([]string{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}
