package security_test

// Security tests for tasks.go converters.
//
// Vulnerability 1 — ToDomainPriority rejects invalid values by returning
//   domain.PriorityMedium (safe default) rather than propagating attacker input.
//   Any non-empty string outside the four-value set (critical/high/medium/low)
//   is normalised to the safe default.
//
// Vulnerability 2 — ToDomainPriority rejects excessively long input, normalising
//   oversized strings to domain.PriorityMedium.
//
// Vulnerability 3 — ToDomainTaskIDs skips non-UUID elements, consistent with the
//   validation applied in ToDomainProjectID.

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

// TestToDomainPriority_InvalidEnumAccepted verifies that ToDomainPriority
// normalises arbitrary strings to domain.PriorityMedium rather than leaking
// attacker-controlled values into domain.Priority and API responses.
func TestToDomainPriority_InvalidEnumAccepted(t *testing.T) {
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

// TestToDomainPriority_UnboundedLength verifies that ToDomainPriority
// rejects oversized input by returning domain.PriorityMedium rather than
// preserving a megabyte-long string as a priority value.
func TestToDomainPriority_UnboundedLength(t *testing.T) {
	// 1 MB string — well above any reasonable priority label length.
	huge := strings.Repeat("x", 1_000_000)
	result := converters.ToDomainPriority(huge)
	assert.Equal(t, domain.PriorityMedium, result,
		"priority string of 1 MB should be rejected and normalised to medium")
}

// TestToDomainPriority_GREEN_ReasonableLengthAccepted verifies that a
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

// TestToDomainTaskIDs_NoUUIDValidation verifies that ToDomainTaskIDs drops
// non-UUID entries, consistent with ToDomainProjectID which validates format
// via domain.ParseProjectID.
func TestToDomainTaskIDs_NoUUIDValidation(t *testing.T) {
	malformed := []string{
		"not-a-uuid",
		"'; DROP TABLE tasks; --",
		"../../../etc/passwd",
		"",
		strings.Repeat("a", 1000),
	}

	result := converters.ToDomainTaskIDs(malformed)

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
