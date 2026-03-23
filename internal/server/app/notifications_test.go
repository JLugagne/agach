package app_test

import (
	"context"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications/notificationstest"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestAppWithNotifications creates a test app with a notifications mock.
func setupTestAppWithNotifications(mockNotifications *notificationstest.MockNotificationRepository) *app.App {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	return app.NewApp(app.Config{
		Notifications: mockNotifications,
		Logger:        logger,
	})
}

// Notification Command Tests

func TestApp_CreateNotification_Success(t *testing.T) {
	ctx := context.Background()

	mockNotifications := &notificationstest.MockNotificationRepository{}
	mockNotifications.CreateFunc = func(ctx context.Context, notification domain.Notification) error {
		return nil
	}

	a := setupTestAppWithNotifications(mockNotifications)

	projectID := domain.NewProjectID()

	notification, err := a.CreateNotification(ctx, &projectID, domain.NotificationScopeProject, "", domain.SeverityInfo, "Build complete", "All tests passed", "/builds/123", "View Build", "primary")

	require.NoError(t, err)
	assert.NotEmpty(t, notification.ID)
	assert.Equal(t, &projectID, notification.ProjectID)
	assert.Equal(t, domain.NotificationScopeProject, notification.Scope)
	assert.Equal(t, domain.SeverityInfo, notification.Severity)
	assert.Equal(t, "Build complete", notification.Title)
	assert.Equal(t, "All tests passed", notification.Text)
	assert.Equal(t, "/builds/123", notification.LinkURL)
	assert.Equal(t, "View Build", notification.LinkText)
	assert.Equal(t, "primary", notification.LinkStyle)
	assert.False(t, notification.CreatedAt.IsZero())
}

func TestApp_CreateNotification_EmptyTitle_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a := setupTestAppWithNotifications(&notificationstest.MockNotificationRepository{})

	projectID := domain.NewProjectID()

	_, err := a.CreateNotification(ctx, &projectID, domain.NotificationScopeProject, "", domain.SeverityInfo, "", "Some text", "", "", "")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrNotificationTitleRequired)
}

func TestApp_CreateNotification_InvalidSeverity_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a := setupTestAppWithNotifications(&notificationstest.MockNotificationRepository{})

	projectID := domain.NewProjectID()

	_, err := a.CreateNotification(ctx, &projectID, domain.NotificationScopeProject, "", domain.NotificationSeverity("invalid"), "Title", "Some text", "", "", "")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrInvalidNotificationData)
}

// Notification Delegation Tests

func TestApp_MarkNotificationRead_DelegatesToRepository(t *testing.T) {
	ctx := context.Background()

	mockNotifications := &notificationstest.MockNotificationRepository{}
	notificationID := domain.NewNotificationID()

	var calledWithID domain.NotificationID
	mockNotifications.MarkReadFunc = func(ctx context.Context, id domain.NotificationID) error {
		calledWithID = id
		return nil
	}

	a := setupTestAppWithNotifications(mockNotifications)

	err := a.MarkNotificationRead(ctx, notificationID)

	require.NoError(t, err)
	assert.Equal(t, notificationID, calledWithID)
}

func TestApp_MarkAllNotificationsRead_DelegatesToRepository(t *testing.T) {
	ctx := context.Background()

	mockNotifications := &notificationstest.MockNotificationRepository{}
	projectID := domain.NewProjectID()

	var calledWithFilters notifications.NotificationFilters
	mockNotifications.MarkAllReadFunc = func(ctx context.Context, filters notifications.NotificationFilters) error {
		calledWithFilters = filters
		return nil
	}

	a := setupTestAppWithNotifications(mockNotifications)

	err := a.MarkAllNotificationsRead(ctx, &projectID)

	require.NoError(t, err)
	assert.Equal(t, &projectID, calledWithFilters.ProjectID)
}

func TestApp_DeleteNotification_DelegatesToRepository(t *testing.T) {
	ctx := context.Background()

	mockNotifications := &notificationstest.MockNotificationRepository{}
	notificationID := domain.NewNotificationID()

	var calledWithID domain.NotificationID
	mockNotifications.DeleteFunc = func(ctx context.Context, id domain.NotificationID) error {
		calledWithID = id
		return nil
	}

	a := setupTestAppWithNotifications(mockNotifications)

	err := a.DeleteNotification(ctx, notificationID)

	require.NoError(t, err)
	assert.Equal(t, notificationID, calledWithID)
}

// Notification Query Tests

func TestApp_ListNotifications_DelegatesToRepository(t *testing.T) {
	ctx := context.Background()

	mockNotifications := &notificationstest.MockNotificationRepository{}
	projectID := domain.NewProjectID()

	expectedNotifications := []domain.Notification{
		{ID: domain.NewNotificationID(), ProjectID: &projectID, Severity: domain.SeverityInfo, Title: "First"},
		{ID: domain.NewNotificationID(), ProjectID: &projectID, Severity: domain.SeverityWarning, Title: "Second"},
	}

	mockNotifications.ListFunc = func(ctx context.Context, filters notifications.NotificationFilters, limit, offset int) ([]domain.Notification, error) {
		if filters.ProjectID != nil && *filters.ProjectID == projectID && filters.UnreadOnly && limit == 10 && offset == 5 {
			return expectedNotifications, nil
		}
		return nil, nil
	}

	a := setupTestAppWithNotifications(mockNotifications)

	notifs, err := a.ListNotifications(ctx, &projectID, nil, "", true, 10, 5)

	require.NoError(t, err)
	assert.Equal(t, expectedNotifications, notifs)
}

func TestApp_GetNotificationUnreadCount_DelegatesToRepository(t *testing.T) {
	ctx := context.Background()

	mockNotifications := &notificationstest.MockNotificationRepository{}
	projectID := domain.NewProjectID()

	mockNotifications.UnreadCountFunc = func(ctx context.Context, filters notifications.NotificationFilters) (int, error) {
		if filters.ProjectID != nil && *filters.ProjectID == projectID {
			return 7, nil
		}
		return 0, nil
	}

	a := setupTestAppWithNotifications(mockNotifications)

	count, err := a.GetNotificationUnreadCount(ctx, &projectID, nil, "")

	require.NoError(t, err)
	assert.Equal(t, 7, count)
}
