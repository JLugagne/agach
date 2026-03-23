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
	result := make([]pkgserver.TimelineEntryResponse, len(entries))
	for i, e := range entries {
		result[i] = ToPublicTimelineEntry(e)
	}
	return result
}
