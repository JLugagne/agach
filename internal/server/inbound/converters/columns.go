package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

var validColumnSlugs = map[domain.ColumnSlug]bool{
	domain.ColumnBacklog:    true,
	domain.ColumnTodo:       true,
	domain.ColumnInProgress: true,
	domain.ColumnDone:       true,
	domain.ColumnBlocked:    true,
}

// ToPublicColumn converts domain.Column to pkgserver.ColumnResponse
func ToPublicColumn(column domain.Column) pkgserver.ColumnResponse {
	slug := string(column.Slug)
	if !validColumnSlugs[column.Slug] {
		slug = ""
	}
	return pkgserver.ColumnResponse{
		ID:        string(column.ID),
		Slug:      slug,
		Name:      column.Name,
		Position:  column.Position,
		CreatedAt: column.CreatedAt,
	}
}

// ToPublicColumns converts []domain.Column to []pkgserver.ColumnResponse
func ToPublicColumns(columns []domain.Column) []pkgserver.ColumnResponse {
	return MapSlice(columns, ToPublicColumn)
}
