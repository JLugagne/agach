package server

import (
	"context"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type AuthQueries interface {
	ValidateJWT(ctx context.Context, token string) (identitydomain.Actor, error)
	ValidateDaemonJWT(ctx context.Context, token string) (identitydomain.DaemonActor, error)
	GetUserTeamIDs(ctx context.Context, userID identitydomain.UserID) ([]identitydomain.TeamID, error)
}
