package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicColumn converts domain.Column to pkgkanban.ColumnResponse
func ToPublicColumn(column domain.Column) pkgkanban.ColumnResponse {
	return pkgkanban.ColumnResponse{
		ID:        string(column.ID),
		Slug:      string(column.Slug),
		Name:      column.Name,
		Position:  column.Position,
		WIPLimit:  column.WIPLimit,
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
