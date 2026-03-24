package app

import (
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
)

func issueToken(user domain.User, tokenType string, ttl time.Duration, secret []byte) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":             user.ID.String(),
		"email":           user.Email,
		"role":            string(user.Role),
		jwtClaimTokenType: tokenType,
		"iat":             now.Unix(),
		"exp":             now.Add(ttl).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

func issueDaemonToken(node domain.Node, ttl time.Duration, secret []byte) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":             node.ID.String(),
		"owner_id":        node.OwnerUserID.String(),
		"mode":            string(node.Mode),
		jwtClaimTokenType: "daemon",
		"iat":             now.Unix(),
		"exp":             now.Add(ttl).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}
