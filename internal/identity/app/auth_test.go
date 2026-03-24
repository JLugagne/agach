package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/app"
	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodes/nodestest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users/userstest"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var testJWTSecret = []byte("test-secret-key-that-is-long-enough-32bytes")

func newTestAuthCommands(mockUsers *userstest.MockUserRepository) interface {
	Register(ctx context.Context, email, password, displayName string) (domain.User, error)
	Login(ctx context.Context, email, password string, rememberMe bool) (accessToken, refreshToken string, err error)
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error)
	Logout(ctx context.Context, token string) error
} {
	return app.NewAuthService(mockUsers, testJWTSecret, nil)
}

func newTestAuthQueries(mockUsers *userstest.MockUserRepository) interface {
	ValidateJWT(ctx context.Context, token string) (domain.Actor, error)
	GetCurrentUser(ctx context.Context, actor domain.Actor) (domain.User, error)
} {
	return app.NewAuthQueriesService(mockUsers, testJWTSecret, nil)
}

// ─────────────────────────────────────────────────────────────────────────────
// Register
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_Register_Success(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error {
			return nil
		},
	}

	svc := newTestAuthCommands(mockUsers)

	user, err := svc.Register(ctx, "test@example.com", "password123", "Test User")

	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.DisplayName)
	assert.Equal(t, domain.RoleMember, user.Role)
	assert.NotEmpty(t, user.PasswordHash)
}

func TestAuthService_Register_PasswordTooShort_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{}

	svc := newTestAuthCommands(mockUsers)

	_, err := svc.Register(ctx, "test@example.com", "short", "Test User")

	require.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
}

func TestAuthService_Register_EmailAlreadyExists_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{ID: domain.NewUserID(), Email: email}, nil
		},
	}

	svc := newTestAuthCommands(mockUsers)

	_, err := svc.Register(ctx, "existing@example.com", "password123", "User")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
}

// ─────────────────────────────────────────────────────────────────────────────
// Login
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_Login_Success(t *testing.T) {
	ctx := context.Background()

	// First register a user to get a valid bcrypt hash.
	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := newTestAuthCommands(mockUsers)
	registeredUser, err := svc.Register(ctx, "login@example.com", "password123", "Login User")
	require.NoError(t, err)

	// Now test login with a fresh service that returns the registered user.
	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := newTestAuthCommands(loginUsers)

	accessToken, refreshToken, err := loginSvc.Login(ctx, "login@example.com", "password123", false)

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
}

func TestAuthService_Login_UserNotFound_ReturnsInvalidCredentials(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	svc := newTestAuthCommands(mockUsers)

	_, _, err := svc.Login(ctx, "unknown@example.com", "password123", false)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_WrongPassword_ReturnsInvalidCredentials(t *testing.T) {
	ctx := context.Background()

	// Register first to get a valid hash.
	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := newTestAuthCommands(mockUsers)
	registeredUser, err := svc.Register(ctx, "wrongpw@example.com", "correctpassword", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
	}
	loginSvc := newTestAuthCommands(loginUsers)

	_, _, err = loginSvc.Login(ctx, "wrongpw@example.com", "wrongpassword", false)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_SSOUserNoPassword_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{
				ID:           domain.NewUserID(),
				Email:        email,
				PasswordHash: "", // SSO user has no password
				SSOProvider:  "google",
			}, nil
		},
	}
	svc := newTestAuthCommands(mockUsers)

	_, _, err := svc.Login(ctx, "sso@example.com", "any", false)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrSSOUserNoPassword)
}

// ─────────────────────────────────────────────────────────────────────────────
// Logout
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_Logout_AlwaysSucceeds(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthCommands(&userstest.MockUserRepository{})

	err := svc.Logout(ctx, "any-token")

	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// RefreshToken
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_RefreshToken_Success(t *testing.T) {
	ctx := context.Background()

	// Create a user and login to get tokens.
	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := newTestAuthCommands(mockUsers)
	registeredUser, err := svc.Register(ctx, "refresh@example.com", "password123", "Refresh User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := newTestAuthCommands(loginUsers)
	_, refreshToken, err := loginSvc.Login(ctx, "refresh@example.com", "password123", false)
	require.NoError(t, err)

	// Now refresh the token.
	refreshUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}

	refreshQueriesSvc := app.NewAuthService(refreshUsers, testJWTSecret, nil)
	newAccessToken, err := refreshQueriesSvc.RefreshToken(ctx, refreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
}

func TestAuthService_RefreshToken_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthService(&userstest.MockUserRepository{}, testJWTSecret, nil)

	_, err := svc.RefreshToken(ctx, "not-a-valid-token")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_RefreshToken_AccessTokenRejected(t *testing.T) {
	ctx := context.Background()

	// Register and login to get an access token.
	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := newTestAuthCommands(mockUsers)
	registeredUser, err := svc.Register(ctx, "access@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "access@example.com", "password123", false)
	require.NoError(t, err)

	// Attempt to use access token as refresh token.
	refreshUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	refreshSvc := app.NewAuthService(refreshUsers, testJWTSecret, nil)
	_, err = refreshSvc.RefreshToken(ctx, accessToken)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidateJWT
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_ValidateJWT_Success(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, testJWTSecret, nil)
	registeredUser, err := svc.Register(ctx, "validate@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "validate@example.com", "password123", false)
	require.NoError(t, err)

	validateUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	validateSvc := app.NewAuthQueriesService(validateUsers, testJWTSecret, nil)

	actor, err := validateSvc.ValidateJWT(ctx, accessToken)

	require.NoError(t, err)
	assert.Equal(t, registeredUser.ID, actor.UserID)
	assert.Equal(t, registeredUser.Email, actor.Email)
}

func TestAuthService_ValidateJWT_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, testJWTSecret, nil)

	_, err := svc.ValidateJWT(ctx, "invalid.token.here")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_ValidateJWT_RefreshTokenRejected(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, testJWTSecret, nil)
	registeredUser, err := svc.Register(ctx, "rejectrefresh@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, testJWTSecret, nil)
	_, refreshToken, err := loginSvc.Login(ctx, "rejectrefresh@example.com", "password123", false)
	require.NoError(t, err)

	// Attempt to use refresh token as access token.
	svc2 := app.NewAuthQueriesService(&userstest.MockUserRepository{}, testJWTSecret, nil)
	_, err = svc2.ValidateJWT(ctx, refreshToken)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_ValidateJWT_UserNotFound_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, testJWTSecret, nil)
	registeredUser, err := svc.Register(ctx, "deleted@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "deleted@example.com", "password123", false)
	require.NoError(t, err)

	// User was deleted after token was issued.
	validateUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	validateSvc := app.NewAuthQueriesService(validateUsers, testJWTSecret, nil)

	_, err = validateSvc.ValidateJWT(ctx, accessToken)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

// ─────────────────────────────────────────────────────────────────────────────
// GetCurrentUser
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_GetCurrentUser_Success(t *testing.T) {
	ctx := context.Background()

	userID := domain.NewUserID()
	actor := domain.Actor{UserID: userID}
	expectedUser := domain.User{
		ID:    userID,
		Email: "current@example.com",
	}

	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return expectedUser, nil
		},
	}

	svc := app.NewAuthQueriesService(mockUsers, testJWTSecret, nil)

	user, err := svc.GetCurrentUser(ctx, actor)

	require.NoError(t, err)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Email, user.Email)
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidateDaemonJWT
// ─────────────────────────────────────────────────────────────────────────────

func makeDaemonToken(t *testing.T, nodeID domain.NodeID, ownerID domain.UserID, mode domain.NodeMode, ttl time.Duration, secret []byte) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":        nodeID.String(),
		"owner_id":   ownerID.String(),
		"mode":       string(mode),
		"token_type": "daemon",
		"iat":        now.Unix(),
		"exp":        now.Add(ttl).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	require.NoError(t, err)
	return signed
}

func TestAuthService_ValidateDaemonJWT_Success(t *testing.T) {
	ctx := context.Background()

	nodeID := domain.NewNodeID()
	ownerID := domain.NewUserID()
	node := domain.Node{
		ID:          nodeID,
		OwnerUserID: ownerID,
		Mode:        domain.NodeModeDefault,
		Status:      domain.NodeStatusActive,
	}

	mockNodes := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, id domain.NodeID) (domain.Node, error) {
			return node, nil
		},
		UpdateLastSeenFunc: func(_ context.Context, id domain.NodeID) error {
			return nil
		},
	}

	svc := app.NewAuthQueriesServiceWithNodes(&userstest.MockUserRepository{}, mockNodes, testJWTSecret, nil)
	token := makeDaemonToken(t, nodeID, ownerID, domain.NodeModeDefault, 15*time.Minute, testJWTSecret)

	actor, err := svc.ValidateDaemonJWT(ctx, token)

	require.NoError(t, err)
	assert.Equal(t, nodeID, actor.NodeID)
	assert.Equal(t, ownerID, actor.OwnerUserID)
	assert.Equal(t, domain.NodeModeDefault, actor.Mode)
}

func TestAuthService_ValidateDaemonJWT_WrongTokenType(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	registerSvc := app.NewAuthService(mockUsers, testJWTSecret, nil)
	registeredUser, err := registerSvc.Register(ctx, "wrongtype@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "wrongtype@example.com", "password123", false)
	require.NoError(t, err)

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, testJWTSecret, nil)
	_, err = svc.ValidateDaemonJWT(ctx, accessToken)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_ValidateDaemonJWT_RevokedNode(t *testing.T) {
	ctx := context.Background()

	nodeID := domain.NewNodeID()
	ownerID := domain.NewUserID()
	node := domain.Node{
		ID:          nodeID,
		OwnerUserID: ownerID,
		Mode:        domain.NodeModeDefault,
		Status:      domain.NodeStatusRevoked,
	}

	mockNodes := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, id domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}

	svc := app.NewAuthQueriesServiceWithNodes(&userstest.MockUserRepository{}, mockNodes, testJWTSecret, nil)
	token := makeDaemonToken(t, nodeID, ownerID, domain.NodeModeDefault, 15*time.Minute, testJWTSecret)

	_, err := svc.ValidateDaemonJWT(ctx, token)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNodeRevoked)
}

// ─────────────────────────────────────────────────────────────────────────────
// RefreshDaemonToken
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_RefreshDaemonToken_Success(t *testing.T) {
	ctx := context.Background()

	nodeID := domain.NewNodeID()
	ownerID := domain.NewUserID()
	rawRefreshToken := "super-secret-refresh-token"
	hash, err := bcrypt.GenerateFromPassword([]byte(rawRefreshToken), 4)
	require.NoError(t, err)

	node := domain.Node{
		ID:               nodeID,
		OwnerUserID:      ownerID,
		Mode:             domain.NodeModeDefault,
		Status:           domain.NodeStatusActive,
		RefreshTokenHash: string(hash),
	}

	mockNodes := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, id domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}

	svc := app.NewAuthServiceWithNodes(&userstest.MockUserRepository{}, mockNodes, testJWTSecret, nil)

	newToken, err := svc.RefreshDaemonToken(ctx, nodeID, rawRefreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, newToken)
}

func TestAuthService_RefreshDaemonToken_RevokedNode(t *testing.T) {
	ctx := context.Background()

	nodeID := domain.NewNodeID()
	node := domain.Node{
		ID:     nodeID,
		Status: domain.NodeStatusRevoked,
	}

	mockNodes := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, id domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}

	svc := app.NewAuthServiceWithNodes(&userstest.MockUserRepository{}, mockNodes, testJWTSecret, nil)

	_, err := svc.RefreshDaemonToken(ctx, nodeID, "any-token")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNodeRevoked)
}
