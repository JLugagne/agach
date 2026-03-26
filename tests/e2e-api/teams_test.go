package e2eapi

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Inline types for decoding team/user API responses.
// ---------------------------------------------------------------------------

type teamResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type userResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	TeamIDs     []string  `json:"team_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func createTeam(t *testing.T, token string, body map[string]any) teamResponse {
	t.Helper()
	return createAndDecode[teamResponse](t, "/api/identity/teams", token, body)
}

func deleteTeam(t *testing.T, token, id string) {
	t.Helper()
	resp := doAuth(t, "DELETE", fmt.Sprintf("/api/identity/teams/%s", id), token, nil)
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()
}

func listUsers(t *testing.T, token string) []userResponse {
	t.Helper()
	return getAndDecode[[]userResponse](t, "/api/identity/users", token)
}

func findAdminUser(t *testing.T, token string) userResponse {
	t.Helper()
	users := listUsers(t, token)
	for _, u := range users {
		if u.Role == "admin" {
			return u
		}
	}
	t.Fatal("no admin user found")
	return userResponse{}
}

func findUserByID(t *testing.T, token, id string) userResponse {
	t.Helper()
	users := listUsers(t, token)
	for _, u := range users {
		if u.ID == id {
			return u
		}
	}
	t.Fatalf("user %s not found", id)
	return userResponse{}
}

func addUserToTeam(t *testing.T, token, userID, teamID string) {
	t.Helper()
	resp := doAuth(t, "PUT", fmt.Sprintf("/api/identity/users/%s/team", userID), token, map[string]any{
		"team_id": teamID,
	})
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()
}

func removeUserFromTeam(t *testing.T, token, userID, teamID string) {
	t.Helper()
	resp := doAuth(t, "DELETE", fmt.Sprintf("/api/identity/users/%s/team", userID), token, map[string]any{
		"team_id": teamID,
	})
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestTeams_CRUD(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	slug := uniqueSlug("e2e-team")

	// Create.
	team := createTeam(t, token, map[string]any{
		"name":        "E2E CRUD Team",
		"slug":        slug,
		"description": "Created by e2e test",
	})
	require.NotEmpty(t, team.ID)
	require.Equal(t, "E2E CRUD Team", team.Name)
	require.Equal(t, slug, team.Slug)
	require.Equal(t, "Created by e2e test", team.Description)

	// List: team should appear.
	teams := getAndDecode[[]teamResponse](t, "/api/identity/teams", token)
	found := false
	for _, tm := range teams {
		if tm.ID == team.ID {
			found = true
			require.Equal(t, team.Name, tm.Name)
			require.Equal(t, team.Slug, tm.Slug)
			break
		}
	}
	require.True(t, found, "created team should appear in list")

	// Delete.
	deleteTeam(t, token, team.ID)

	// Verify gone from list.
	teamsAfter := getAndDecode[[]teamResponse](t, "/api/identity/teams", token)
	for _, tm := range teamsAfter {
		require.NotEqual(t, team.ID, tm.ID, "deleted team should not appear in list")
	}
}

func TestTeams_AssignUserToTeam(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	// Create a team.
	slug := uniqueSlug("e2e-assign")
	team := createTeam(t, token, map[string]any{
		"name":        "E2E Assign Team",
		"slug":        slug,
		"description": "team for assignment test",
	})
	t.Cleanup(func() { deleteTeam(t, token, team.ID) })

	// Get admin user ID.
	admin := findAdminUser(t, token)

	// Assign admin to team.
	addUserToTeam(t, token, admin.ID, team.ID)

	// Verify via list users.
	user := findUserByID(t, token, admin.ID)
	require.Contains(t, user.TeamIDs, team.ID, "admin should be in the team")

	// Remove from team.
	removeUserFromTeam(t, token, admin.ID, team.ID)

	// Verify team_ids no longer contains the team.
	userAfter := findUserByID(t, token, admin.ID)
	require.NotContains(t, userAfter.TeamIDs, team.ID, "admin should not be in the team after removal")
}

func TestTeams_UserInMultipleTeams(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	// Create two teams.
	teamA := createTeam(t, token, map[string]any{
		"name": "Multi Team A",
		"slug": uniqueSlug("e2e-multi-a"),
	})
	t.Cleanup(func() { deleteTeam(t, token, teamA.ID) })

	teamB := createTeam(t, token, map[string]any{
		"name": "Multi Team B",
		"slug": uniqueSlug("e2e-multi-b"),
	})
	t.Cleanup(func() { deleteTeam(t, token, teamB.ID) })

	admin := findAdminUser(t, token)

	// Assign to both teams.
	addUserToTeam(t, token, admin.ID, teamA.ID)
	addUserToTeam(t, token, admin.ID, teamB.ID)

	// Verify user is in both teams.
	user := findUserByID(t, token, admin.ID)
	require.Contains(t, user.TeamIDs, teamA.ID, "user should be in team A")
	require.Contains(t, user.TeamIDs, teamB.ID, "user should be in team B")

	// Remove from team A only.
	removeUserFromTeam(t, token, admin.ID, teamA.ID)

	// Verify still in B but not A.
	userAfter := findUserByID(t, token, admin.ID)
	require.NotContains(t, userAfter.TeamIDs, teamA.ID, "user should not be in team A after removal")
	require.Contains(t, userAfter.TeamIDs, teamB.ID, "user should still be in team B")

	// Cleanup.
	removeUserFromTeam(t, token, admin.ID, teamB.ID)
}

func TestTeams_DuplicateAssignmentIsIdempotent(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	team := createTeam(t, token, map[string]any{
		"name": "Idempotent Team",
		"slug": uniqueSlug("e2e-idempotent"),
	})
	t.Cleanup(func() { deleteTeam(t, token, team.ID) })

	admin := findAdminUser(t, token)

	// Assign twice — should not error.
	addUserToTeam(t, token, admin.ID, team.ID)
	addUserToTeam(t, token, admin.ID, team.ID)

	// Verify user appears exactly once with this team.
	user := findUserByID(t, token, admin.ID)
	count := 0
	for _, tid := range user.TeamIDs {
		if tid == team.ID {
			count++
		}
	}
	require.Equal(t, 1, count, "team should appear exactly once in team_ids")

	// Cleanup.
	removeUserFromTeam(t, token, admin.ID, team.ID)
}

func TestTeams_DeleteTeamRemovesMemberships(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	team := createTeam(t, token, map[string]any{
		"name": "Ephemeral Team",
		"slug": uniqueSlug("e2e-ephemeral"),
	})

	admin := findAdminUser(t, token)
	addUserToTeam(t, token, admin.ID, team.ID)

	// Verify assigned.
	user := findUserByID(t, token, admin.ID)
	require.Contains(t, user.TeamIDs, team.ID)

	// Delete the team.
	deleteTeam(t, token, team.ID)

	// Verify team_ids no longer contains the deleted team (CASCADE).
	userAfter := findUserByID(t, token, admin.ID)
	require.NotContains(t, userAfter.TeamIDs, team.ID, "deleted team should be gone from user's team_ids")
}

func TestTeams_SetUserRole(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)

	// Get admin user ID.
	admin := findAdminUser(t, token)

	// Set role to member.
	resp := doAuth(t, "PUT", fmt.Sprintf("/api/identity/users/%s/role", admin.ID), token, map[string]any{
		"role": "member",
	})
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Verify role changed.
	users := listUsers(t, token)
	for _, u := range users {
		if u.ID == admin.ID {
			require.Equal(t, "member", u.Role)
			break
		}
	}

	// Restore to admin.
	resp = doAuth(t, "PUT", fmt.Sprintf("/api/identity/users/%s/role", admin.ID), token, map[string]any{
		"role": "admin",
	})
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Verify restored.
	usersAfter := listUsers(t, token)
	for _, u := range usersAfter {
		if u.ID == admin.ID {
			require.Equal(t, "admin", u.Role)
			break
		}
	}
}

func TestTeams_DB_Verification(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)
	pool := testPool(t)

	slug := uniqueSlug("e2e-dbteam")

	team := createTeam(t, token, map[string]any{
		"name":        "E2E DB Team",
		"slug":        slug,
		"description": "verify row in pg",
	})
	t.Cleanup(func() { deleteTeam(t, token, team.ID) })

	// Verify team exists in the teams table.
	require.True(t, rowExists(t, pool, "teams", "id = $1", team.ID),
		"team should exist in teams table")

	// Verify name matches.
	name := queryString(t, pool, "SELECT name FROM teams WHERE id = $1", team.ID)
	require.Equal(t, "E2E DB Team", name)

	// Verify slug.
	gotSlug := queryString(t, pool, "SELECT slug FROM teams WHERE id = $1", team.ID)
	require.Equal(t, slug, gotSlug)

	// Verify description.
	desc := queryString(t, pool, "SELECT description FROM teams WHERE id = $1", team.ID)
	require.Equal(t, "verify row in pg", desc)
}

func TestTeams_DB_TeamMembers(t *testing.T) {
	ensureServer(t)
	token := adminToken(t)
	pool := testPool(t)

	team := createTeam(t, token, map[string]any{
		"name": "DB Members Team",
		"slug": uniqueSlug("e2e-dbmembers"),
	})
	t.Cleanup(func() { deleteTeam(t, token, team.ID) })

	admin := findAdminUser(t, token)
	addUserToTeam(t, token, admin.ID, team.ID)
	t.Cleanup(func() { removeUserFromTeam(t, token, admin.ID, team.ID) })

	// Verify row in team_members table.
	require.True(t, rowExists(t, pool, "team_members", "team_id = $1 AND user_id = $2", team.ID, admin.ID),
		"team_members row should exist after assignment")

	// Remove and verify row is gone.
	removeUserFromTeam(t, token, admin.ID, team.ID)
	require.False(t, rowExists(t, pool, "team_members", "team_id = $1 AND user_id = $2", team.ID, admin.ID),
		"team_members row should be gone after removal")
}
