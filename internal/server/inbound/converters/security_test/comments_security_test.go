package security_test

// Security tests for comments.go converters.
//
// Vulnerability 4 — ToPublicComment normalises unrecognised AuthorType values to
//   a safe default ("agent") rather than leaking arbitrary strings into public
//   responses. Values outside {"agent","human"} must not appear in API output.

import (
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/stretchr/testify/assert"
)

// TestToPublicComment_InvalidAuthorTypePropagates verifies that any unrecognised
// string in domain.Comment.AuthorType is normalised by ToPublicComment rather than
// reaching the public CommentResponse unchanged.
func TestToPublicComment_InvalidAuthorTypePropagates(t *testing.T) {
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
