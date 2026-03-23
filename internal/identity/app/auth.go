package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/apikeys"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost        = 12
	minPasswordLen    = 8
	minSecretLen      = 32
	apiKeyPrefix      = "agach_"
	accessTokenTTL    = 15 * time.Minute
	refreshTokenTTL   = 7 * 24 * time.Hour
	jwtClaimTokenType = "token_type"
	tokenTypeAccess   = "access"
	tokenTypeRefresh  = "refresh"
)

var allowedScopes = map[string]bool{
	"kanban:read":  true,
	"kanban:write": true,
}

var tokenBlocklist sync.Map

type authService struct {
	users   users.UserRepository
	apikeys apikeys.APIKeyRepository
	secret  []byte
	sso     *SSOService
}

// NewAuthService creates an auth service backed by the provided repositories.
// secret must be at least 32 bytes. sso may be nil if SSO is not configured.
func NewAuthService(u users.UserRepository, k apikeys.APIKeyRepository, secret []byte, sso *SSOService) service.AuthCommands {
	return &authService{users: u, apikeys: k, secret: secret, sso: sso}
}

// NewAuthQueriesService returns the queries-side of the same auth service.
// sso may be nil if SSO is not configured.
func NewAuthQueriesService(u users.UserRepository, k apikeys.APIKeyRepository, secret []byte, sso *SSOService) service.AuthQueries {
	return &authService{users: u, apikeys: k, secret: secret, sso: sso}
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

func (s *authService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error) {
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
	refreshToken, err = s.issueToken(user, tokenTypeRefresh, refreshTokenTTL)
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

func (s *authService) CreateAPIKey(ctx context.Context, actor domain.Actor, name string, scopes []string, expiresAt *time.Time) (domain.APIKey, string, error) {
	for _, scope := range scopes {
		if !allowedScopes[scope] {
			return domain.APIKey{}, "", &domain.Error{
				Code:    "INVALID_SCOPE",
				Message: fmt.Sprintf("unknown or invalid scope: %q", scope),
			}
		}
	}

	raw, err := generateRawKey()
	if err != nil {
		return domain.APIKey{}, "", fmt.Errorf("generate api key: %w", err)
	}

	keyHash := s.hashKey(raw)

	now := time.Now()
	key := domain.APIKey{
		ID:        domain.NewAPIKeyID(),
		UserID:    actor.UserID,
		Name:      name,
		KeyHash:   keyHash,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	if err := s.apikeys.Create(ctx, key); err != nil {
		return domain.APIKey{}, "", err
	}
	return key, raw, nil
}

func (s *authService) RevokeAPIKey(ctx context.Context, actor domain.Actor, keyID domain.APIKeyID) error {
	key, err := s.apikeys.FindByID(ctx, keyID)
	if err != nil {
		return err
	}
	if key.UserID != actor.UserID && !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	return s.apikeys.Revoke(ctx, keyID)
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

func (s *authService) ValidateAPIKey(ctx context.Context, rawKey string) (domain.Actor, error) {
	if len(rawKey) <= len(apiKeyPrefix) {
		return domain.Actor{}, domain.ErrAPIKeyInvalid
	}

	prefix := rawKey[:len(apiKeyPrefix)]
	if subtle.ConstantTimeCompare([]byte(prefix), []byte(apiKeyPrefix)) != 1 {
		return domain.Actor{}, domain.ErrAPIKeyInvalid
	}

	keyHash := s.hashKey(rawKey)

	key, err := s.apikeys.FindByHash(ctx, keyHash)
	if err != nil {
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			return domain.Actor{}, domain.ErrAPIKeyInvalid
		}
		return domain.Actor{}, err
	}

	if key.RevokedAt != nil {
		return domain.Actor{}, domain.ErrAPIKeyRevoked
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return domain.Actor{}, domain.ErrAPIKeyExpired
	}

	_ = s.apikeys.UpdateLastUsed(ctx, key.ID, time.Now())

	user, err := s.users.FindByID(ctx, key.UserID)
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

func (s *authService) ListAPIKeys(ctx context.Context, actor domain.Actor) ([]domain.APIKey, error) {
	return s.apikeys.ListByUser(ctx, actor.UserID)
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

func generateRawKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return apiKeyPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *authService) hashKey(raw string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}
