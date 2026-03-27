package security_test

// Security tests for the identity initialisation logic.
//
// Each vulnerability has a test that documents the correct, hardened behaviour
// expected from the production code.

import (
	"context"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/identity/app"
	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users/userstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// SEC-ID-04  JWT secret validation returns an error on a short value
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC_ID_04_JWTSecretValidationReturnsErrorOnShortValue asserts
// that app.NewAuthService returns an INSECURE_JWT_SECRET domain error from
// Login() when the JWT secret is shorter than 32 bytes, rather than panicking.
func TestSecurity_SEC_ID_04_JWTSecretValidationReturnsErrorOnShortValue(t *testing.T) {
	// FindByEmail must not be reached; if it is, the test should fail loudly.
	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, _ string) (identitydomain.User, error) {
			t.Fatal("FindByEmail must not be called when the JWT secret is too short")
			return identitydomain.User{}, nil
		},
	}

	shortSecret := []byte("tooshort") // < 32 bytes
	svc := app.NewAuthService(mockUsers, shortSecret, nil)

	_, _, err := svc.Login(context.Background(), "user@example.com", "password", false)

	require.Error(t, err, "SEC-ID-04: Login must return an error when the JWT secret is too short")

	var domainErr *identitydomain.Error
	require.ErrorAs(t, err, &domainErr,
		"SEC-ID-04: the error must be a domain error")
	assert.Equal(t, "INSECURE_JWT_SECRET", domainErr.Code,
		"SEC-ID-04: error code must be INSECURE_JWT_SECRET")
}

// TestSecurity_SEC_ID_04_JWTSecretValidationAcceptsLongEnoughSecret asserts
// that app.NewAuthService proceeds past the secret-length guard and attempts
// the real login flow when the JWT secret is at least 32 bytes long.
func TestSecurity_SEC_ID_04_JWTSecretValidationAcceptsLongEnoughSecret(t *testing.T) {
	mockUsers := &userstest.MockUserRepository{
		FindByEmailFunc: func(_ context.Context, _ string) (identitydomain.User, error) {
			// The secret length guard passed; the service now tries to look up
			// the user. Return ErrUserNotFound to keep the test self-contained.
			return identitydomain.User{}, identitydomain.ErrUserNotFound
		},
	}

	longSecret := []byte("this-is-exactly-32-characters!!!") // exactly 32 bytes
	require.Equal(t, 32, len(longSecret), "test precondition: secret must be exactly 32 bytes")

	svc := app.NewAuthService(mockUsers, longSecret, nil)

	_, _, err := svc.Login(context.Background(), "user@example.com", "password", false)

	require.Error(t, err, "Login must fail because the user does not exist")

	var domainErr *identitydomain.Error
	require.ErrorAs(t, err, &domainErr)
	assert.NotEqual(t, "INSECURE_JWT_SECRET", domainErr.Code,
		"SEC-ID-04: a 32-byte secret must not trigger the INSECURE_JWT_SECRET guard")
}
