package security_test

// NEW security tests for vulnerabilities not covered by existing tests.
//
// Each test is a RED test: it demonstrates an existing vulnerability by asserting
// the SECURE behaviour that is currently NOT enforced. The test FAILS today and
// should PASS after the vulnerability is fixed.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/app"
	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users/userstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-12: Blocked user can still log in
// File: internal/identity/app/auth.go:123-161
//
// Login never checks user.IsBlocked(). A user who has been blocked by an admin
// via BlockUser can still authenticate and receive fresh access/refresh tokens.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_BlockedUserCanLogin documents that a blocked user can still
// successfully call Login and receive tokens.
func TestSecurity_BlockedUserCanLogin(t *testing.T) {
	ctx := context.Background()

	// Register a user first to get a real bcrypt hash.
	mockUsersReg := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	regSvc := app.NewAuthService(mockUsersReg, testJWTSecret, nil)
	registeredUser, err := regSvc.Register(ctx, "blocked@example.com", "password123", "Blocked User")
	require.NoError(t, err)

	// Mark the user as blocked.
	blockedAt := time.Now()
	registeredUser.BlockedAt = &blockedAt

	mockUsersLogin := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(mockUsersLogin, testJWTSecret, nil)

	// RED: Login should fail for a blocked user, but currently it succeeds.
	_, _, err = loginSvc.Login(ctx, "blocked@example.com", "password123", false)
	assert.Error(t, err, "RED: Login must reject blocked users (auth.go:123-161)")
	t.Log("RED: blocked user can still log in — Login does not check user.IsBlocked()")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-13: Blocked user's existing JWT tokens remain valid
// File: internal/identity/app/auth.go:209-247
//
// ValidateJWT fetches the user from the DB and builds an Actor, but never
// checks user.IsBlocked(). A blocked user's previously-issued access tokens
// continue to work until they expire.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_BlockedUserTokenStillValid documents that a blocked user's
// access token is still accepted by ValidateJWT.
func TestSecurity_BlockedUserTokenStillValid(t *testing.T) {
	ctx := context.Background()

	// Create a user and get a valid token.
	mockUsersReg := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	regSvc := app.NewAuthService(mockUsersReg, testJWTSecret, nil)
	registeredUser, err := regSvc.Register(ctx, "blockedtoken@example.com", "password123", "Will Be Blocked")
	require.NoError(t, err)

	mockUsersLogin := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(mockUsersLogin, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "blockedtoken@example.com", "password123", false)
	require.NoError(t, err)

	// Now block the user in the DB.
	blockedAt := time.Now()
	blockedUser := registeredUser
	blockedUser.BlockedAt = &blockedAt

	validateUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return blockedUser, nil // returns the blocked version
		},
	}
	validateSvc := app.NewAuthQueriesService(validateUsers, testJWTSecret, nil)

	// RED: Token should be rejected for a blocked user, but currently it is accepted.
	_, err = validateSvc.ValidateJWT(ctx, accessToken)
	assert.Error(t, err, "RED: ValidateJWT must reject tokens for blocked users (auth.go:209-247)")
	t.Log("RED: blocked user's access token is still valid — ValidateJWT does not check IsBlocked()")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-14: ChangePassword does not invalidate existing sessions/tokens
// File: internal/identity/app/auth.go:272-305
//
// After a password change, all previously-issued access and refresh tokens
// remain valid. An attacker who has stolen a token can continue using it
// even after the legitimate user changes their password. The password change
// should invalidate all existing tokens for that user.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_ChangePasswordDoesNotInvalidateTokens documents that
// tokens issued before a password change remain valid afterward.
func TestSecurity_ChangePasswordDoesNotInvalidateTokens(t *testing.T) {
	ctx := context.Background()

	// Register and login to get a valid access token.
	mockUsersReg := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	regSvc := app.NewAuthService(mockUsersReg, testJWTSecret, nil)
	registeredUser, err := regSvc.Register(ctx, "changepw@example.com", "oldpassword1", "PW User")
	require.NoError(t, err)

	mockUsersLogin := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	svc := app.NewAuthService(mockUsersLogin, testJWTSecret, nil)
	accessToken, _, err := svc.Login(ctx, "changepw@example.com", "oldpassword1", false)
	require.NoError(t, err)

	// Change password.
	actor := domain.Actor{UserID: registeredUser.ID, Email: registeredUser.Email, Role: registeredUser.Role}
	err = svc.ChangePassword(ctx, actor, "oldpassword1", "newpassword1", accessToken)
	require.NoError(t, err)

	// The old access token should now be invalid, but it is not.
	validateSvc := app.NewAuthQueriesService(mockUsersLogin, testJWTSecret, nil)
	_, err = validateSvc.ValidateJWT(ctx, accessToken)
	// RED: The old token still works after password change.
	assert.Error(t, err,
		"RED: old access token remains valid after password change (auth.go:272-305)")
	t.Log("RED: ChangePassword does not invalidate existing tokens — stolen tokens survive password changes")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-15: Refresh token is not rotated on use
// File: internal/identity/app/auth.go:170-200
//
// RefreshToken issues a new access token but does NOT rotate or invalidate
// the refresh token itself. A stolen refresh token can be used repeatedly
// until it expires (7-30 days), giving an attacker persistent access.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RefreshTokenNotRotated documents that a refresh token
// can be reused multiple times to get new access tokens.
func TestSecurity_RefreshTokenNotRotated(t *testing.T) {
	ctx := context.Background()

	mockUsersReg := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	regSvc := app.NewAuthService(mockUsersReg, testJWTSecret, nil)
	registeredUser, err := regSvc.Register(ctx, "refresh@example.com", "password123", "Refresh User")
	require.NoError(t, err)

	mockUsersLogin := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(mockUsersLogin, testJWTSecret, nil)
	_, refreshToken, err := loginSvc.Login(ctx, "refresh@example.com", "password123", false)
	require.NoError(t, err)

	// Use the refresh token to get a new access token.
	mockUsersFind := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	refreshSvc := app.NewAuthService(mockUsersFind, testJWTSecret, nil)

	newAccess1, err := refreshSvc.RefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, newAccess1)

	// RED: Use the SAME refresh token again — this should fail after rotation is
	// implemented, but currently succeeds because the token is never invalidated.
	newAccess2, err := refreshSvc.RefreshToken(ctx, refreshToken)
	assert.Error(t, err,
		"RED: same refresh token can be reused — no rotation implemented (auth.go:170-200)")
	if err == nil {
		assert.NotEmpty(t, newAccess2)
		t.Log("RED: refresh token reuse succeeds — stolen tokens provide persistent access for 7-30 days")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-16: Admin can self-demote, potentially locking out all admins
// File: internal/identity/app/teams.go:97-108
//
// SetUserRole allows an admin to set any user's role, including their own.
// If the last admin demotes themselves, no one has admin access anymore.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_AdminCanSelfDemote documents that an admin can demote
// themselves to member role.
func TestSecurity_AdminCanSelfDemote(t *testing.T) {
	ctx := context.Background()

	adminID := domain.NewUserID()
	admin := domain.Actor{UserID: adminID, Email: "admin@example.com", Role: domain.RoleAdmin}

	adminUser := domain.User{
		ID:    adminID,
		Email: "admin@example.com",
		Role:  domain.RoleAdmin,
	}

	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return adminUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	mockTeams := &teamsTestMock{} // not used for this operation

	svc := app.NewTeamService(mockTeams, mockUsers)

	// RED: Admin demotes themselves — should be rejected but currently succeeds.
	err := svc.SetUserRole(ctx, admin, adminID, domain.RoleMember)
	assert.Error(t, err,
		"RED: admin can self-demote to member, risking total admin lockout (teams.go:97-108)")
	t.Log("RED: SetUserRole allows admin to demote themselves — last-admin lockout is possible")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-17: DisplayName has no length limit
// File: internal/identity/app/auth.go:107 (Register), auth.go:263 (UpdateProfile)
//
// Register and UpdateProfile accept arbitrarily long display names with no
// validation. An attacker can store megabytes of data in the display_name field,
// causing storage bloat and potential UI rendering issues (DoS via rendering).
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_DisplayNameNoLengthLimit documents that Register accepts
// an excessively long display name without validation.
func TestSecurity_DisplayNameNoLengthLimit(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, testJWTSecret, nil)

	// 10,000 character display name — far beyond any reasonable limit.
	longName := strings.Repeat("A", 10000)

	_, err := svc.Register(ctx, "longname@example.com", "password123", longName)
	assert.Error(t, err,
		"RED: Register accepts a 10,000-char display name without validation (auth.go:107)")
	t.Log("RED: no length limit on display_name — storage bloat and UI DoS possible")
}
