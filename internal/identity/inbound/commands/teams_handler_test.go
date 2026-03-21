package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/inbound/commands"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock team services
// ─────────────────────────────────────────────────────────────────────────────

type mockTeamCommands struct {
	createTeamFunc        func(ctx context.Context, actor domain.Actor, name, slug, description string) (domain.Team, error)
	updateTeamFunc        func(ctx context.Context, actor domain.Actor, team domain.Team) error
	deleteTeamFunc        func(ctx context.Context, actor domain.Actor, id domain.TeamID) error
	addUserToTeamFunc     func(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error
	removeUserFromTeamFunc func(ctx context.Context, actor domain.Actor, userID domain.UserID) error
	setUserRoleFunc       func(ctx context.Context, actor domain.Actor, userID domain.UserID, role domain.MemberRole) error
}

func (m *mockTeamCommands) CreateTeam(ctx context.Context, actor domain.Actor, name, slug, description string) (domain.Team, error) {
	return m.createTeamFunc(ctx, actor, name, slug, description)
}
func (m *mockTeamCommands) UpdateTeam(ctx context.Context, actor domain.Actor, team domain.Team) error {
	return m.updateTeamFunc(ctx, actor, team)
}
func (m *mockTeamCommands) DeleteTeam(ctx context.Context, actor domain.Actor, id domain.TeamID) error {
	return m.deleteTeamFunc(ctx, actor, id)
}
func (m *mockTeamCommands) AddUserToTeam(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error {
	return m.addUserToTeamFunc(ctx, actor, userID, teamID)
}
func (m *mockTeamCommands) RemoveUserFromTeam(ctx context.Context, actor domain.Actor, userID domain.UserID) error {
	return m.removeUserFromTeamFunc(ctx, actor, userID)
}
func (m *mockTeamCommands) SetUserRole(ctx context.Context, actor domain.Actor, userID domain.UserID, role domain.MemberRole) error {
	return m.setUserRoleFunc(ctx, actor, userID, role)
}

type mockTeamQueries struct {
	listTeamsFunc       func(ctx context.Context) ([]domain.Team, error)
	getTeamFunc         func(ctx context.Context, id domain.TeamID) (domain.Team, error)
	listUsersFunc       func(ctx context.Context) ([]domain.User, error)
	listTeamMembersFunc func(ctx context.Context, teamID domain.TeamID) ([]domain.User, error)
}

func (m *mockTeamQueries) ListTeams(ctx context.Context) ([]domain.Team, error) {
	return m.listTeamsFunc(ctx)
}
func (m *mockTeamQueries) GetTeam(ctx context.Context, id domain.TeamID) (domain.Team, error) {
	return m.getTeamFunc(ctx, id)
}
func (m *mockTeamQueries) ListUsers(ctx context.Context) ([]domain.User, error) {
	return m.listUsersFunc(ctx)
}
func (m *mockTeamQueries) ListTeamMembers(ctx context.Context, teamID domain.TeamID) ([]domain.User, error) {
	return m.listTeamMembersFunc(ctx, teamID)
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func newTeamsTestHandler(cmds *mockTeamCommands, qrs *mockTeamQueries, authQrs *mockAuthQueries) *mux.Router {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	ctrl := controller.NewController(logger)
	h := commands.NewTeamsHandler(cmds, qrs, authQrs, ctrl)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func adminAuthQueries() *mockAuthQueries {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "admin@example.com", Role: domain.RoleAdmin}
	return &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}
}

func memberAuthQueries() *mockAuthQueries {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "member@example.com", Role: domain.RoleMember}
	return &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}
}

func unauthorizedAuthQueries() *mockAuthQueries {
	return &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ListTeams
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_ListTeams_Success(t *testing.T) {
	teams := []domain.Team{
		{ID: domain.NewTeamID(), Name: "Engineering", Slug: "engineering"},
	}

	qrs := &mockTeamQueries{
		listTeamsFunc: func(_ context.Context) ([]domain.Team, error) {
			return teams, nil
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, adminAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/teams", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
}

// ─────────────────────────────────────────────────────────────────────────────
// CreateTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_CreateTeam_AdminSuccess(t *testing.T) {
	team := domain.Team{ID: domain.NewTeamID(), Name: "Engineering", Slug: "engineering"}

	cmds := &mockTeamCommands{
		createTeamFunc: func(_ context.Context, _ domain.Actor, name, slug, _ string) (domain.Team, error) {
			return team, nil
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"name": "Engineering", "slug": "engineering"})
	req := httptest.NewRequest("POST", "/api/identity/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestTeamsHandler_CreateTeam_NoAuth_ReturnsUnauthorized(t *testing.T) {
	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, unauthorizedAuthQueries())

	body, _ := json.Marshal(map[string]string{"name": "Engineering", "slug": "engineering"})
	req := httptest.NewRequest("POST", "/api/identity/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestTeamsHandler_CreateTeam_Forbidden_ReturnsForbidden(t *testing.T) {
	cmds := &mockTeamCommands{
		createTeamFunc: func(_ context.Context, _ domain.Actor, _, _, _ string) (domain.Team, error) {
			return domain.Team{}, domain.ErrForbidden
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, memberAuthQueries())

	body, _ := json.Marshal(map[string]string{"name": "Engineering", "slug": "engineering"})
	req := httptest.NewRequest("POST", "/api/identity/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer member-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestTeamsHandler_CreateTeam_SlugConflict_ReturnsConflict(t *testing.T) {
	cmds := &mockTeamCommands{
		createTeamFunc: func(_ context.Context, _ domain.Actor, _, _, _ string) (domain.Team, error) {
			return domain.Team{}, domain.ErrTeamSlugConflict
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"name": "Engineering", "slug": "engineering"})
	req := httptest.NewRequest("POST", "/api/identity/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// DeleteTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_DeleteTeam_AdminSuccess(t *testing.T) {
	teamID := domain.NewTeamID()

	cmds := &mockTeamCommands{
		deleteTeamFunc: func(_ context.Context, _ domain.Actor, id domain.TeamID) error {
			return nil
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	req := httptest.NewRequest("DELETE", "/api/identity/teams/"+teamID.String(), nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestTeamsHandler_DeleteTeam_InvalidID_ReturnsBadRequest(t *testing.T) {
	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, adminAuthQueries())

	req := httptest.NewRequest("DELETE", "/api/identity/teams/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTeamsHandler_DeleteTeam_Forbidden_ReturnsForbidden(t *testing.T) {
	teamID := domain.NewTeamID()

	cmds := &mockTeamCommands{
		deleteTeamFunc: func(_ context.Context, _ domain.Actor, _ domain.TeamID) error {
			return domain.ErrForbidden
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	req := httptest.NewRequest("DELETE", "/api/identity/teams/"+teamID.String(), nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestTeamsHandler_DeleteTeam_NotFound_ReturnsNotFound(t *testing.T) {
	teamID := domain.NewTeamID()

	cmds := &mockTeamCommands{
		deleteTeamFunc: func(_ context.Context, _ domain.Actor, _ domain.TeamID) error {
			return domain.ErrTeamNotFound
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	req := httptest.NewRequest("DELETE", "/api/identity/teams/"+teamID.String(), nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// ListUsers
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_ListUsers_Success(t *testing.T) {
	users := []domain.User{
		{ID: domain.NewUserID(), Email: "a@example.com"},
		{ID: domain.NewUserID(), Email: "b@example.com"},
	}

	qrs := &mockTeamQueries{
		listUsersFunc: func(_ context.Context) ([]domain.User, error) {
			return users, nil
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, adminAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/users", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	data, _ := resp["data"].([]interface{})
	assert.Len(t, data, 2)
}

// ─────────────────────────────────────────────────────────────────────────────
// SetUserTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_SetUserTeam_AdminSuccess(t *testing.T) {
	userID := domain.NewUserID()
	teamID := domain.NewTeamID()

	cmds := &mockTeamCommands{
		addUserToTeamFunc: func(_ context.Context, _ domain.Actor, _ domain.UserID, _ domain.TeamID) error {
			return nil
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"team_id": teamID.String()})
	req := httptest.NewRequest("PUT", "/api/identity/users/"+userID.String()+"/team", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestTeamsHandler_SetUserTeam_InvalidUserID_ReturnsBadRequest(t *testing.T) {
	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"team_id": domain.NewTeamID().String()})
	req := httptest.NewRequest("PUT", "/api/identity/users/not-a-uuid/team", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTeamsHandler_SetUserTeam_InvalidTeamID_ReturnsBadRequest(t *testing.T) {
	userID := domain.NewUserID()
	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"team_id": "not-a-uuid"})
	req := httptest.NewRequest("PUT", "/api/identity/users/"+userID.String()+"/team", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTeamsHandler_SetUserTeam_UserNotFound_ReturnsNotFound(t *testing.T) {
	userID := domain.NewUserID()
	teamID := domain.NewTeamID()

	cmds := &mockTeamCommands{
		addUserToTeamFunc: func(_ context.Context, _ domain.Actor, _ domain.UserID, _ domain.TeamID) error {
			return domain.ErrUserNotFound
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"team_id": teamID.String()})
	req := httptest.NewRequest("PUT", "/api/identity/users/"+userID.String()+"/team", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// RemoveUserFromTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_RemoveUserFromTeam_AdminSuccess(t *testing.T) {
	userID := domain.NewUserID()

	cmds := &mockTeamCommands{
		removeUserFromTeamFunc: func(_ context.Context, _ domain.Actor, _ domain.UserID) error {
			return nil
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	req := httptest.NewRequest("DELETE", "/api/identity/users/"+userID.String()+"/team", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestTeamsHandler_RemoveUserFromTeam_InvalidUserID_ReturnsBadRequest(t *testing.T) {
	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, adminAuthQueries())

	req := httptest.NewRequest("DELETE", "/api/identity/users/not-uuid/team", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTeamsHandler_RemoveUserFromTeam_Forbidden_ReturnsForbidden(t *testing.T) {
	userID := domain.NewUserID()

	cmds := &mockTeamCommands{
		removeUserFromTeamFunc: func(_ context.Context, _ domain.Actor, _ domain.UserID) error {
			return domain.ErrForbidden
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, memberAuthQueries())

	req := httptest.NewRequest("DELETE", "/api/identity/users/"+userID.String()+"/team", nil)
	req.Header.Set("Authorization", "Bearer member-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// SetUserRole
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_SetUserRole_AdminSuccess(t *testing.T) {
	userID := domain.NewUserID()

	cmds := &mockTeamCommands{
		setUserRoleFunc: func(_ context.Context, _ domain.Actor, _ domain.UserID, _ domain.MemberRole) error {
			return nil
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req := httptest.NewRequest("PUT", "/api/identity/users/"+userID.String()+"/role", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestTeamsHandler_SetUserRole_InvalidUserID_ReturnsBadRequest(t *testing.T) {
	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req := httptest.NewRequest("PUT", "/api/identity/users/not-uuid/role", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTeamsHandler_SetUserRole_InvalidRole_ReturnsBadRequest(t *testing.T) {
	userID := domain.NewUserID()
	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"role": "superadmin"})
	req := httptest.NewRequest("PUT", "/api/identity/users/"+userID.String()+"/role", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTeamsHandler_SetUserRole_UserNotFound_ReturnsNotFound(t *testing.T) {
	userID := domain.NewUserID()

	cmds := &mockTeamCommands{
		setUserRoleFunc: func(_ context.Context, _ domain.Actor, _ domain.UserID, _ domain.MemberRole) error {
			return domain.ErrUserNotFound
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"role": "member"})
	req := httptest.NewRequest("PUT", "/api/identity/users/"+userID.String()+"/role", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// TeamsHandler auth: API key header
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_APIKeyAuth_Success(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleAdmin}
	team := domain.Team{ID: domain.NewTeamID(), Name: "Engineering", Slug: "engineering"}

	cmds := &mockTeamCommands{
		createTeamFunc: func(_ context.Context, _ domain.Actor, _, _, _ string) (domain.Team, error) {
			return team, nil
		},
	}

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
		validateAPIKeyFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, authQrs)

	body, _ := json.Marshal(map[string]string{"name": "Engineering", "slug": "engineering"})
	req := httptest.NewRequest("POST", "/api/identity/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", "agach_somevalidapikey")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// handleTeamError - missing branch (ErrTeamNotFound via SetUserTeam)
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_SetUserTeam_TeamNotFound_ReturnsNotFound(t *testing.T) {
	userID := domain.NewUserID()
	teamID := domain.NewTeamID()

	cmds := &mockTeamCommands{
		addUserToTeamFunc: func(_ context.Context, _ domain.Actor, _ domain.UserID, _ domain.TeamID) error {
			return domain.ErrTeamNotFound
		},
	}

	router := newTeamsTestHandler(cmds, &mockTeamQueries{}, adminAuthQueries())

	body, _ := json.Marshal(map[string]string{"team_id": teamID.String()})
	req := httptest.NewRequest("PUT", "/api/identity/users/"+userID.String()+"/team", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	errData, _ := resp["error"].(map[string]interface{})
	assert.Equal(t, "TEAM_NOT_FOUND", errData["code"])
}

// ─────────────────────────────────────────────────────────────────────────────
// userToPublicMap - user with team_id set
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_ListUsers_UserWithTeamID(t *testing.T) {
	teamID := domain.NewTeamID()
	users := []domain.User{
		{ID: domain.NewUserID(), Email: "a@example.com", TeamID: &teamID},
	}

	qrs := &mockTeamQueries{
		listUsersFunc: func(_ context.Context) ([]domain.User, error) {
			return users, nil
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, adminAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/users", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	data, _ := resp["data"].([]interface{})
	require.Len(t, data, 1)
	user, _ := data[0].(map[string]interface{})
	assert.NotNil(t, user["team_id"], "team_id should be set")
}

// ─────────────────────────────────────────────────────────────────────────────
// ListTeams / ListUsers error branches
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamsHandler_ListTeams_ServiceError_ReturnsInternalError(t *testing.T) {
	qrs := &mockTeamQueries{
		listTeamsFunc: func(_ context.Context) ([]domain.Team, error) {
			return nil, domain.ErrTeamNotFound
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, adminAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/teams", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTeamsHandler_ListUsers_ServiceError_ReturnsInternalError(t *testing.T) {
	qrs := &mockTeamQueries{
		listUsersFunc: func(_ context.Context) ([]domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, qrs, adminAuthQueries())

	req := httptest.NewRequest("GET", "/api/identity/users", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTeamsHandler_APIKeyAuth_InvalidKey_ReturnsUnauthorized(t *testing.T) {
	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
		validateAPIKeyFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrAPIKeyInvalid
		},
	}

	router := newTeamsTestHandler(&mockTeamCommands{}, &mockTeamQueries{}, authQrs)

	body, _ := json.Marshal(map[string]string{"name": "Engineering", "slug": "engineering"})
	req := httptest.NewRequest("POST", "/api/identity/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", "bad-api-key")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
