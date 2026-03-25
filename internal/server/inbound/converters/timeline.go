package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicTimelineEntry converts domain.TimelineEntry to pkgserver.TimelineEntryResponse
func ToPublicTimelineEntry(entry domain.TimelineEntry) pkgserver.TimelineEntryResponse {
	return pkgserver.TimelineEntryResponse{
		Date:           entry.Date,
		TasksCreated:   entry.TasksCreated,
		TasksCompleted: entry.TasksCompleted,
	}
}

// ToPublicTimeline converts []domain.TimelineEntry to []pkgserver.TimelineEntryResponse
func ToPublicTimeline(entries []domain.TimelineEntry) []pkgserver.TimelineEntryResponse {
	return MapSlice(entries, ToPublicTimelineEntry)
}
