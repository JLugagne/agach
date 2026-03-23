package notificationstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/notifications"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockNotificationRepository is a function-based mock implementation
type MockNotificationRepository struct {
	CreateFunc      func(ctx context.Context, notification domain.Notification) error
	FindByIDFunc    func(ctx context.Context, id domain.NotificationID) (*domain.Notification, error)
	ListFunc        func(ctx context.Context, projectID domain.ProjectID, unreadOnly bool, limit, offset int) ([]domain.Notification, error)
	UnreadCountFunc func(ctx context.Context, projectID domain.ProjectID) (int, error)
	MarkReadFunc    func(ctx context.Context, id domain.NotificationID) error
	MarkAllReadFunc func(ctx context.Context, projectID domain.ProjectID) error
	DeleteFunc      func(ctx context.Context, id domain.NotificationID) error
}

func (m *MockNotificationRepository) Create(ctx context.Context, notification domain.Notification) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, notification)
}

func (m *MockNotificationRepository) FindByID(ctx context.Context, id domain.NotificationID) (*domain.Notification, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockNotificationRepository) List(ctx context.Context, projectID domain.ProjectID, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, projectID, unreadOnly, limit, offset)
}

func (m *MockNotificationRepository) UnreadCount(ctx context.Context, projectID domain.ProjectID) (int, error) {
	if m.UnreadCountFunc == nil {
		panic("called not defined UnreadCountFunc")
	}
	return m.UnreadCountFunc(ctx, projectID)
}

func (m *MockNotificationRepository) MarkRead(ctx context.Context, id domain.NotificationID) error {
	if m.MarkReadFunc == nil {
		panic("called not defined MarkReadFunc")
	}
	return m.MarkReadFunc(ctx, id)
}

func (m *MockNotificationRepository) MarkAllRead(ctx context.Context, projectID domain.ProjectID) error {
	if m.MarkAllReadFunc == nil {
		panic("called not defined MarkAllReadFunc")
	}
	return m.MarkAllReadFunc(ctx, projectID)
}

func (m *MockNotificationRepository) Delete(ctx context.Context, id domain.NotificationID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

// NotificationsContractTesting runs all contract tests for a NotificationRepository implementation.
func NotificationsContractTesting(t *testing.T, repo notifications.NotificationRepository, projectID domain.ProjectID) {
	ctx := context.Background()

	t.Run("Contract: Create stores notification and FindByID retrieves it", func(t *testing.T) {
		notification := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: projectID,
			Severity:  domain.SeverityInfo,
			Title:     "Test Notification",
			Text:      "This is a test notification",
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, notification)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, notification.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved, "Retrieved notification must not be nil")
		assert.Equal(t, notification.ID, retrieved.ID)
		assert.Equal(t, notification.ProjectID, retrieved.ProjectID)
		assert.Equal(t, notification.Severity, retrieved.Severity)
		assert.Equal(t, notification.Title, retrieved.Title)
		assert.Equal(t, notification.Text, retrieved.Text)
		assert.Nil(t, retrieved.ReadAt, "ReadAt should be nil for new notification")
	})

	t.Run("Contract: Create stores notification with link fields", func(t *testing.T) {
		notification := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: projectID,
			Severity:  domain.SeveritySuccess,
			Title:     "Feature Completed",
			Text:      "Feature X is done",
			LinkURL:   "/projects/123/features/456",
			LinkText:  "View Feature",
			LinkStyle: "primary",
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, notification)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, notification.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, notification.LinkURL, retrieved.LinkURL)
		assert.Equal(t, notification.LinkText, retrieved.LinkText)
		assert.Equal(t, notification.LinkStyle, retrieved.LinkStyle)
	})

	t.Run("Contract: FindByID returns error for non-existent notification", func(t *testing.T) {
		nonExistentID := domain.NewNotificationID()
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Error(t, err, "FindByID should return error for non-existent notification")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Contract: List returns notifications ordered by created_at DESC", func(t *testing.T) {
		// Create a unique project to avoid contamination from other tests
		// We'll use the same projectID but rely on created_at ordering
		now := time.Now()
		notifs := []domain.Notification{
			{
				ID:        domain.NewNotificationID(),
				ProjectID: projectID,
				Severity:  domain.SeverityInfo,
				Title:     "Oldest",
				Text:      "Oldest notification",
				CreatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:        domain.NewNotificationID(),
				ProjectID: projectID,
				Severity:  domain.SeverityWarning,
				Title:     "Middle",
				Text:      "Middle notification",
				CreatedAt: now.Add(-1 * time.Hour),
			},
			{
				ID:        domain.NewNotificationID(),
				ProjectID: projectID,
				Severity:  domain.SeveritySuccess,
				Title:     "Newest",
				Text:      "Newest notification",
				CreatedAt: now,
			},
		}

		for _, n := range notifs {
			err := repo.Create(ctx, n)
			require.NoError(t, err, "Create should succeed")
		}

		retrieved, err := repo.List(ctx, projectID, false, 0, 0)
		require.NoError(t, err, "List should succeed")
		require.GreaterOrEqual(t, len(retrieved), 3, "Should return at least 3 notifications")

		// Find our test notifications and verify DESC ordering
		var testNotifs []domain.Notification
		for _, r := range retrieved {
			for _, n := range notifs {
				if r.ID == n.ID {
					testNotifs = append(testNotifs, r)
					break
				}
			}
		}
		require.Len(t, testNotifs, 3, "Should find all test notifications")
		assert.Equal(t, "Newest", testNotifs[0].Title)
		assert.Equal(t, "Middle", testNotifs[1].Title)
		assert.Equal(t, "Oldest", testNotifs[2].Title)
	})

	t.Run("Contract: List with limit and offset", func(t *testing.T) {
		all, err := repo.List(ctx, projectID, false, 0, 0)
		require.NoError(t, err)

		if len(all) >= 2 {
			limited, err := repo.List(ctx, projectID, false, 1, 0)
			require.NoError(t, err)
			assert.Len(t, limited, 1, "Should return exactly 1 notification with limit=1")

			offset, err := repo.List(ctx, projectID, false, 1, 1)
			require.NoError(t, err)
			assert.Len(t, offset, 1, "Should return exactly 1 notification with limit=1 offset=1")
			assert.NotEqual(t, limited[0].ID, offset[0].ID, "Offset results should differ from first page")
		}
	})

	t.Run("Contract: List unreadOnly filters out read notifications", func(t *testing.T) {
		// Create a notification and mark it as read
		readNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: projectID,
			Severity:  domain.SeverityInfo,
			Title:     "Will be read",
			Text:      "This will be marked as read",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, readNotif)
		require.NoError(t, err)

		err = repo.MarkRead(ctx, readNotif.ID)
		require.NoError(t, err)

		// Create an unread notification
		unreadNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: projectID,
			Severity:  domain.SeverityWarning,
			Title:     "Still unread",
			Text:      "This is still unread",
			CreatedAt: time.Now(),
		}
		err = repo.Create(ctx, unreadNotif)
		require.NoError(t, err)

		// List unread only
		unreadList, err := repo.List(ctx, projectID, true, 0, 0)
		require.NoError(t, err)

		// The read notification should not appear
		for _, n := range unreadList {
			assert.NotEqual(t, readNotif.ID, n.ID, "Read notification should not appear in unread list")
		}

		// The unread notification should appear
		found := false
		for _, n := range unreadList {
			if n.ID == unreadNotif.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Unread notification should appear in unread list")
	})

	t.Run("Contract: UnreadCount returns correct count", func(t *testing.T) {
		// Get initial count
		initialCount, err := repo.UnreadCount(ctx, projectID)
		require.NoError(t, err)

		// Create an unread notification
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: projectID,
			Severity:  domain.SeverityInfo,
			Title:     "Count test",
			Text:      "For counting",
			CreatedAt: time.Now(),
		}
		err = repo.Create(ctx, notif)
		require.NoError(t, err)

		newCount, err := repo.UnreadCount(ctx, projectID)
		require.NoError(t, err)
		assert.Equal(t, initialCount+1, newCount, "Unread count should increase by 1")

		// Mark it as read
		err = repo.MarkRead(ctx, notif.ID)
		require.NoError(t, err)

		afterReadCount, err := repo.UnreadCount(ctx, projectID)
		require.NoError(t, err)
		assert.Equal(t, initialCount, afterReadCount, "Unread count should decrease after marking read")
	})

	t.Run("Contract: MarkRead sets read_at", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: projectID,
			Severity:  domain.SeverityInfo,
			Title:     "Mark read test",
			Text:      "Will be marked read",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, notif)
		require.NoError(t, err)

		err = repo.MarkRead(ctx, notif.ID)
		require.NoError(t, err)

		retrieved, err := repo.FindByID(ctx, notif.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.ReadAt, "ReadAt should be set after MarkRead")
	})

	t.Run("Contract: MarkRead returns error for non-existent notification", func(t *testing.T) {
		nonExistentID := domain.NewNotificationID()
		err := repo.MarkRead(ctx, nonExistentID)
		assert.Error(t, err)
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Contract: MarkAllRead marks all notifications as read", func(t *testing.T) {
		// Create a couple of unread notifications
		for i := 0; i < 2; i++ {
			notif := domain.Notification{
				ID:        domain.NewNotificationID(),
				ProjectID: projectID,
				Severity:  domain.SeverityInfo,
				Title:     "MarkAllRead test",
				Text:      "For mark all read",
				CreatedAt: time.Now(),
			}
			err := repo.Create(ctx, notif)
			require.NoError(t, err)
		}

		err := repo.MarkAllRead(ctx, projectID)
		require.NoError(t, err)

		count, err := repo.UnreadCount(ctx, projectID)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Unread count should be 0 after MarkAllRead")
	})

	t.Run("Contract: Delete removes notification", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: projectID,
			Severity:  domain.SeverityInfo,
			Title:     "Delete test",
			Text:      "Will be deleted",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, notif)
		require.NoError(t, err)

		err = repo.Delete(ctx, notif.ID)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, notif.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	t.Run("Contract: Delete returns error for non-existent notification", func(t *testing.T) {
		nonExistentID := domain.NewNotificationID()
		err := repo.Delete(ctx, nonExistentID)
		assert.Error(t, err)
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}
