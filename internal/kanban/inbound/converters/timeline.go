package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicTimelineEntry converts domain.TimelineEntry to pkgkanban.TimelineEntryResponse
func ToPublicTimelineEntry(entry domain.TimelineEntry) pkgkanban.TimelineEntryResponse {
	return pkgkanban.TimelineEntryResponse{
		Date:           entry.Date,
		TasksCreated:   entry.TasksCreated,
		TasksCompleted: entry.TasksCompleted,
	}
}

// ToPublicTimeline converts []domain.TimelineEntry to []pkgkanban.TimelineEntryResponse
func ToPublicTimeline(entries []domain.TimelineEntry) []pkgkanban.TimelineEntryResponse {
	result := make([]pkgkanban.TimelineEntryResponse, len(entries))
	for i, e := range entries {
		result[i] = ToPublicTimelineEntry(e)
	}
	return result
}
