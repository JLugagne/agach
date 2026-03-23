package notifications

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// NotificationFilters defines filtering criteria for listing notifications.
type NotificationFilters struct {
	ProjectID  *domain.ProjectID         // filter by project (nil = no project filter)
	Scope      *domain.NotificationScope // filter by scope (nil = all scopes)
	AgentSlug  string                    // filter by agent slug (empty = no agent filter)
	UnreadOnly bool                      // only unread notifications
}

// NotificationRepository defines operations for managing notifications
type NotificationRepository interface {
	// Create creates a new notification
	Create(ctx context.Context, notification domain.Notification) error

	// FindByID retrieves a notification by ID
	FindByID(ctx context.Context, id domain.NotificationID) (*domain.Notification, error)

	// List retrieves notifications matching filters, ordered by created_at DESC.
	List(ctx context.Context, filters NotificationFilters, limit, offset int) ([]domain.Notification, error)

	// UnreadCount returns the number of unread notifications matching filters.
	UnreadCount(ctx context.Context, filters NotificationFilters) (int, error)

	// MarkRead marks a notification as read
	MarkRead(ctx context.Context, id domain.NotificationID) error

	// MarkAllRead marks all notifications matching filters as read
	MarkAllRead(ctx context.Context, filters NotificationFilters) error

	// Delete deletes a notification
	Delete(ctx context.Context, id domain.NotificationID) error
}
