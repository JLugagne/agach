package notifications

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// NotificationRepository defines operations for managing notifications within a project
type NotificationRepository interface {
	// Create creates a new notification
	Create(ctx context.Context, notification domain.Notification) error

	// FindByID retrieves a notification by ID
	FindByID(ctx context.Context, id domain.NotificationID) (*domain.Notification, error)

	// List retrieves notifications for a project, ordered by created_at DESC.
	// If unreadOnly is true, only unread notifications are returned.
	List(ctx context.Context, projectID domain.ProjectID, unreadOnly bool, limit, offset int) ([]domain.Notification, error)

	// UnreadCount returns the number of unread notifications for a project
	UnreadCount(ctx context.Context, projectID domain.ProjectID) (int, error)

	// MarkRead marks a notification as read
	MarkRead(ctx context.Context, id domain.NotificationID) error

	// MarkAllRead marks all notifications as read for a project
	MarkAllRead(ctx context.Context, projectID domain.ProjectID) error

	// Delete deletes a notification
	Delete(ctx context.Context, id domain.NotificationID) error
}
