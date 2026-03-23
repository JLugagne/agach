package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

var validColumnSlugs = map[domain.ColumnSlug]bool{
	domain.ColumnBacklog:    true,
	domain.ColumnTodo:       true,
	domain.ColumnInProgress: true,
	domain.ColumnDone:       true,
	domain.ColumnBlocked:    true,
}

// ToPublicColumn converts domain.Column to pkgkanban.ColumnResponse
func ToPublicColumn(column domain.Column) pkgkanban.ColumnResponse {
	slug := string(column.Slug)
	if !validColumnSlugs[column.Slug] {
		slug = ""
	}
	return pkgkanban.ColumnResponse{
		ID:        string(column.ID),
		Slug:      slug,
		Name:      column.Name,
		Position:  column.Position,
		CreatedAt: column.CreatedAt,
	}
}

// ToPublicColumns converts []domain.Column to []pkgkanban.ColumnResponse
func ToPublicColumns(columns []domain.Column) []pkgkanban.ColumnResponse {
	result := make([]pkgkanban.ColumnResponse, len(columns))
	for i, c := range columns {
		result[i] = ToPublicColumn(c)
	}
	return result
}
