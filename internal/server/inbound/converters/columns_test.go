package converters_test

import (
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/stretchr/testify/assert"
)

func TestToPublicColumn(t *testing.T) {
	column := domain.Column{
		ID:       domain.ColumnID("col-123"),
		Slug:     domain.ColumnTodo,
		Name:     "To Do",
		Position: 0,
	}

	result := converters.ToPublicColumn(column)

	assert.Equal(t, "col-123", result.ID)
	assert.Equal(t, "todo", result.Slug)
	assert.Equal(t, "To Do", result.Name)
	assert.Equal(t, 0, result.Position)
}

func TestToPublicColumns(t *testing.T) {
	columns := []domain.Column{
		{ID: domain.ColumnID("col-1"), Slug: domain.ColumnTodo, Name: "To Do"},
		{ID: domain.ColumnID("col-2"), Slug: domain.ColumnDone, Name: "Done"},
	}

	result := converters.ToPublicColumns(columns)

	assert.Len(t, result, 2)
	assert.Equal(t, "col-1", result[0].ID)
	assert.Equal(t, "col-2", result[1].ID)
}
