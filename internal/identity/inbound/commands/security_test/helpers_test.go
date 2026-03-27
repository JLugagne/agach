package security_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock types (inlined from parent package test files)
// ─────────────────────────────────────────────────────────────────────────────

// mockAuthQueries is a shared mock for AuthQueries across all command handler tests.
type mockAuthQueries struct {
	validateJWTFunc       func(ctx context.Context, token string) (domain.Actor, error)
	validateDaemonJWTFunc func(ctx context.Context, token string) (domain.DaemonActor, error)
	getCurrentUserFunc    func(ctx context.Context, actor domain.Actor) (domain.User, error)
}

func (m *mockAuthQueries) ValidateJWT(ctx context.Context, token string) (domain.Actor, error) {
	if m.validateJWTFunc != nil {
		return m.validateJWTFunc(ctx, token)
	}
	return domain.Actor{}, nil
}

func (m *mockAuthQueries) ValidateDaemonJWT(ctx context.Context, token string) (domain.DaemonActor, error) {
	if m.validateDaemonJWTFunc != nil {
		return m.validateDaemonJWTFunc(ctx, token)
	}
	return domain.DaemonActor{}, nil
}

func (m *mockAuthQueries) GetCurrentUser(ctx context.Context, actor domain.Actor) (domain.User, error) {
	if m.getCurrentUserFunc != nil {
		return m.getCurrentUserFunc(ctx, actor)
	}
	return domain.User{}, nil
}

func (m *mockAuthQueries) GetUserTeamIDs(ctx context.Context, userID domain.UserID) ([]domain.TeamID, error) {
	return nil, nil
}

// mockAuthCommands is a shared mock for AuthCommands across all command handler tests.
type mockAuthCommands struct {
	registerFunc           func(ctx context.Context, email, password, displayName string) (domain.User, error)
	loginFunc              func(ctx context.Context, email, password string, rememberMe bool) (string, string, error)
	loginSSOFunc           func(ctx context.Context, provider, idToken, nonce string) (string, string, error)
	refreshTokenFunc       func(ctx context.Context, refreshToken string) (string, error)
	logoutFunc             func(ctx context.Context, token string) error
	updateProfileFunc      func(ctx context.Context, actor domain.Actor, displayName string) (domain.User, error)
	changePasswordFunc     func(ctx context.Context, actor domain.Actor, currentPassword, newPassword string) error
	refreshDaemonTokenFunc func(ctx context.Context, nodeID domain.NodeID, refreshToken string) (string, error)
	inviteUserFunc         func(ctx context.Context, actor domain.Actor, email string) (string, error)
	completeInviteFunc     func(ctx context.Context, token, displayName, password string) (domain.User, error)
}

func (m *mockAuthCommands) Register(ctx context.Context, email, password, displayName string) (domain.User, error) {
	return m.registerFunc(ctx, email, password, displayName)
}

func (m *mockAuthCommands) Login(ctx context.Context, email, password string, rememberMe bool) (string, string, error) {
	return m.loginFunc(ctx, email, password, rememberMe)
}

func (m *mockAuthCommands) LoginSSO(ctx context.Context, provider, idToken, nonce string) (string, string, error) {
	if m.loginSSOFunc != nil {
		return m.loginSSOFunc(ctx, provider, idToken, nonce)
	}
	return "", "", nil
}

func (m *mockAuthCommands) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	return m.refreshTokenFunc(ctx, refreshToken)
}

func (m *mockAuthCommands) Logout(ctx context.Context, token string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, token)
	}
	return nil
}

func (m *mockAuthCommands) UpdateProfile(ctx context.Context, actor domain.Actor, displayName string) (domain.User, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, actor, displayName)
	}
	return domain.User{}, nil
}

func (m *mockAuthCommands) ChangePassword(ctx context.Context, actor domain.Actor, currentPassword, newPassword, callerToken string) error {
	if m.changePasswordFunc != nil {
		return m.changePasswordFunc(ctx, actor, currentPassword, newPassword)
	}
	return nil
}

func (m *mockAuthCommands) RefreshDaemonToken(ctx context.Context, nodeID domain.NodeID, refreshToken string) (string, error) {
	if m.refreshDaemonTokenFunc != nil {
		return m.refreshDaemonTokenFunc(ctx, nodeID, refreshToken)
	}
	return "", nil
}

func (m *mockAuthCommands) InviteUser(ctx context.Context, actor domain.Actor, email string) (string, error) {
	if m.inviteUserFunc != nil {
		return m.inviteUserFunc(ctx, actor, email)
	}
	return "", nil
}

func (m *mockAuthCommands) CompleteInvite(ctx context.Context, token, displayName, password string) (domain.User, error) {
	if m.completeInviteFunc != nil {
		return m.completeInviteFunc(ctx, token, displayName, password)
	}
	return domain.User{}, nil
}

// mockTeamCommands is a mock for TeamCommands.
type mockTeamCommands struct {
	createTeamFunc         func(ctx context.Context, actor domain.Actor, name, slug, description string) (domain.Team, error)
	updateTeamFunc         func(ctx context.Context, actor domain.Actor, team domain.Team) error
	deleteTeamFunc         func(ctx context.Context, actor domain.Actor, id domain.TeamID) error
	addUserToTeamFunc      func(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error
	removeUserFromTeamFunc func(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error
	setUserRoleFunc        func(ctx context.Context, actor domain.Actor, userID domain.UserID, role domain.MemberRole) error
	blockUserFunc          func(ctx context.Context, actor domain.Actor, userID domain.UserID) error
	unblockUserFunc        func(ctx context.Context, actor domain.Actor, userID domain.UserID) error
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
func (m *mockTeamCommands) RemoveUserFromTeam(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error {
	return m.removeUserFromTeamFunc(ctx, actor, userID, teamID)
}
func (m *mockTeamCommands) SetUserRole(ctx context.Context, actor domain.Actor, userID domain.UserID, role domain.MemberRole) error {
	return m.setUserRoleFunc(ctx, actor, userID, role)
}
func (m *mockTeamCommands) BlockUser(ctx context.Context, actor domain.Actor, userID domain.UserID) error {
	if m.blockUserFunc == nil {
		return nil
	}
	return m.blockUserFunc(ctx, actor, userID)
}
func (m *mockTeamCommands) UnblockUser(ctx context.Context, actor domain.Actor, userID domain.UserID) error {
	if m.unblockUserFunc == nil {
		return nil
	}
	return m.unblockUserFunc(ctx, actor, userID)
}

// mockTeamQueries is a mock for TeamQueries.
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
// Helper constructors
// ─────────────────────────────────────────────────────────────────────────────

func newTestHandler(cmds *mockAuthCommands, qrs *mockAuthQueries) (*commands.AuthCommandsHandler, *mux.Router) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	ctrl := controller.NewController(logger)
	h := commands.NewAuthCommandsHandler(cmds, qrs, ctrl, 0, 0)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return h, r
}

func postJSON(router *mux.Router, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func newTeamsTestHandler(cmds *mockTeamCommands, qrs *mockTeamQueries, authQrs *mockAuthQueries) *mux.Router {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	ctrl := controller.NewController(logger)
	h := commands.NewTeamsHandler(cmds, qrs, authQrs, nil, ctrl)
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

// Ensure unused imports are used.
var _ http.Handler = (*mux.Router)(nil)
