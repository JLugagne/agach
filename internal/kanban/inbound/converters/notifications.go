package converters

import (
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicNotification converts domain.Notification to pkgkanban.NotificationResponse
func ToPublicNotification(n domain.Notification) pkgkanban.NotificationResponse {
	var readAt *string
	if n.ReadAt != nil {
		s := n.ReadAt.Format(time.RFC3339)
		readAt = &s
	}
	return pkgkanban.NotificationResponse{
		ID:        n.ID.String(),
		ProjectID: n.ProjectID.String(),
		Severity:  string(n.Severity),
		Title:     n.Title,
		Text:      n.Text,
		LinkURL:   n.LinkURL,
		LinkText:  n.LinkText,
		LinkStyle: n.LinkStyle,
		ReadAt:    readAt,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
}

// ToPublicNotifications converts []domain.Notification to []pkgkanban.NotificationResponse
func ToPublicNotifications(ns []domain.Notification) []pkgkanban.NotificationResponse {
	result := make([]pkgkanban.NotificationResponse, len(ns))
	for i, n := range ns {
		result[i] = ToPublicNotification(n)
	}
	return result
}
