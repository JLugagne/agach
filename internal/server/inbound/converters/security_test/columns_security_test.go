package security_test

// Security tests for columns.go converters.
//
// Vulnerability 5 — ToPublicColumn normalises unrecognised ColumnSlug values
//   to an empty string (or another safe sentinel) rather than propagating
//   attacker-controlled input. Values outside the five fixed slugs must not
//   appear in public API responses.

import (
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/stretchr/testify/assert"
)

// TestToPublicColumn_InvalidSlugPropagates verifies that unrecognised ColumnSlug
// values stored in the domain struct are normalised by ToPublicColumn rather than
// propagated to public API responses.
func TestToPublicColumn_InvalidSlugPropagates(t *testing.T) {
	invalidSlugs := []domain.ColumnSlug{
		"nonexistent",
		"../../etc/passwd",
		"'; DROP TABLE columns; --",
		"<img src=x onerror=alert(1)>",
		"TODO",        // wrong case
		"In_Progress", // wrong case
	}

	validSlugs := map[string]bool{
		"backlog":     true,
		"todo":        true,
		"in_progress": true,
		"done":        true,
		"blocked":     true,
	}

	for _, slug := range invalidSlugs {
		col := domain.Column{
			ID:       domain.ColumnID("col-1"),
			Slug:     slug,
			Name:     "Test Column",
			Position: 0,
		}
		result := converters.ToPublicColumn(col)

		assert.True(t, validSlugs[result.Slug] || result.Slug == "",
			"ColumnSlug %q must be normalised; got %q in public response", slug, result.Slug)
	}
}

// TestToPublicColumn_GREEN_ValidSlugsPassThrough verifies that all five fixed
// column slugs are converted correctly before and after any fix.
func TestToPublicColumn_GREEN_ValidSlugsPassThrough(t *testing.T) {
	cases := []struct {
		slug     domain.ColumnSlug
		expected string
	}{
		{domain.ColumnBacklog, "backlog"},
		{domain.ColumnTodo, "todo"},
		{domain.ColumnInProgress, "in_progress"},
		{domain.ColumnDone, "done"},
		{domain.ColumnBlocked, "blocked"},
	}

	for _, tc := range cases {
		col := domain.Column{
			ID:       domain.ColumnID("col-valid"),
			Slug:     tc.slug,
			Name:     "Valid Column",
			Position: 0,
		}
		result := converters.ToPublicColumn(col)
		assert.Equal(t, tc.expected, result.Slug,
			"valid ColumnSlug %q should convert to %q", tc.slug, tc.expected)
	}
}
