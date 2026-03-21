package converters_test

// Security tests for comments.go converters.
//
// Vulnerability 4 (RED)  — ToPublicComment passes AuthorType through without enum
//   validation. comments.go:15: string(comment.AuthorType) converts any stored value
//   directly to the public response. Values outside {"agent","human"} propagate.
//
// Vulnerability 4 (GREEN) — ToPublicComment normalises unrecognised AuthorType
//   values to a safe default ("agent") rather than leaking arbitrary strings.

import (
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/stretchr/testify/assert"
)

// TestToPublicComment_RED_InvalidAuthorTypePropagates demonstrates that any
// string stored in domain.Comment.AuthorType reaches the public CommentResponse
// without validation. An attacker who can influence the domain object (e.g. via
// direct DB writes or future inbound paths) can inject arbitrary values.
//
// This test is expected to FAIL against the current implementation (red test).
func TestToPublicComment_RED_InvalidAuthorTypePropagates(t *testing.T) {
	invalidAuthorTypes := []domain.AuthorType{
		"admin",
		"'; DROP TABLE comments; --",
		"<script>alert(1)</script>",
		"AGENT",
		"HUMAN",
		"system",
	}

	for _, at := range invalidAuthorTypes {
		comment := domain.Comment{
			ID:         domain.CommentID("cmt-1"),
			TaskID:     domain.TaskID("task-1"),
			AuthorRole: "backend",
			AuthorName: "agent007",
			AuthorType: at,
			Content:    "test content",
		}
		result := converters.ToPublicComment(comment)

		// RED assertion: after a fix, invalid author types should be normalised.
		// Currently the raw value is returned, so this assertion will fail.
		validTypes := map[string]bool{"agent": true, "human": true}
		assert.True(t, validTypes[result.AuthorType],
			"AuthorType %q must be normalised to a valid value, got %q", at, result.AuthorType)
	}
}

// TestToPublicComment_GREEN_ValidAuthorTypesPassThrough verifies that the two
// valid author type values are converted correctly before and after any fix.
func TestToPublicComment_GREEN_ValidAuthorTypesPassThrough(t *testing.T) {
	cases := []struct {
		authorType domain.AuthorType
		expected   string
	}{
		{domain.AuthorTypeAgent, "agent"},
		{domain.AuthorTypeHuman, "human"},
	}

	for _, tc := range cases {
		comment := domain.Comment{
			ID:         domain.CommentID("cmt-valid"),
			TaskID:     domain.TaskID("task-valid"),
			AuthorRole: "backend",
			AuthorName: "agent007",
			AuthorType: tc.authorType,
			Content:    "some content",
		}
		result := converters.ToPublicComment(comment)
		assert.Equal(t, tc.expected, result.AuthorType,
			"valid AuthorType %q should convert to %q", tc.authorType, tc.expected)
	}
}
