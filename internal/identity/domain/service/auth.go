package service

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type AuthCommands interface {
	Register(ctx context.Context, email, password, displayName string) (domain.User, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error)
	LoginSSO(ctx context.Context, provider, code, redirectURI string) (accessToken, refreshToken string, err error)
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
	CreateAPIKey(ctx context.Context, actor domain.Actor, name string, scopes []string, expiresAt *time.Time) (key domain.APIKey, rawKey string, err error)
	RevokeAPIKey(ctx context.Context, actor domain.Actor, keyID domain.APIKeyID) error
	UpdateProfile(ctx context.Context, actor domain.Actor, displayName string) (domain.User, error)
	ChangePassword(ctx context.Context, actor domain.Actor, currentPassword, newPassword string) error
}

type AuthQueries interface {
	ValidateJWT(ctx context.Context, token string) (domain.Actor, error)
	ValidateAPIKey(ctx context.Context, rawKey string) (domain.Actor, error)
	ListAPIKeys(ctx context.Context, actor domain.Actor) ([]domain.APIKey, error)
	GetCurrentUser(ctx context.Context, actor domain.Actor) (domain.User, error)
}
