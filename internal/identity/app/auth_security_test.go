package app_test

// Security tests for the identity/app authentication service.
//
// Each test is labelled RED or GREEN:
//   - RED  = demonstrates a vulnerability (the test catches a bad behavior;
//            it FAILS when the vulnerability is present and would PASS only after a fix)
//   - GREEN = shows the already-correct behavior that must be preserved
//
// Run with: go test -race -failfast ./internal/identity/app/...

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/app"
	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/apikeys/apikeystest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users/userstest"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-1: No email format validation in Register
// ─────────────────────────────────────────────────────────────────────────────

// RED: Register must reject obviously invalid email strings.
// Currently the service layer accepts any non-empty string as an email.
// This test FAILS until the production code validates email format.
func TestSecurity_Register_InvalidEmail_IsRejected_RED(t *testing.T) {
	ctx := context.Background()

	// We must not even reach the repository layer for an invalid email.
	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			// If this is called, it means the service did not pre-validate the email.
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error {
			// If this is called, the service stored a user with an invalid email.
			return nil
		},
	}
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	invalidEmails := []string{
		"notanemail",
		"@nodomain",
		"missing-at-sign",
		"double@@domain.com",
		"   ",
		"",
	}

	for _, email := range invalidEmails {
		email := email
		t.Run("email="+email, func(t *testing.T) {
			_, err := svc.Register(ctx, email, "validpassword123", "Test User")
			// RED: this assertion currently FAILS because the service does not validate emails.
			require.Error(t, err, "Register must reject invalid email %q", email)
		})
	}
}

// GREEN: Register accepts a well-formed email.
func TestSecurity_Register_ValidEmail_IsAccepted_GREEN(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	_, err := svc.Register(ctx, "valid@example.com", "validpassword123", "Test User")
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-2: No minimum secret length enforcement in NewAuthService
// A short/empty JWT secret allows trivially forgeable tokens.
// ─────────────────────────────────────────────────────────────────────────────

// RED: NewAuthService (or a downstream call) must refuse a dangerously short secret.
// An attacker can brute-force or forge JWTs signed with a 1-byte secret.
// This test FAILS until the constructor or first usage rejects short secrets.
func TestSecurity_NewAuthService_ShortSecret_IsRejected_RED(t *testing.T) {
	ctx := context.Background()

	shortSecret := []byte("x") // Only 1 byte — trivially brute-forceable

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, shortSecret, nil)

	// The vulnerability is detectable during Login (token issuance):
	registeredUser, err := svc.Register(ctx, "weak@example.com", "password123", "Weak")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, &apikeystest.MockAPIKeyRepository{}, shortSecret, nil)

	// A token issued with a 1-byte secret is insecure.
	// RED: the service should error when the secret is too short, but currently it does not.
	_, _, err = loginSvc.Login(ctx, "weak@example.com", "password123", false)
	assert.Error(t, err, "Login (token issuance) must fail when the JWT secret is dangerously short (<32 bytes)")
}

// GREEN: NewAuthService works correctly with a secret of at least 32 bytes.
func TestSecurity_NewAuthService_AdequateSecret_Works_GREEN(t *testing.T) {
	ctx := context.Background()

	adequateSecret := []byte("this-secret-is-exactly-32-bytes!") // 32 bytes
	assert.Equal(t, 32, len(adequateSecret))

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, adequateSecret, nil)

	user, err := svc.Register(ctx, "adequate@example.com", "password123", "User")
	require.NoError(t, err)

	loginUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return user, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(loginUsers, &apikeystest.MockAPIKeyRepository{}, adequateSecret, nil)

	access, refresh, err := loginSvc.Login(ctx, "adequate@example.com", "password123", false)
	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-3: JWT signed with "none" algorithm is rejected
// Classic JWT algorithm confusion — verifies the guard is in place.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN: A "none"-algorithm JWT must be rejected by ValidateJWT.
func TestSecurity_ValidateJWT_NoneAlgorithm_Rejected_GREEN(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	// Craft a JWT that uses alg=none (unsigned).
	// golang-jwt/jwt v5 rejects alg:none by default, but we verify the app-layer guard too.
	claims := jwt.MapClaims{
		"sub":        domain.NewUserID().String(),
		"email":      "attacker@evil.com",
		"role":       "admin",
		"token_type": "access",
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err, "crafting none-alg token should succeed")

	_, err = svc.ValidateJWT(ctx, tokenStr)
	assert.ErrorIs(t, err, domain.ErrUnauthorized, "none-algorithm JWT must be rejected")
}

// GREEN: A JWT signed with a different HMAC key is rejected.
func TestSecurity_ValidateJWT_WrongKey_Rejected_GREEN(t *testing.T) {
	ctx := context.Background()

	wrongSecret := []byte("wrong-secret-key-that-is-long-enough-32b")
	claims := jwt.MapClaims{
		"sub":        domain.NewUserID().String(),
		"email":      "attacker@evil.com",
		"role":       "admin",
		"token_type": "access",
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(wrongSecret)
	require.NoError(t, err)

	svc := app.NewAuthQueriesService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	_, err = svc.ValidateJWT(ctx, tokenStr)
	assert.ErrorIs(t, err, domain.ErrUnauthorized, "JWT signed with wrong key must be rejected")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-4: Logout is a no-op — tokens remain valid after logout
// ─────────────────────────────────────────────────────────────────────────────

// RED: After Logout, a previously-issued access token must be invalid.
// Currently Logout does nothing server-side, so the token remains valid.
// This test FAILS until token blacklisting / revocation is implemented.
func TestSecurity_Logout_TokenIsInvalidated_RED(t *testing.T) {
	ctx := context.Background()

	// Register and login to get a valid access token.
	mockUsersReg := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	regSvc := app.NewAuthService(mockUsersReg, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	registeredUser, err := regSvc.Register(ctx, "logout@example.com", "password123", "Logout User")
	require.NoError(t, err)

	mockUsersLogin := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(mockUsersLogin, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, "logout@example.com", "password123", false)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)

	// Logout.
	err = loginSvc.Logout(ctx, accessToken)
	require.NoError(t, err)

	// RED: The token should be invalid after logout, but it is not because Logout is a no-op.
	validateUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return registeredUser, nil
		},
	}
	validateSvc := app.NewAuthQueriesService(validateUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	_, err = validateSvc.ValidateJWT(ctx, accessToken)
	assert.Error(t, err, "Token must be invalidated after Logout — currently Logout is a no-op (RED)")
}

// GREEN: Registering with a brand-new email succeeds.
func TestSecurity_Register_NewEmail_Succeeds_GREEN(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	_, err := svc.Register(ctx, "brand-new@example.com", "password123", "New User")
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-6: Rate limiter IP spoofing via X-Forwarded-For
// (app-layer test: the IP extraction logic is in inbound/commands/auth.go but
// the rate limiter trust of unvalidated headers is a design vulnerability)
// We document it with a behavioral test here confirming it trusts untrusted headers.
// ─────────────────────────────────────────────────────────────────────────────

// No app-layer unit test possible here (logic lives in HTTP handler); documented above.

// ─────────────────────────────────────────────────────────────────────────────
// VULN-7: API key scopes are not validated against an allowlist
// ─────────────────────────────────────────────────────────────────────────────

// RED: CreateAPIKey must reject unrecognised scope strings.
// Currently any arbitrary string is stored as a scope, enabling privilege escalation
// via crafted scope values like "admin:*" or "billing:write".
// This test FAILS until scope validation is implemented.
func TestSecurity_CreateAPIKey_UnknownScope_IsRejected_RED(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}

	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		CreateFunc: func(_ context.Context, key domain.APIKey) error { return nil },
	}
	svc := app.NewAuthService(&userstest.MockUserRepository{}, mockAPIKeys, testJWTSecret, nil)

	invalidScopes := [][]string{
		{"admin:*"},          // wildcard admin escalation
		{"billing:write"},    // non-existent scope
		{""},                 // empty scope string
		{"kanban:read", ";"}, // injection attempt
	}

	for _, scopes := range invalidScopes {
		scopes := scopes
		_, _, err := svc.CreateAPIKey(ctx, actor, "Bad Key", scopes, nil)
		assert.Error(t, err,
			"CreateAPIKey must reject unknown/invalid scope values %v — currently it does not (RED)", scopes)
	}
}

// GREEN: CreateAPIKey accepts recognised, well-formed scopes.
func TestSecurity_CreateAPIKey_ValidScope_IsAccepted_GREEN(t *testing.T) {
	ctx := context.Background()

	actor := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}
	mockAPIKeys := &apikeystest.MockAPIKeyRepository{
		CreateFunc: func(_ context.Context, key domain.APIKey) error { return nil },
	}
	svc := app.NewAuthService(&userstest.MockUserRepository{}, mockAPIKeys, testJWTSecret, nil)

	validScopes := []string{"kanban:read", "kanban:write"}
	key, rawKey, err := svc.CreateAPIKey(ctx, actor, "Valid Key", validScopes, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, rawKey)
	assert.Equal(t, validScopes, key.Scopes)
}

// GREEN: Admin operations on teams correctly enforce admin authorization.
func TestSecurity_TeamService_MutatingOps_RequireAdmin_GREEN(t *testing.T) {
	ctx := context.Background()

	// All mutating operations must return ErrForbidden for a non-admin actor.
	svc := app.NewTeamService(&teamsTestMock{}, &userstest.MockUserRepository{})

	member := domain.Actor{UserID: domain.NewUserID(), Role: domain.RoleMember}

	_, err := svc.CreateTeam(ctx, member, "Team", "team", "")
	assert.ErrorIs(t, err, domain.ErrForbidden, "CreateTeam must forbid non-admin")

	err = svc.UpdateTeam(ctx, member, domain.Team{ID: domain.NewTeamID(), Name: "X", Slug: "x"})
	assert.ErrorIs(t, err, domain.ErrForbidden, "UpdateTeam must forbid non-admin")

	err = svc.DeleteTeam(ctx, member, domain.NewTeamID())
	assert.ErrorIs(t, err, domain.ErrForbidden, "DeleteTeam must forbid non-admin")

	err = svc.SetUserRole(ctx, member, domain.NewUserID(), domain.RoleAdmin)
	assert.ErrorIs(t, err, domain.ErrForbidden, "SetUserRole must forbid non-admin")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-9: UpdateTeam — mass assignment / no slug conflict check
// An admin can set a team's slug to an empty string or to a slug already in use.
// ─────────────────────────────────────────────────────────────────────────────

// RED: UpdateTeam must validate that the new slug is not empty.
// Currently it passes the struct directly to the repository without validation.
func TestSecurity_UpdateTeam_EmptySlug_IsRejected_RED(t *testing.T) {
	ctx := context.Background()

	mockTeams := &teamsTestMock{
		updateFn: func(_ context.Context, team domain.Team) error { return nil },
	}
	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	// An admin explicitly sets slug to empty.
	team := domain.Team{
		ID:   domain.NewTeamID(),
		Name: "Valid Name",
		Slug: "", // empty slug — should be rejected
	}
	err := svc.UpdateTeam(ctx, adminActor(), team)
	// RED: currently no validation, so this silently passes.
	assert.Error(t, err, "UpdateTeam must reject an empty slug (RED)")
}

// GREEN: UpdateTeam with a valid team struct succeeds for an admin.
func TestSecurity_UpdateTeam_ValidInput_Succeeds_GREEN(t *testing.T) {
	ctx := context.Background()

	mockTeams := &teamsTestMock{
		updateFn: func(_ context.Context, team domain.Team) error { return nil },
	}
	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	team := domain.Team{
		ID:   domain.NewTeamID(),
		Name: "Engineering",
		Slug: "engineering",
	}
	err := svc.UpdateTeam(ctx, adminActor(), team)
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-10: Password with leading/trailing whitespace is silently accepted
// ─────────────────────────────────────────────────────────────────────────────

// RED: Register must either trim whitespace from passwords or reject passwords
// that are purely whitespace, to prevent invisible-character credential confusion.
func TestSecurity_Register_WhitespaceOnlyPassword_IsRejected_RED(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	svc := app.NewAuthService(mockUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	// A password that is 8+ chars but is purely whitespace.
	whitespacePassword := strings.Repeat(" ", 10)

	_, err := svc.Register(ctx, "ws@example.com", whitespacePassword, "User")
	// RED: this currently succeeds because only len() is checked, not content.
	assert.Error(t, err, "Register must reject passwords composed entirely of whitespace (RED)")
}

// GREEN: Register rejects passwords that are too short.
func TestSecurity_Register_ShortPassword_IsRejected_GREEN(t *testing.T) {
	ctx := context.Background()

	svc := app.NewAuthService(&userstest.MockUserRepository{}, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	_, err := svc.Register(ctx, "short@example.com", "abc", "User")
	require.Error(t, err)
	assert.True(t, domain.IsDomainError(err), "short password error must be a domain error")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-11: Role in JWT claims can differ from the live DB role
// ValidateJWT fetches the user from DB (which is correct), but the actor's Role
// comes from the DB. This is actually implemented correctly — document the CORRECT behavior.
// ─────────────────────────────────────────────────────────────────────────────

// GREEN: ValidateJWT returns the role from the DB, not from the JWT claims.
// This ensures that a downgraded user's old token cannot be used to escalate privileges.
func TestSecurity_ValidateJWT_RoleTakenFromDB_NotFromToken_GREEN(t *testing.T) {
	ctx := context.Background()

	// Issue a token claiming admin role.
	adminUser := domain.User{
		ID:           domain.NewUserID(),
		Email:        "wasadmin@example.com",
		PasswordHash: "$2a$14$placeholder",
		Role:         domain.RoleAdmin, // was admin at issuance
	}

	mockUsersIssue := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return adminUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc := app.NewAuthService(mockUsersIssue, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	accessToken, _, err := loginSvc.Login(ctx, adminUser.Email, "irrelevant", false)
	// Login calls bcrypt.CompareHashAndPassword which will fail for placeholder hash,
	// so we need to create a real user first.
	// Use a real bcrypt hash to make this work.
	_ = accessToken
	_ = err

	// Use the real setup from existing tests to get a valid token.
	mockUsersReg := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
		CreateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	regSvc := app.NewAuthService(mockUsersReg, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	registeredAdminUser, err := regSvc.Register(ctx, "wasadmin2@example.com", "password123", "WasAdmin")
	require.NoError(t, err)

	// Manually set them as admin for issuance.
	registeredAdminUser.Role = domain.RoleAdmin

	mockUsersLogin := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, email string) (domain.User, error) {
			return registeredAdminUser, nil
		},
		UpdateFunc: func(_ context.Context, user domain.User) error { return nil },
	}
	loginSvc2 := app.NewAuthService(mockUsersLogin, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)
	adminToken, _, err := loginSvc2.Login(ctx, "wasadmin2@example.com", "password123", false)
	require.NoError(t, err)

	// Now the user has been demoted to member in the DB.
	demotedUser := registeredAdminUser
	demotedUser.Role = domain.RoleMember

	validateUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return demotedUser, nil // returns the demoted version
		},
	}
	validateSvc := app.NewAuthQueriesService(validateUsers, &apikeystest.MockAPIKeyRepository{}, testJWTSecret, nil)

	actor, err := validateSvc.ValidateJWT(ctx, adminToken)
	require.NoError(t, err)

	// GREEN: The role must come from the DB (member), not from the JWT claim (admin).
	assert.Equal(t, domain.RoleMember, actor.Role,
		"Actor role must reflect current DB state, not stale JWT claim")
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: minimal teamsTestMock to avoid import cycles / keep test self-contained
// ─────────────────────────────────────────────────────────────────────────────

// teamsTestMock implements teams.TeamRepository using function fields.
// Named differently from teamstest.MockTeamRepository to avoid confusion.
type teamsTestMock struct {
	createFn      func(ctx context.Context, team domain.Team) error
	findByIDFn    func(ctx context.Context, id domain.TeamID) (domain.Team, error)
	findBySlugFn  func(ctx context.Context, slug string) (domain.Team, error)
	listFn        func(ctx context.Context) ([]domain.Team, error)
	updateFn      func(ctx context.Context, team domain.Team) error
	deleteFn      func(ctx context.Context, id domain.TeamID) error
}

func (m *teamsTestMock) Create(ctx context.Context, team domain.Team) error {
	if m.createFn == nil {
		panic("called not defined createFn")
	}
	return m.createFn(ctx, team)
}

func (m *teamsTestMock) FindByID(ctx context.Context, id domain.TeamID) (domain.Team, error) {
	if m.findByIDFn == nil {
		panic("called not defined findByIDFn")
	}
	return m.findByIDFn(ctx, id)
}

func (m *teamsTestMock) FindBySlug(ctx context.Context, slug string) (domain.Team, error) {
	if m.findBySlugFn == nil {
		panic("called not defined findBySlugFn")
	}
	return m.findBySlugFn(ctx, slug)
}

func (m *teamsTestMock) List(ctx context.Context) ([]domain.Team, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx)
}

func (m *teamsTestMock) Update(ctx context.Context, team domain.Team) error {
	if m.updateFn == nil {
		panic("called not defined updateFn")
	}
	return m.updateFn(ctx, team)
}

func (m *teamsTestMock) Delete(ctx context.Context, id domain.TeamID) error {
	if m.deleteFn == nil {
		panic("called not defined deleteFn")
	}
	return m.deleteFn(ctx, id)
}
