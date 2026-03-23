package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications"
)

func (a *App) CreateNotification(ctx context.Context, projectID *domain.ProjectID, scope domain.NotificationScope, agentSlug string, severity domain.NotificationSeverity, title, text, linkURL, linkText, linkStyle string) (domain.Notification, error) {
	if title == "" {
		return domain.Notification{}, domain.ErrNotificationTitleRequired
	}

	if !domain.ValidNotificationSeverities[severity] {
		return domain.Notification{}, domain.ErrInvalidNotificationData
	}

	if !domain.ValidNotificationScopes[scope] {
		return domain.Notification{}, domain.ErrInvalidNotificationData
	}

	notification := domain.Notification{
		ID:        domain.NewNotificationID(),
		ProjectID: projectID,
		Scope:     scope,
		AgentSlug: agentSlug,
		Severity:  severity,
		Title:     title,
		Text:      text,
		LinkURL:   linkURL,
		LinkText:  linkText,
		LinkStyle: linkStyle,
		CreatedAt: time.Now(),
	}

	if err := a.notifications.Create(ctx, notification); err != nil {
		a.logger.WithError(err).Error("failed to create notification")
		return domain.Notification{}, err
	}

	return notification, nil
}

func (a *App) MarkNotificationRead(ctx context.Context, notificationID domain.NotificationID) error {
	return a.notifications.MarkRead(ctx, notificationID)
}

func (a *App) MarkAllNotificationsRead(ctx context.Context, projectID *domain.ProjectID) error {
	return a.notifications.MarkAllRead(ctx, notifications.NotificationFilters{
		ProjectID: projectID,
	})
}

func (a *App) DeleteNotification(ctx context.Context, notificationID domain.NotificationID) error {
	return a.notifications.Delete(ctx, notificationID)
}

func (a *App) ListNotifications(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	return a.notifications.List(ctx, notifications.NotificationFilters{
		ProjectID:  projectID,
		Scope:      scope,
		AgentSlug:  agentSlug,
		UnreadOnly: unreadOnly,
	}, limit, offset)
}

func (a *App) GetNotificationUnreadCount(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string) (int, error) {
	return a.notifications.UnreadCount(ctx, notifications.NotificationFilters{
		ProjectID: projectID,
		Scope:     scope,
		AgentSlug: agentSlug,
	})
}
