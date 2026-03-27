package commands_test

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

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
	registerFunc            func(ctx context.Context, email, password, displayName string) (domain.User, error)
	loginFunc               func(ctx context.Context, email, password string, rememberMe bool) (string, string, error)
	loginSSOFunc            func(ctx context.Context, provider, idToken, nonce string) (string, string, error)
	refreshTokenFunc        func(ctx context.Context, refreshToken string) (string, error)
	logoutFunc              func(ctx context.Context, token string) error
	updateProfileFunc       func(ctx context.Context, actor domain.Actor, displayName string) (domain.User, error)
	changePasswordFunc      func(ctx context.Context, actor domain.Actor, currentPassword, newPassword string) error
	refreshDaemonTokenFunc  func(ctx context.Context, nodeID domain.NodeID, refreshToken string) (string, error)
	inviteUserFunc          func(ctx context.Context, actor domain.Actor, email string) (string, error)
	completeInviteFunc      func(ctx context.Context, token, displayName, password string) (domain.User, error)
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
