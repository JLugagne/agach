package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type AuthCommands interface {
	Register(ctx context.Context, email, password, displayName string) (domain.User, error)
	Login(ctx context.Context, email, password string, rememberMe bool) (accessToken, refreshToken string, err error)
	LoginSSO(ctx context.Context, provider, code, redirectURI string) (accessToken, refreshToken string, err error)
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
	UpdateProfile(ctx context.Context, actor domain.Actor, displayName string) (domain.User, error)
	ChangePassword(ctx context.Context, actor domain.Actor, currentPassword, newPassword, callerToken string) error
	RefreshDaemonToken(ctx context.Context, nodeID domain.NodeID, refreshToken string) (newAccessToken string, err error)
	InviteUser(ctx context.Context, actor domain.Actor, email string) (inviteToken string, err error)
	CompleteInvite(ctx context.Context, token, displayName, password string) (domain.User, error)
}

type AuthQueries interface {
	ValidateJWT(ctx context.Context, token string) (domain.Actor, error)
	ValidateDaemonJWT(ctx context.Context, token string) (domain.DaemonActor, error)
	GetCurrentUser(ctx context.Context, actor domain.Actor) (domain.User, error)
	GetUserTeamIDs(ctx context.Context, userID domain.UserID) ([]domain.TeamID, error)
}
