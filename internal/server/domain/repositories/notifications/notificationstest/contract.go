package notificationstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockNotificationRepository is a function-based mock implementation
type MockNotificationRepository struct {
	CreateFunc      func(ctx context.Context, notification domain.Notification) error
	FindByIDFunc    func(ctx context.Context, id domain.NotificationID) (*domain.Notification, error)
	ListFunc        func(ctx context.Context, filters notifications.NotificationFilters, limit, offset int) ([]domain.Notification, error)
	UnreadCountFunc func(ctx context.Context, filters notifications.NotificationFilters) (int, error)
	MarkReadFunc    func(ctx context.Context, id domain.NotificationID) error
	MarkAllReadFunc func(ctx context.Context, filters notifications.NotificationFilters) error
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

func (m *MockNotificationRepository) List(ctx context.Context, filters notifications.NotificationFilters, limit, offset int) ([]domain.Notification, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, filters, limit, offset)
}

func (m *MockNotificationRepository) UnreadCount(ctx context.Context, filters notifications.NotificationFilters) (int, error) {
	if m.UnreadCountFunc == nil {
		panic("called not defined UnreadCountFunc")
	}
	return m.UnreadCountFunc(ctx, filters)
}

func (m *MockNotificationRepository) MarkRead(ctx context.Context, id domain.NotificationID) error {
	if m.MarkReadFunc == nil {
		panic("called not defined MarkReadFunc")
	}
	return m.MarkReadFunc(ctx, id)
}

func (m *MockNotificationRepository) MarkAllRead(ctx context.Context, filters notifications.NotificationFilters) error {
	if m.MarkAllReadFunc == nil {
		panic("called not defined MarkAllReadFunc")
	}
	return m.MarkAllReadFunc(ctx, filters)
}

func (m *MockNotificationRepository) Delete(ctx context.Context, id domain.NotificationID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

// helper to create a *ProjectID from a ProjectID value.
func ptrProjectID(id domain.ProjectID) *domain.ProjectID {
	return &id
}

// NotificationsContractTesting runs all contract tests for a NotificationRepository implementation.
func NotificationsContractTesting(t *testing.T, repo notifications.NotificationRepository, projectID domain.ProjectID) {
	ctx := context.Background()

	// ── Create / FindByID with project scope ────────────────────────────
	t.Run("Contract: Create and FindByID with project scope", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeverityInfo,
			Title:     "Project Notification",
			Text:      "Project-scoped notification",
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, notif)
		require.NoError(t, err)

		retrieved, err := repo.FindByID(ctx, notif.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, notif.ID, retrieved.ID)
		assert.Equal(t, projectID, *retrieved.ProjectID)
		assert.Equal(t, domain.NotificationScopeProject, retrieved.Scope)
		assert.Empty(t, retrieved.AgentSlug)
		assert.Nil(t, retrieved.ReadAt)
	})

	// ── Create / FindByID with agent scope ──────────────────────────────
	t.Run("Contract: Create and FindByID with agent scope", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeAgent,
			AgentSlug: "backend-dev",
			Severity:  domain.SeverityWarning,
			Title:     "Agent Notification",
			Text:      "Agent-scoped notification",
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, notif)
		require.NoError(t, err)

		retrieved, err := repo.FindByID(ctx, notif.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, domain.NotificationScopeAgent, retrieved.Scope)
		assert.Equal(t, "backend-dev", retrieved.AgentSlug)
	})

	// ── Create / FindByID with global scope ─────────────────────────────
	t.Run("Contract: Create and FindByID with global scope", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			Scope:     domain.NotificationScopeGlobal,
			Severity:  domain.SeveritySuccess,
			Title:     "Global Notification",
			Text:      "Global-scoped notification",
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, notif)
		require.NoError(t, err)

		retrieved, err := repo.FindByID(ctx, notif.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, domain.NotificationScopeGlobal, retrieved.Scope)
		assert.Nil(t, retrieved.ProjectID)
	})

	// ── Create with link fields ─────────────────────────────────────────
	t.Run("Contract: Create stores notification with link fields", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeveritySuccess,
			Title:     "Feature Completed",
			Text:      "Feature X is done",
			LinkURL:   "/projects/123/features/456",
			LinkText:  "View Feature",
			LinkStyle: "primary",
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, notif)
		require.NoError(t, err)

		retrieved, err := repo.FindByID(ctx, notif.ID)
		require.NoError(t, err)
		assert.Equal(t, notif.LinkURL, retrieved.LinkURL)
		assert.Equal(t, notif.LinkText, retrieved.LinkText)
		assert.Equal(t, notif.LinkStyle, retrieved.LinkStyle)
	})

	// ── FindByID non-existent ───────────────────────────────────────────
	t.Run("Contract: FindByID returns error for non-existent notification", func(t *testing.T) {
		_, err := repo.FindByID(ctx, domain.NewNotificationID())
		assert.Error(t, err)
		assert.True(t, domain.IsDomainError(err))
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	// ── List with no filters returns all (global view) ──────────────────
	t.Run("Contract: List with no filters returns all notifications", func(t *testing.T) {
		// Create one notification of each scope
		now := time.Now()
		projectNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeverityInfo,
			Title:     "List-All Project",
			Text:      "project scoped",
			CreatedAt: now.Add(-2 * time.Second),
		}
		agentNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeAgent,
			AgentSlug: "frontend-dev",
			Severity:  domain.SeverityInfo,
			Title:     "List-All Agent",
			Text:      "agent scoped",
			CreatedAt: now.Add(-1 * time.Second),
		}
		globalNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			Scope:     domain.NotificationScopeGlobal,
			Severity:  domain.SeverityInfo,
			Title:     "List-All Global",
			Text:      "global scoped",
			CreatedAt: now,
		}

		for _, n := range []domain.Notification{projectNotif, agentNotif, globalNotif} {
			err := repo.Create(ctx, n)
			require.NoError(t, err)
		}

		all, err := repo.List(ctx, notifications.NotificationFilters{}, 0, 0)
		require.NoError(t, err)

		ids := make(map[domain.NotificationID]bool)
		for _, n := range all {
			ids[n.ID] = true
		}
		assert.True(t, ids[projectNotif.ID], "Project-scoped notification should appear in unfiltered list")
		assert.True(t, ids[agentNotif.ID], "Agent-scoped notification should appear in unfiltered list")
		assert.True(t, ids[globalNotif.ID], "Global-scoped notification should appear in unfiltered list")
	})

	// ── List filtered by project ────────────────────────────────────────
	t.Run("Contract: List filtered by project", func(t *testing.T) {
		// Use the provided projectID (which exists in DB) and a global notification
		projNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeverityInfo,
			Title:     "Project Filter Test",
			Text:      "belongs to test project",
			CreatedAt: time.Now(),
		}
		globalNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			Scope:     domain.NotificationScopeGlobal,
			Severity:  domain.SeverityInfo,
			Title:     "Global Filter Test",
			Text:      "should not appear in project filter",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, projNotif)
		require.NoError(t, err)
		err = repo.Create(ctx, globalNotif)
		require.NoError(t, err)

		filtered, err := repo.List(ctx, notifications.NotificationFilters{
			ProjectID: ptrProjectID(projectID),
		}, 0, 0)
		require.NoError(t, err)

		for _, n := range filtered {
			require.NotNil(t, n.ProjectID, "All returned notifications should have a project ID")
			assert.Equal(t, projectID, *n.ProjectID, "All returned notifications should belong to the filtered project")
		}
		foundProj := false
		foundGlobal := false
		for _, n := range filtered {
			if n.ID == projNotif.ID {
				foundProj = true
			}
			if n.ID == globalNotif.ID {
				foundGlobal = true
			}
		}
		assert.True(t, foundProj, "The project notification should be returned")
		assert.False(t, foundGlobal, "The global notification should NOT be returned in project filter")
	})

	// ── List filtered by scope ──────────────────────────────────────────
	t.Run("Contract: List filtered by scope", func(t *testing.T) {
		globalScope := domain.NotificationScopeGlobal
		filtered, err := repo.List(ctx, notifications.NotificationFilters{
			Scope: &globalScope,
		}, 0, 0)
		require.NoError(t, err)

		for _, n := range filtered {
			assert.Equal(t, domain.NotificationScopeGlobal, n.Scope, "All returned notifications should have global scope")
		}
		assert.NotEmpty(t, filtered, "Should find at least one global notification")
	})

	// ── List filtered by agent_slug ─────────────────────────────────────
	t.Run("Contract: List filtered by agent_slug", func(t *testing.T) {
		agentNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeAgent,
			AgentSlug: "unique-agent-slug",
			Severity:  domain.SeverityInfo,
			Title:     "Agent Filter Test",
			Text:      "for agent filter test",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, agentNotif)
		require.NoError(t, err)

		filtered, err := repo.List(ctx, notifications.NotificationFilters{
			AgentSlug: "unique-agent-slug",
		}, 0, 0)
		require.NoError(t, err)

		require.NotEmpty(t, filtered)
		for _, n := range filtered {
			assert.Equal(t, "unique-agent-slug", n.AgentSlug)
		}
	})

	// ── List unreadOnly ─────────────────────────────────────────────────
	t.Run("Contract: List unreadOnly filters out read notifications", func(t *testing.T) {
		readNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeverityInfo,
			Title:     "Will be read",
			Text:      "This will be marked as read",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, readNotif)
		require.NoError(t, err)
		err = repo.MarkRead(ctx, readNotif.ID)
		require.NoError(t, err)

		unreadNotif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeverityWarning,
			Title:     "Still unread",
			Text:      "This is still unread",
			CreatedAt: time.Now(),
		}
		err = repo.Create(ctx, unreadNotif)
		require.NoError(t, err)

		unreadList, err := repo.List(ctx, notifications.NotificationFilters{UnreadOnly: true}, 0, 0)
		require.NoError(t, err)

		for _, n := range unreadList {
			assert.NotEqual(t, readNotif.ID, n.ID, "Read notification should not appear in unread list")
		}
		found := false
		for _, n := range unreadList {
			if n.ID == unreadNotif.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Unread notification should appear in unread list")
	})

	// ── List with limit and offset ──────────────────────────────────────
	t.Run("Contract: List with limit and offset", func(t *testing.T) {
		all, err := repo.List(ctx, notifications.NotificationFilters{}, 0, 0)
		require.NoError(t, err)

		if len(all) >= 2 {
			limited, err := repo.List(ctx, notifications.NotificationFilters{}, 1, 0)
			require.NoError(t, err)
			assert.Len(t, limited, 1, "Should return exactly 1 notification with limit=1")

			offset, err := repo.List(ctx, notifications.NotificationFilters{}, 1, 1)
			require.NoError(t, err)
			assert.Len(t, offset, 1, "Should return exactly 1 notification with limit=1 offset=1")
			assert.NotEqual(t, limited[0].ID, offset[0].ID, "Offset results should differ from first page")
		}
	})

	// ── List ordered by created_at DESC ─────────────────────────────────
	t.Run("Contract: List returns notifications ordered by created_at DESC", func(t *testing.T) {
		now := time.Now()
		notifs := []domain.Notification{
			{
				ID:        domain.NewNotificationID(),
				ProjectID: ptrProjectID(projectID),
				Scope:     domain.NotificationScopeProject,
				Severity:  domain.SeverityInfo,
				Title:     "Order Oldest",
				Text:      "oldest",
				CreatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:        domain.NewNotificationID(),
				ProjectID: ptrProjectID(projectID),
				Scope:     domain.NotificationScopeProject,
				Severity:  domain.SeverityWarning,
				Title:     "Order Middle",
				Text:      "middle",
				CreatedAt: now.Add(-1 * time.Hour),
			},
			{
				ID:        domain.NewNotificationID(),
				ProjectID: ptrProjectID(projectID),
				Scope:     domain.NotificationScopeProject,
				Severity:  domain.SeveritySuccess,
				Title:     "Order Newest",
				Text:      "newest",
				CreatedAt: now,
			},
		}

		for _, n := range notifs {
			err := repo.Create(ctx, n)
			require.NoError(t, err)
		}

		retrieved, err := repo.List(ctx, notifications.NotificationFilters{
			ProjectID: ptrProjectID(projectID),
		}, 0, 0)
		require.NoError(t, err)

		var testNotifs []domain.Notification
		for _, r := range retrieved {
			for _, n := range notifs {
				if r.ID == n.ID {
					testNotifs = append(testNotifs, r)
					break
				}
			}
		}
		require.Len(t, testNotifs, 3)
		assert.Equal(t, "Order Newest", testNotifs[0].Title)
		assert.Equal(t, "Order Middle", testNotifs[1].Title)
		assert.Equal(t, "Order Oldest", testNotifs[2].Title)
	})

	// ── UnreadCount with no filters ─────────────────────────────────────
	t.Run("Contract: UnreadCount with no filters", func(t *testing.T) {
		initialCount, err := repo.UnreadCount(ctx, notifications.NotificationFilters{})
		require.NoError(t, err)

		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			Scope:     domain.NotificationScopeGlobal,
			Severity:  domain.SeverityInfo,
			Title:     "Global Count Test",
			Text:      "for counting globally",
			CreatedAt: time.Now(),
		}
		err = repo.Create(ctx, notif)
		require.NoError(t, err)

		newCount, err := repo.UnreadCount(ctx, notifications.NotificationFilters{})
		require.NoError(t, err)
		assert.Equal(t, initialCount+1, newCount, "Unread count should increase by 1")

		err = repo.MarkRead(ctx, notif.ID)
		require.NoError(t, err)

		afterReadCount, err := repo.UnreadCount(ctx, notifications.NotificationFilters{})
		require.NoError(t, err)
		assert.Equal(t, initialCount, afterReadCount, "Unread count should decrease after marking read")
	})

	// ── UnreadCount with project filter ─────────────────────────────────
	t.Run("Contract: UnreadCount with project filter", func(t *testing.T) {
		// Mark all existing project notifications as read first
		err := repo.MarkAllRead(ctx, notifications.NotificationFilters{
			ProjectID: ptrProjectID(projectID),
		})
		require.NoError(t, err)

		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeverityInfo,
			Title:     "Isolated Count",
			Text:      "project-specific count",
			CreatedAt: time.Now(),
		}
		err = repo.Create(ctx, notif)
		require.NoError(t, err)

		count, err := repo.UnreadCount(ctx, notifications.NotificationFilters{
			ProjectID: ptrProjectID(projectID),
		})
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	// ── MarkRead sets read_at ───────────────────────────────────────────
	t.Run("Contract: MarkRead sets read_at", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
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

	// ── MarkRead non-existent ───────────────────────────────────────────
	t.Run("Contract: MarkRead returns error for non-existent notification", func(t *testing.T) {
		err := repo.MarkRead(ctx, domain.NewNotificationID())
		assert.Error(t, err)
		assert.True(t, domain.IsDomainError(err))
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})

	// ── MarkAllRead with project filter only affects that project ───────
	t.Run("Contract: MarkAllRead with project filter only affects that project", func(t *testing.T) {
		// Use project-scoped (existing projectID) vs global-scoped to test isolation
		notifProject := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
			Severity:  domain.SeverityInfo,
			Title:     "MarkAllRead Project",
			Text:      "project notification",
			CreatedAt: time.Now(),
		}
		notifGlobal := domain.Notification{
			ID:        domain.NewNotificationID(),
			Scope:     domain.NotificationScopeGlobal,
			Severity:  domain.SeverityInfo,
			Title:     "MarkAllRead Global",
			Text:      "global notification",
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, notifProject)
		require.NoError(t, err)
		err = repo.Create(ctx, notifGlobal)
		require.NoError(t, err)

		// Mark all read for the project only
		err = repo.MarkAllRead(ctx, notifications.NotificationFilters{
			ProjectID: ptrProjectID(projectID),
		})
		require.NoError(t, err)

		// The global notification should still be unread
		globalNotif, err := repo.FindByID(ctx, notifGlobal.ID)
		require.NoError(t, err)
		assert.Nil(t, globalNotif.ReadAt, "Global notification should still be unread after project-scoped MarkAllRead")

		// The project notification should be read
		projNotif, err := repo.FindByID(ctx, notifProject.ID)
		require.NoError(t, err)
		assert.NotNil(t, projNotif.ReadAt, "Project notification should be read after project-scoped MarkAllRead")
	})

	// ── Delete removes notification ─────────────────────────────────────
	t.Run("Contract: Delete removes notification", func(t *testing.T) {
		notif := domain.Notification{
			ID:        domain.NewNotificationID(),
			ProjectID: ptrProjectID(projectID),
			Scope:     domain.NotificationScopeProject,
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

	// ── Delete non-existent ─────────────────────────────────────────────
	t.Run("Contract: Delete returns error for non-existent notification", func(t *testing.T) {
		err := repo.Delete(ctx, domain.NewNotificationID())
		assert.Error(t, err)
		assert.True(t, domain.IsDomainError(err))
		assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	})
}
