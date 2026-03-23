package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// Notification Commands

func (a *App) CreateNotification(ctx context.Context, projectID domain.ProjectID, severity domain.NotificationSeverity, title, text, linkURL, linkText, linkStyle string) (domain.Notification, error) {
	if title == "" {
		return domain.Notification{}, domain.ErrNotificationTitleRequired
	}

	if !domain.ValidNotificationSeverities[severity] {
		return domain.Notification{}, domain.ErrInvalidNotificationData
	}

	notification := domain.Notification{
		ID:        domain.NewNotificationID(),
		ProjectID: projectID,
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

func (a *App) MarkAllNotificationsRead(ctx context.Context, projectID domain.ProjectID) error {
	return a.notifications.MarkAllRead(ctx, projectID)
}

func (a *App) DeleteNotification(ctx context.Context, notificationID domain.NotificationID) error {
	return a.notifications.Delete(ctx, notificationID)
}

// Notification Queries

func (a *App) ListNotifications(ctx context.Context, projectID domain.ProjectID, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	return a.notifications.List(ctx, projectID, unreadOnly, limit, offset)
}

func (a *App) GetNotificationUnreadCount(ctx context.Context, projectID domain.ProjectID) (int, error) {
	return a.notifications.UnreadCount(ctx, projectID)
}
