package app

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodes"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost        = 12
	minPasswordLen    = 8
	minSecretLen      = 32
	accessTokenTTL    = 15 * time.Minute
	refreshTokenTTL         = 7 * 24 * time.Hour
	refreshTokenTTLRemember = 30 * 24 * time.Hour
	jwtClaimTokenType = "token_type"
	tokenTypeAccess   = "access"
	tokenTypeRefresh  = "refresh"
)

var tokenBlocklist sync.Map

type authService struct {
	users  users.UserRepository
	nodes  nodes.NodeRepository
	secret []byte
	sso    *SSOService
}

// NewAuthService creates an auth service backed by the provided repositories.
// secret must be at least 32 bytes. sso may be nil if SSO is not configured.
// nodes may be nil; when nil, daemon token validation skips revocation checks.
func NewAuthService(u users.UserRepository, secret []byte, sso *SSOService) service.AuthCommands {
	return &authService{users: u, secret: secret, sso: sso}
}

// NewAuthServiceWithNodes creates an auth service with nodes repository support for daemon tokens.
func NewAuthServiceWithNodes(u users.UserRepository, n nodes.NodeRepository, secret []byte, sso *SSOService) service.AuthCommands {
	return &authService{users: u, nodes: n, secret: secret, sso: sso}
}

// NewAuthQueriesService returns the queries-side of the same auth service.
// sso may be nil if SSO is not configured.
func NewAuthQueriesService(u users.UserRepository, secret []byte, sso *SSOService) service.AuthQueries {
	return &authService{users: u, secret: secret, sso: sso}
}

// NewAuthQueriesServiceWithNodes returns the queries-side of the auth service with nodes repository support.
func NewAuthQueriesServiceWithNodes(u users.UserRepository, n nodes.NodeRepository, secret []byte, sso *SSOService) service.AuthQueries {
	return &authService{users: u, nodes: n, secret: secret, sso: sso}
}

var (
	_ service.AuthCommands = (*authService)(nil)
	_ service.AuthQueries  = (*authService)(nil)
)

func (s *authService) Register(ctx context.Context, email, password, displayName string) (domain.User, error) {
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		return domain.User{}, &domain.Error{
			Code:    "INVALID_EMAIL",
			Message: "invalid email address",
		}
	}

	if len(password) < minPasswordLen {
		return domain.User{}, &domain.Error{
			Code:    "PASSWORD_TOO_SHORT",
			Message: fmt.Sprintf("password must be at least %d characters", minPasswordLen),
		}
	}

	if strings.TrimSpace(password) == "" {
		return domain.User{}, &domain.Error{
			Code:    "PASSWORD_WHITESPACE",
			Message: "password must not consist entirely of whitespace",
		}
	}

	_, err := s.users.FindByEmail(ctx, email)
	if err == nil {
		return domain.User{}, domain.ErrEmailAlreadyExists
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := domain.User{
		ID:           domain.NewUserID(),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: string(hash),
		Role:         domain.RoleMember,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string, rememberMe bool) (accessToken, refreshToken string, err error) {
	if len(s.secret) < minSecretLen {
		return "", "", &domain.Error{
			Code:    "INSECURE_JWT_SECRET",
			Message: fmt.Sprintf("JWT secret must be at least %d bytes", minSecretLen),
		}
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", err
	}

	if user.PasswordHash == "" {
		return "", "", domain.ErrSSOUserNoPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	now := time.Now()
	user.UpdatedAt = now
	_ = s.users.Update(ctx, user)

	accessToken, err = s.issueToken(user, tokenTypeAccess, accessTokenTTL)
	if err != nil {
		return "", "", err
	}
	rttl := refreshTokenTTL
	if rememberMe {
		rttl = refreshTokenTTLRemember
	}
	refreshToken, err = s.issueToken(user, tokenTypeRefresh, rttl)
	return accessToken, refreshToken, err
}

func (s *authService) LoginSSO(ctx context.Context, provider, code, redirectURI string) (string, string, error) {
	if s.sso == nil {
		return "", "", &domain.Error{Code: "SSO_NOT_CONFIGURED", Message: "SSO is not configured"}
	}
	return s.sso.LoginSSO(ctx, provider, code, redirectURI)
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	tokenType, _ := claims[jwtClaimTokenType].(string)
	if tokenType != tokenTypeRefresh {
		return "", domain.ErrUnauthorized
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	userID, err := domain.ParseUserID(sub)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrUnauthorized
		}
		return "", err
	}

	return s.issueToken(user, tokenTypeAccess, accessTokenTTL)
}

func (s *authService) Logout(_ context.Context, token string) error {
	if token != "" {
		tokenBlocklist.Store(token, true)
	}
	return nil
}

func (s *authService) ValidateJWT(ctx context.Context, token string) (domain.Actor, error) {
	if _, blocked := tokenBlocklist.Load(token); blocked {
		return domain.Actor{}, domain.ErrUnauthorized
	}

	claims, err := s.parseToken(token)
	if err != nil {
		return domain.Actor{}, domain.ErrUnauthorized
	}

	tokenType, _ := claims[jwtClaimTokenType].(string)
	if tokenType != tokenTypeAccess {
		return domain.Actor{}, domain.ErrUnauthorized
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return domain.Actor{}, domain.ErrUnauthorized
	}

	userID, err := domain.ParseUserID(sub)
	if err != nil {
		return domain.Actor{}, domain.ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.Actor{}, domain.ErrUnauthorized
		}
		return domain.Actor{}, err
	}

	return domain.Actor{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	}, nil
}

func (s *authService) GetCurrentUser(ctx context.Context, actor domain.Actor) (domain.User, error) {
	return s.users.FindByID(ctx, actor.UserID)
}

func (s *authService) UpdateProfile(ctx context.Context, actor domain.Actor, displayName string) (domain.User, error) {
	user, err := s.users.FindByID(ctx, actor.UserID)
	if err != nil {
		return domain.User{}, err
	}

	user.DisplayName = displayName
	user.UpdatedAt = time.Now()

	if err := s.users.Update(ctx, user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *authService) ChangePassword(ctx context.Context, actor domain.Actor, currentPassword, newPassword string) error {
	if len(newPassword) < minPasswordLen {
		return &domain.Error{
			Code:    "PASSWORD_TOO_SHORT",
			Message: fmt.Sprintf("password must be at least %d characters", minPasswordLen),
		}
	}

	if strings.TrimSpace(newPassword) == "" {
		return &domain.Error{
			Code:    "PASSWORD_WHITESPACE",
			Message: "password must not consist entirely of whitespace",
		}
	}

	user, err := s.users.FindByID(ctx, actor.UserID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	return s.users.Update(ctx, user)
}

func (s *authService) ValidateDaemonJWT(ctx context.Context, token string) (domain.DaemonActor, error) {
	if _, blocked := tokenBlocklist.Load(token); blocked {
		return domain.DaemonActor{}, domain.ErrUnauthorized
	}

	claims, err := s.parseToken(token)
	if err != nil {
		return domain.DaemonActor{}, domain.ErrUnauthorized
	}

	tokenType, _ := claims[jwtClaimTokenType].(string)
	if tokenType != "daemon" {
		return domain.DaemonActor{}, domain.ErrUnauthorized
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return domain.DaemonActor{}, domain.ErrUnauthorized
	}

	nodeID, err := domain.ParseNodeID(sub)
	if err != nil {
		return domain.DaemonActor{}, domain.ErrUnauthorized
	}

	ownerIDStr, _ := claims["owner_id"].(string)
	ownerID, err := domain.ParseUserID(ownerIDStr)
	if err != nil {
		return domain.DaemonActor{}, domain.ErrUnauthorized
	}

	modeStr, _ := claims["mode"].(string)
	mode := domain.NodeMode(modeStr)

	if s.nodes != nil {
		node, err := s.nodes.FindByID(ctx, nodeID)
		if err != nil {
			return domain.DaemonActor{}, domain.ErrUnauthorized
		}
		if node.IsRevoked() {
			return domain.DaemonActor{}, domain.ErrNodeRevoked
		}
		go s.nodes.UpdateLastSeen(context.Background(), nodeID)
	}

	return domain.DaemonActor{
		NodeID:      nodeID,
		OwnerUserID: ownerID,
		Mode:        mode,
	}, nil
}

func (s *authService) RefreshDaemonToken(ctx context.Context, nodeID domain.NodeID, refreshToken string) (string, error) {
	if s.nodes == nil {
		return "", domain.ErrUnauthorized
	}

	node, err := s.nodes.FindByID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, domain.ErrNodeNotFound) {
			return "", domain.ErrUnauthorized
		}
		return "", err
	}

	if node.IsRevoked() {
		return "", domain.ErrNodeRevoked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(node.RefreshTokenHash), []byte(refreshToken)); err != nil {
		return "", domain.ErrUnauthorized
	}

	return issueDaemonToken(node, accessTokenTTL, s.secret)
}

func (s *authService) issueToken(user domain.User, tokenType string, ttl time.Duration) (string, error) {
	return issueToken(user, tokenType, ttl, s.secret)
}

func (s *authService) parseToken(tokenStr string) (jwt.MapClaims, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !t.Valid {
		return nil, domain.ErrUnauthorized
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return claims, nil
}

