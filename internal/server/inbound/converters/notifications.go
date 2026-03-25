package converters

import (
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicNotification converts domain.Notification to pkgserver.NotificationResponse
func ToPublicNotification(n domain.Notification) pkgserver.NotificationResponse {
	var readAt *string
	if n.ReadAt != nil {
		s := n.ReadAt.Format(time.RFC3339)
		readAt = &s
	}
	var projectID *string
	if n.ProjectID != nil {
		s := n.ProjectID.String()
		projectID = &s
	}
	return pkgserver.NotificationResponse{
		ID:        n.ID.String(),
		ProjectID: projectID,
		Scope:     string(n.Scope),
		AgentSlug: n.AgentSlug,
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

// ToPublicNotifications converts []domain.Notification to []pkgserver.NotificationResponse
func ToPublicNotifications(ns []domain.Notification) []pkgserver.NotificationResponse {
	return MapSlice(ns, ToPublicNotification)
}
