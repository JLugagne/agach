package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/app"
	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/apikeys/apikeystest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users/userstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJWTSecret = []byte("test-secret-key-that-is-long-enough-32bytes")

func newTestAuthCommands(mockUsers *userstest.MockUserRepository, mockAPIKeys *apikeystest.MockAPIKeyRepository) interface {
	Register(ctx context.Context, email, password, displayName string) (domain.User, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error)
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error)
	Logout(ctx context.Context, token string) error
	CreateAPIKey(ctx context.Context, actor domain.Actor, name string, scopes []string, expiresAt *time.Time) (domain.APIKey, string, error)
	RevokeAPIKey(ctx context.Context, actor domain.Actor, keyID domain.APIKeyID) error
} {
	return app.NewAuthService(mockUsers, mockAPIKeys, testJWTSecret, nil)
}

func newTestAuthQueries(mockUsers *userstest.MockUserRepository, mockAPIKeys *apikeystest.MockAPIKeyRepository) interface {
	ValidateJWT(ctx context.Context, token string) (domain.Actor, error)
	ValidateAPIKey(ctx context.Context, rawKey string) (domain.Actor, error)
	ListAPIKeys(ctx context.Context, actor domain.Actor) ([]domain.APIKey, error)
	GetCurrentUser(ctx context.Context, actor domain.Actor) (domain.User, error)
} {
	return app.NewAuthQueriesService(mockUsers, mockAPIKeys, testJWTSecret, nil)
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
	mockAPIKeys := &apikeystest.MockAPIKeyRepository{}

	svc := newTestAuthCommands(mockUsers, mockAPIKeys)

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
	mockAPIKeys := &apikeystest.MockAPIKeyRepository{}

	svc := newTestAuthCommands(mockUsers, mockAPIKeys)

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
	mockAPIKeys := &apikeystest.MockAPIKeyRepository{}

	svc := newTestAuthCommands(mockUsers, mockAPIKeys)

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
	svc := newTestAuthCommands(mockUsers, &apikeystest.MockAPIKeyRepository{})
	registeredUser, err := svc.Register(ctx, "login@example.com", "password123", "Login User")
	require.NoError(t, err)

	// Now test login with a fresh service that returns the registered user.
	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := newTestAuthCommands(loginUsers, &apikeystest.MockAPIKeyRepository{})

	accessToken, refreshToken, err := loginSvc.Login(ctx, "login@example.com", "password123")

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
	svc := newTestAuthCommands(mockUsers, &apikeystest.MockAPIKeyRepository{})

	_, _, err := svc.Login(ctx, "unknown@example.com", "password123")

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
	svc := newTestAuthCommands(mockUsers, &apikeystest.MockAPIKeyRepository{})
	registeredUser, err := svc.Register(ctx, "wrongpw@example.com", "correctpassword", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
	}
	loginSvc := newTestAuthCommands(loginUsers, &apikeystest.MockAPIKeyRepository{})

	_, _, err = loginSvc.Login(ctx, "wrongpw@example.com", "wrongpassword")

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
	svc := newTestAuthCommands(mockUsers, &apikeystest.MockAPIKeyRepository{})

	_, _, err := svc.Login(ctx, "sso@example.com", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrSSOUserNoPassword)
}

// ─────────────────────────────────────────────────────────────────────────────
// Logout
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_Logout_AlwaysSucceeds(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthCommands(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{})

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
	svc := newTestAuthCommands(mockUsers, &apikeystest.MockAPIKeyRepository{})
	registeredUser, err := svc.Register(ctx, "refresh@example.com", "password123", "Refresh User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := newTestAuthCommands(loginUsers, &apikeystest.MockAPIKeyRepository{})
	_, refreshToken, err := loginSvc.Login(ctx, "refresh@example.com", "password123")
	require.NoError(t, err)

	// Now refresh the token.
	refreshUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	refreshSvc := newTestAuthCommands(refreshUsers, &apikeystest.MockAPIKeyRepository{})

	// RefreshToken is on the commands interface; use the same secret.
	refreshQueriesSvc := app.NewAuthService(refreshUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	newAccessToken, err := refreshQueriesSvc.RefreshToken(ctx, refreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
	_ = refreshSvc // suppress unused variable
}

func TestAuthService_RefreshToken_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

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
	svc := newTestAuthCommands(mockUsers, &apikeystest.MockAPIKeyRepository{})
	registeredUser, err := svc.Register(ctx, "access@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "access@example.com", "password123")
	require.NoError(t, err)

	// Attempt to use access token as refresh token.
	refreshUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	refreshSvc := app.NewAuthService(refreshUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
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
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	registeredUser, err := svc.Register(ctx, "validate@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "validate@example.com", "password123")
	require.NoError(t, err)

	validateUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	validateSvc := app.NewAuthQueriesService(validateUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	actor, err := validateSvc.ValidateJWT(ctx, accessToken)

	require.NoError(t, err)
	assert.Equal(t, registeredUser.ID, actor.UserID)
	assert.Equal(t, registeredUser.Email, actor.Email)
}

func TestAuthService_ValidateJWT_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

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
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	registeredUser, err := svc.Register(ctx, "rejectrefresh@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	_, refreshToken, err := loginSvc.Login(ctx, "rejectrefresh@example.com", "password123")
	require.NoError(t, err)

	// Attempt to use refresh token as access token.
	svc2 := app.NewAuthQueriesService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
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
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	registeredUser, err := svc.Register(ctx, "deleted@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "deleted@example.com", "password123")
	require.NoError(t, err)

	// User was deleted after token was issued.
	validateUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}
	validateSvc := app.NewAuthQueriesService(validateUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	_, err = validateSvc.ValidateJWT(ctx, accessToken)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

// ─────────────────────────────────────────────────────────────────────────────
// CreateAPIKey / RevokeAPIKey
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_CreateAPIKey_Success(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{
		UserID: domain.NewUserID(),
		Email:  "apikey@example.com",
		Role:   domain.RoleMember,
	}

	var storedKey domain.APIKey
	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		CreateFunc: func(_ context.Context, key domain.APIKey) error {
			storedKey = key
			return nil
		},
	}

	svc := newTestAuthCommands(&userstest.MockUserRepository{}, mockAPIKeys)

	key, rawKey, err := svc.CreateAPIKey(ctx, actor, "My Key", []string{"kanban:read"}, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, key.ID)
	assert.Equal(t, actor.UserID, key.UserID)
	assert.Equal(t, "My Key", key.Name)
	assert.Equal(t, []string{"kanban:read"}, key.Scopes)
	assert.NotEmpty(t, rawKey)
	assert.True(t, len(rawKey) > 6, "raw key should have prefix + content")
	assert.Equal(t, key.ID, storedKey.ID)
}

func TestAuthService_CreateAPIKey_WithExpiry_Success(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}
	expiry := time.Now().Add(24 * time.Hour)

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		CreateFunc: func(_ context.Context, key domain.APIKey) error { return nil },
	}

	svc := newTestAuthCommands(&userstest.MockUserRepository{}, mockAPIKeys)

	key, _, err := svc.CreateAPIKey(ctx, actor, "Expiring Key", nil, &expiry)

	require.NoError(t, err)
	require.NotNil(t, key.ExpiresAt)
}

func TestAuthService_RevokeAPIKey_Success(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}
	keyID := domain.NewAPIKeyID()

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		FindByIDFunc: func(_ context.Context, id domain.APIKeyID) (domain.APIKey, error) {
			return domain.APIKey{
				ID:     id,
				UserID: actor.UserID, // same user
			}, nil
		},
		RevokeFunc: func(_ context.Context, id domain.APIKeyID) error { return nil },
	}

	svc := newTestAuthCommands(&userstest.MockUserRepository{}, mockAPIKeys)

	err := svc.RevokeAPIKey(ctx, actor, keyID)

	require.NoError(t, err)
}

func TestAuthService_RevokeAPIKey_NotOwner_NotAdmin_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}
	keyID := domain.NewAPIKeyID()
	differentOwner := domain.NewUserID()

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		FindByIDFunc: func(_ context.Context, id domain.APIKeyID) (domain.APIKey, error) {
			return domain.APIKey{
				ID:     id,
				UserID: differentOwner, // different owner
			}, nil
		},
	}

	svc := newTestAuthCommands(&userstest.MockUserRepository{}, mockAPIKeys)

	err := svc.RevokeAPIKey(ctx, actor, keyID)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAuthService_RevokeAPIKey_AdminCanRevokeAny(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleAdmin}
	keyID := domain.NewAPIKeyID()
	differentOwner := domain.NewUserID()

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		FindByIDFunc: func(_ context.Context, id domain.APIKeyID) (domain.APIKey, error) {
			return domain.APIKey{
				ID:     id,
				UserID: differentOwner,
			}, nil
		},
		RevokeFunc: func(_ context.Context, id domain.APIKeyID) error { return nil },
	}

	svc := newTestAuthCommands(&userstest.MockUserRepository{}, mockAPIKeys)

	err := svc.RevokeAPIKey(ctx, actor, keyID)

	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// ValidateAPIKey
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_ValidateAPIKey_Success(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}
	// Create a real API key first.
	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		CreateFunc: func(_ context.Context, key domain.APIKey) error { return nil },
	}
	cmdSvc := newTestAuthCommands(&userstest.MockUserRepository{}, mockAPIKeys)
	key, rawKey, err := cmdSvc.CreateAPIKey(ctx, actor, "Test Key", []string{"kanban:read"}, nil)
	require.NoError(t, err)

	targetUser := domain.User{
		ID:    actor.UserID,
		Email: "apikey@example.com",
		Role:  domain.RoleMember,
	}

	mockAPIKeysQuery := &apikeystest.MockAPIKeyRepository{
		FindByHashFunc: func(_ context.Context, hash string) (domain.APIKey, error) {
			return key, nil
		},
		UpdateLastUsedFunc: func(_ context.Context, id domain.APIKeyID, at time.Time) error { return nil },
	}
	mockUsersQuery := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return targetUser, nil
		},
	}

	querySvc := app.NewAuthQueriesService(mockUsersQuery, mockAPIKeysQuery, testJWTSecret, nil)

	result, err := querySvc.ValidateAPIKey(ctx, rawKey)

	require.NoError(t, err)
	assert.Equal(t, targetUser.ID, result.UserID)
	assert.Equal(t, targetUser.Email, result.Email)
}

func TestAuthService_ValidateAPIKey_InvalidPrefix_ReturnsAPIKeyInvalid(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	_, err := svc.ValidateAPIKey(ctx, "wrong_prefix_1234")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAPIKeyInvalid)
}

func TestAuthService_ValidateAPIKey_TooShort_ReturnsAPIKeyInvalid(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	_, err := svc.ValidateAPIKey(ctx, "agach")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAPIKeyInvalid)
}

func TestAuthService_ValidateAPIKey_Revoked_ReturnsAPIKeyRevoked(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	revokedKey := domain.APIKey{
		ID:        domain.NewAPIKeyID(),
		UserID:    domain.NewUserID(),
		RevokedAt: &now,
	}

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		FindByHashFunc: func(_ context.Context, hash string) (domain.APIKey, error) {
			return revokedKey, nil
		},
	}

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, mockAPIKeys, testJWTSecret, nil)

	_, err := svc.ValidateAPIKey(ctx, "agach_validkeylongenough123456789012345")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAPIKeyRevoked)
}

func TestAuthService_ValidateAPIKey_Expired_ReturnsAPIKeyExpired(t *testing.T) {
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour)
	expiredKey := domain.APIKey{
		ID:        domain.NewAPIKeyID(),
		UserID:    domain.NewUserID(),
		ExpiresAt: &past,
	}

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		FindByHashFunc: func(_ context.Context, hash string) (domain.APIKey, error) {
			return expiredKey, nil
		},
	}

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, mockAPIKeys, testJWTSecret, nil)

	_, err := svc.ValidateAPIKey(ctx, "agach_validkeylongenough123456789012345")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAPIKeyExpired)
}

func TestAuthService_ValidateAPIKey_NotFound_ReturnsAPIKeyInvalid(t *testing.T) {
	ctx := context.Background()

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		FindByHashFunc: func(_ context.Context, hash string) (domain.APIKey, error) {
			return domain.APIKey{}, domain.ErrAPIKeyNotFound
		},
	}

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, mockAPIKeys, testJWTSecret, nil)

	_, err := svc.ValidateAPIKey(ctx, "agach_validkeylongenough123456789012345")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAPIKeyInvalid)
}

// ─────────────────────────────────────────────────────────────────────────────
// ListAPIKeys / GetCurrentUser
// ─────────────────────────────────────────────────────────────────────────────

func TestAuthService_ListAPIKeys_Success(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID()}
	expectedKeys := []domain.APIKey{
		{ID: domain.NewAPIKeyID(), UserID: actor.UserID, Name: "Key 1"},
		{ID: domain.NewAPIKeyID(), UserID: actor.UserID, Name: "Key 2"},
	}

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		ListByUserFunc: func(_ context.Context, userID domain.UserID) ([]domain.APIKey, error) {
			return expectedKeys, nil
		},
	}

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, mockAPIKeys, testJWTSecret, nil)

	keys, err := svc.ListAPIKeys(ctx, actor)

	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

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

	svc := app.NewAuthQueriesService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	user, err := svc.GetCurrentUser(ctx, actor)

	require.NoError(t, err)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Email, user.Email)
}
