package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	notificationsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications"
	"github.com/sirupsen/logrus"
)

type NotificationService struct {
	notifications notificationsrepo.NotificationRepository
	logger        *logrus.Logger
}

func newNotificationService(notifications notificationsrepo.NotificationRepository, logger *logrus.Logger) *NotificationService {
	return &NotificationService{
		notifications: notifications,
		logger:        logger,
	}
}

func (s *NotificationService) CreateNotification(ctx context.Context, projectID *domain.ProjectID, scope domain.NotificationScope, agentSlug string, severity domain.NotificationSeverity, title, text, linkURL, linkText, linkStyle string) (domain.Notification, error) {
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

	if err := s.notifications.Create(ctx, notification); err != nil {
		s.logger.WithError(err).Error("failed to create notification")
		return domain.Notification{}, err
	}

	return notification, nil
}

func (s *NotificationService) MarkNotificationRead(ctx context.Context, notificationID domain.NotificationID) error {
	return s.notifications.MarkRead(ctx, notificationID)
}

func (s *NotificationService) MarkAllNotificationsRead(ctx context.Context, projectID *domain.ProjectID) error {
	return s.notifications.MarkAllRead(ctx, notificationsrepo.NotificationFilters{
		ProjectID: projectID,
	})
}

func (s *NotificationService) DeleteNotification(ctx context.Context, notificationID domain.NotificationID) error {
	return s.notifications.Delete(ctx, notificationID)
}

func (s *NotificationService) ListNotifications(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	return s.notifications.List(ctx, notificationsrepo.NotificationFilters{
		ProjectID:  projectID,
		Scope:      scope,
		AgentSlug:  agentSlug,
		UnreadOnly: unreadOnly,
	}, limit, offset)
}

func (s *NotificationService) GetNotificationUnreadCount(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string) (int, error) {
	return s.notifications.UnreadCount(ctx, notificationsrepo.NotificationFilters{
		ProjectID: projectID,
		Scope:     scope,
		AgentSlug: agentSlug,
	})
}
