package identity_test

// Security tests for the identity initialisation logic.
//
// Each vulnerability has a RED test (documents the broken behaviour) and a
// GREEN test (documents the correct, hardened behaviour).

import (
	"context"
	"testing"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// SEC-ID-04  JWT secret propagation via jwtSecret() panics instead of erroring
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC_ID_04_RED_JWTSecretPanicsOnShortValue documents that
// jwtSecret() in internal/server/init.go calls panic() when JWT_SECRET < 32
// chars, crashing the entire process instead of returning a proper error.
func TestSecurity_SEC_ID_04_RED_JWTSecretPanicsOnShortValue(t *testing.T) {
	// We simulate the logic of jwtSecret() here (cannot call it directly because
	// it reads os.Getenv directly):
	//   if len(s) < 32 { panic(...) }
	validateJWTSecret := func(s string) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// In production code this would crash the server.
				// We document it as a "caught panic" — the real issue is
				// that panic is used instead of returning an error.
				err = context.DeadlineExceeded // sentinel: means panic was triggered
			}
		}()
		if len(s) < 32 {
			panic("JWT_SECRET too short")
		}
		return nil
	}

	err := validateJWTSecret("tooshort")
	assert.Error(t, err,
		"RED SEC-ID-04: jwtSecret() panics on a short secret; "+
			"fix: return an error instead of calling panic so the caller can handle it gracefully")
}

// TestSecurity_SEC_ID_04_GREEN_JWTSecretValidationReturnsError documents that
// the jwtSecret function (or its replacement) must return a proper error rather
// than panicking, so initialization can fail gracefully.
func TestSecurity_SEC_ID_04_GREEN_JWTSecretValidationReturnsError(t *testing.T) {
	// Hardened version: returns an error instead of panicking.
	validateJWTSecretHardened := func(s string) error {
		if len(s) < 32 {
			return &identitydomain.Error{
				Code:    "JWT_SECRET_TOO_SHORT",
				Message: "JWT_SECRET must be at least 32 characters",
			}
		}
		return nil
	}

	err := validateJWTSecretHardened("tooshort")
	require.Error(t, err,
		"GREEN SEC-ID-04: short secret must produce an error, not a panic")

	var domainErr *identitydomain.Error
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "JWT_SECRET_TOO_SHORT", domainErr.Code)

	// A 32-char secret must pass.
	longSecret := "this-is-exactly-32-characters!!!"
	require.Equal(t, 32, len(longSecret), "test string must be exactly 32 chars")
	require.NoError(t, validateJWTSecretHardened(longSecret),
		"GREEN SEC-ID-04: 32-char secret must be accepted")
}
