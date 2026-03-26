package e2eapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func createProjectForNotifications(t *testing.T, token string) string {
	t.Helper()
	type proj struct {
		ID string `json:"id"`
	}
	p := createAndDecode[proj](t, "/api/projects", token, map[string]any{
		"name":        "Notification Test " + t.Name(),
		"description": "test",
	})
	return p.ID
}

// ---------------------------------------------------------------------------
// Inline response structs
// ---------------------------------------------------------------------------

type notificationResp struct {
	ID        string  `json:"id"`
	ProjectID *string `json:"project_id"`
	Scope     string  `json:"scope"`
	AgentSlug *string `json:"agent_slug"`
	Severity  string  `json:"severity"`
	Title     string  `json:"title"`
	Text      string  `json:"text"`
	LinkURL   *string `json:"link_url"`
	LinkText  *string `json:"link_text"`
	LinkStyle *string `json:"link_style"`
	ReadAt    *string `json:"read_at"`
	CreatedAt string  `json:"created_at"`
}

type unreadCountResp struct {
	Count int `json:"count"`
}

// ---------------------------------------------------------------------------
// 1. TestNotifications_ProjectScoped_CRUD
// ---------------------------------------------------------------------------

func TestNotifications_ProjectScoped_CRUD(t *testing.T) {
	tok := adminToken(t)
	projID := createProjectForNotifications(t, tok)

	// CREATE project-scoped notification
	created := createAndDecode[notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok, map[string]any{
			"severity": "info",
			"title":    "Build completed",
			"text":     "The build finished successfully.",
		})
	require.NotEmpty(t, created.ID)
	assert.Equal(t, "info", created.Severity)
	assert.Equal(t, "Build completed", created.Title)
	assert.Equal(t, "The build finished successfully.", created.Text)
	assert.Nil(t, created.ReadAt, "new notification should be unread")

	// LIST project notifications — must contain the created one
	notifs := getAndDecode[[]notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok)
	found := false
	for _, n := range notifs {
		if n.ID == created.ID {
			found = true
			assert.Equal(t, "info", n.Severity)
			break
		}
	}
	require.True(t, found, "created notification not found in project list")

	// MARK READ
	resp := doAuth(t, "PUT", fmt.Sprintf("/api/notifications/%s/read", created.ID), tok, nil)
	requireStatus(t, resp, http.StatusOK)
	marked := decode[notificationResp](t, resp)
	require.NotNil(t, marked.ReadAt, "read_at should be set after marking read")

	// Verify read_at is set via GET
	allNotifs := getAndDecode[[]notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok)
	for _, n := range allNotifs {
		if n.ID == created.ID {
			require.NotNil(t, n.ReadAt, "read_at should persist")
			break
		}
	}

	// DELETE
	deleteResource(t, fmt.Sprintf("/api/notifications/%s", created.ID), tok)

	// Verify gone from project list
	notifsAfter := getAndDecode[[]notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok)
	for _, n := range notifsAfter {
		assert.NotEqual(t, created.ID, n.ID, "deleted notification should not appear in list")
	}
}

// ---------------------------------------------------------------------------
// 2. TestNotifications_Global
// ---------------------------------------------------------------------------

func TestNotifications_Global(t *testing.T) {
	tok := adminToken(t)

	// CREATE global notification
	created := createAndDecode[notificationResp](t, "/api/notifications", tok, map[string]any{
		"severity": "warning",
		"title":    "System maintenance",
		"text":     "Scheduled downtime at midnight.",
	})
	require.NotEmpty(t, created.ID)
	assert.Equal(t, "global", created.Scope)
	assert.Equal(t, "warning", created.Severity)
	assert.Equal(t, "System maintenance", created.Title)

	// LIST all notifications — must contain the global one
	notifs := getAndDecode[[]notificationResp](t, "/api/notifications", tok)
	found := false
	for _, n := range notifs {
		if n.ID == created.ID {
			found = true
			assert.Equal(t, "global", n.Scope)
			break
		}
	}
	require.True(t, found, "global notification not found in list")

	// Cleanup
	deleteResource(t, fmt.Sprintf("/api/notifications/%s", created.ID), tok)
}

// ---------------------------------------------------------------------------
// 3. TestNotifications_UnreadCount
// ---------------------------------------------------------------------------

func TestNotifications_UnreadCount(t *testing.T) {
	tok := adminToken(t)
	projID := createProjectForNotifications(t, tok)

	// Create 2 project notifications
	n1 := createAndDecode[notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok, map[string]any{
			"severity": "info",
			"title":    "Unread 1",
			"text":     "First unread notification.",
		})
	n2 := createAndDecode[notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok, map[string]any{
			"severity": "error",
			"title":    "Unread 2",
			"text":     "Second unread notification.",
		})

	// Check project unread count = 2
	uc := getAndDecode[unreadCountResp](t,
		fmt.Sprintf("/api/projects/%s/notifications/unread-count", projID), tok)
	assert.Equal(t, 2, uc.Count)

	// Mark one read
	resp := doAuth(t, "PUT", fmt.Sprintf("/api/notifications/%s/read", n1.ID), tok, nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Check project unread count = 1
	uc2 := getAndDecode[unreadCountResp](t,
		fmt.Sprintf("/api/projects/%s/notifications/unread-count", projID), tok)
	assert.Equal(t, 1, uc2.Count)

	// Also verify the global unread count endpoint includes them
	ucGlobal := getAndDecode[unreadCountResp](t, "/api/notifications/unread-count", tok)
	assert.GreaterOrEqual(t, ucGlobal.Count, 1, "global unread count should include project notifications")

	// Cleanup
	deleteResource(t, fmt.Sprintf("/api/notifications/%s", n1.ID), tok)
	deleteResource(t, fmt.Sprintf("/api/notifications/%s", n2.ID), tok)
}

// ---------------------------------------------------------------------------
// 4. TestNotifications_ReadAll
// ---------------------------------------------------------------------------

func TestNotifications_ReadAll(t *testing.T) {
	tok := adminToken(t)
	projID := createProjectForNotifications(t, tok)

	// Create 3 project notifications
	var ids []string
	for i := 1; i <= 3; i++ {
		n := createAndDecode[notificationResp](t,
			fmt.Sprintf("/api/projects/%s/notifications", projID), tok, map[string]any{
				"severity": "info",
				"title":    fmt.Sprintf("ReadAll %d", i),
				"text":     fmt.Sprintf("Notification %d for read-all test.", i),
			})
		ids = append(ids, n.ID)
	}

	// Verify all are unread
	uc := getAndDecode[unreadCountResp](t,
		fmt.Sprintf("/api/projects/%s/notifications/unread-count", projID), tok)
	assert.Equal(t, 3, uc.Count)

	// Mark all project notifications read
	resp := doAuth(t, "PUT", fmt.Sprintf("/api/projects/%s/notifications/read-all", projID), tok, nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify all have read_at set
	notifs := getAndDecode[[]notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok)
	for _, n := range notifs {
		for _, id := range ids {
			if n.ID == id {
				require.NotNil(t, n.ReadAt, "notification %s should have read_at after read-all", id)
			}
		}
	}

	// Verify unread count is 0
	uc2 := getAndDecode[unreadCountResp](t,
		fmt.Sprintf("/api/projects/%s/notifications/unread-count", projID), tok)
	assert.Equal(t, 0, uc2.Count)

	// Cleanup
	for _, id := range ids {
		deleteResource(t, fmt.Sprintf("/api/notifications/%s", id), tok)
	}
}

// ---------------------------------------------------------------------------
// 5. TestNotifications_Severities
// ---------------------------------------------------------------------------

func TestNotifications_Severities(t *testing.T) {
	tok := adminToken(t)
	projID := createProjectForNotifications(t, tok)

	severities := []string{"info", "success", "warning", "error"}
	createdIDs := make(map[string]string) // severity -> id

	// Create one notification per severity
	for _, sev := range severities {
		n := createAndDecode[notificationResp](t,
			fmt.Sprintf("/api/projects/%s/notifications", projID), tok, map[string]any{
				"severity": sev,
				"title":    "Severity " + sev,
				"text":     "Testing severity level " + sev,
			})
		require.NotEmpty(t, n.ID)
		assert.Equal(t, sev, n.Severity)
		createdIDs[sev] = n.ID
	}

	// List project notifications and verify all severities are present
	notifs := getAndDecode[[]notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok)

	foundSeverities := make(map[string]bool)
	for _, n := range notifs {
		if _, ok := createdIDs[n.Severity]; ok && createdIDs[n.Severity] == n.ID {
			foundSeverities[n.Severity] = true
		}
	}

	for _, sev := range severities {
		assert.True(t, foundSeverities[sev], "severity %q should be present in notification list", sev)
	}

	// Cleanup
	for _, id := range createdIDs {
		deleteResource(t, fmt.Sprintf("/api/notifications/%s", id), tok)
	}
}

// ---------------------------------------------------------------------------
// 6. TestNotifications_DB_Verification
// ---------------------------------------------------------------------------

func TestNotifications_DB_Verification(t *testing.T) {
	tok := adminToken(t)
	pool := testPool(t)
	projID := createProjectForNotifications(t, tok)

	// Create a notification
	created := createAndDecode[notificationResp](t,
		fmt.Sprintf("/api/projects/%s/notifications", projID), tok, map[string]any{
			"severity": "error",
			"title":    "DB Verified Notification",
			"text":     "This should exist in the database.",
		})
	require.NotEmpty(t, created.ID)

	// Verify notification exists in "notifications" table
	require.True(t, rowExists(t, pool, "notifications", "id = $1", created.ID),
		"notification should exist in notifications table")

	// Verify correct values in DB
	title := queryString(t, pool,
		"SELECT title FROM notifications WHERE id = $1", created.ID)
	assert.Equal(t, "DB Verified Notification", title)

	severity := queryString(t, pool,
		"SELECT severity FROM notifications WHERE id = $1", created.ID)
	assert.Equal(t, "error", severity)

	// Delete and verify removal
	deleteResource(t, fmt.Sprintf("/api/notifications/%s", created.ID), tok)

	require.False(t, rowExists(t, pool, "notifications", "id = $1", created.ID),
		"notification should be removed from notifications table after delete")
}
