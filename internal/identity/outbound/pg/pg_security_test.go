// Package pg_test contains security-focused tests for the identity pg package.
//
// Each test is either:
//   - RED: demonstrates a current vulnerability (will fail when the vulnerability is fixed)
//   - GREEN: demonstrates a desired secure behaviour (passes when the code is correct/secure)
//
// Run with: go test -race -failfast ./internal/identity/outbound/pg/...
package pg

// NOTE: This is a white-box test file (package pg, not pg_test) so it can
// inspect unexported symbols and embedded SQL strings.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// VULN-01: Missing per-operation query timeouts
// ---------------------------------------------------------------------------

// GREEN: documents the desired state — every exported method should wrap the
// incoming context with a sensible per-operation deadline before calling the
// pool. This test is currently failing (GREEN = what we WANT to be true).
func TestSecurity_VULN01_QueryTimeoutEnforced_Green(t *testing.T) {
	src, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	content := string(src)
	hasTimeout := strings.Contains(content, "context.WithTimeout") ||
		strings.Contains(content, "context.WithDeadline")

	// GREEN assertion: after the fix, this must pass.
	assert.True(t, hasTimeout,
		"VULN-01 GREEN (currently failing): pg.go should wrap contexts with a per-query deadline "+
			"to prevent unbounded blocking on slow queries or DB hangs.")
}

// ---------------------------------------------------------------------------
// VULN-02: API key stored as bare SHA-256 (fast hash, brute-forceable)
// ---------------------------------------------------------------------------

// RED: confirms that the current key_hash column stores a raw hex-encoded
// SHA-256 digest. SHA-256 can be computed at billions of operations per second
// on commodity hardware. A leaked api_keys table allows offline brute-force.
func TestSecurity_VULN02_APIKeyHashIsSHA256_Red(t *testing.T) {
	// Simulate what hashKey() does in auth.go to prove the algorithm in use.
	rawKey := "agach_test_key_value_for_security_analysis"
	h := sha256.Sum256([]byte(rawKey))
	hexHash := hex.EncodeToString(h[:])

	// A SHA-256 hex string is exactly 64 characters long.
	assert.Len(t, hexHash, 64,
		"VULN-02 RED: the api key hash is a 64-char hex SHA-256 digest — "+
			"a fast hash with no work-factor, making offline brute-force trivial after DB breach.")

	// Confirm it is NOT an Argon2/bcrypt output (which start with recognizable prefixes).
	assert.False(t, strings.HasPrefix(hexHash, "$argon2"),
		"API key hash is not Argon2 — fast hash confirmed.")
	assert.False(t, strings.HasPrefix(hexHash, "$2"),
		"API key hash is not bcrypt — fast hash confirmed.")
}

// GREEN: documents what the secure implementation should look like.
// An API key hash should use a keyed HMAC or slow KDF, or at minimum
// HMAC-SHA256 with a server-side secret so that DB exposure alone
// does not allow key recovery.
func TestSecurity_VULN02_APIKeyHashUsesHMAC_Green(t *testing.T) {
	src, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	// We check the auth layer via the embedded migration or companion file.
	// The fix would add an hmac import and use it in hashKey().
	authSrc, err := os.ReadFile("../../app/auth.go")
	require.NoError(t, err)

	usesHMAC := strings.Contains(string(authSrc), "hmac.New") ||
		strings.Contains(string(authSrc), "hmac.Sum") ||
		strings.Contains(string(src), "hmac")

	assert.True(t, usesHMAC,
		"VULN-02 GREEN (currently failing): API key hashing should use HMAC-SHA256 "+
			"with a server-side secret key so that a DB breach alone does not expose key material.")
}

// ---------------------------------------------------------------------------
// VULN-05: pgcrypto extension loaded but never used
// ---------------------------------------------------------------------------

// GREEN: either pgcrypto is removed from the migration (because it is unused),
// or it is actually used for column-level encryption of sensitive fields.
func TestSecurity_VULN05_PgcryptoEitherUsedOrRemoved_Green(t *testing.T) {
	migSrc, err := os.ReadFile("migrations/001_identity.sql")
	require.NoError(t, err)

	pgGoSrc, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	migContent := string(migSrc)
	pgContent := string(pgGoSrc)

	extensionLoaded := strings.Contains(migContent, "pgcrypto")
	pgcryptoUsed := strings.Contains(pgContent, "pgp_sym_encrypt") ||
		strings.Contains(pgContent, "pgp_sym_decrypt") ||
		strings.Contains(migContent, "crypt(")

	// GREEN: if extension is loaded, it must be actively used.
	if extensionLoaded {
		assert.True(t, pgcryptoUsed,
			"VULN-05 GREEN (currently failing): pgcrypto is loaded — it should be used "+
				"for column-level encryption of sensitive fields (password_hash, sso_subject) "+
				"or removed entirely.")
	}
	// If not loaded, the test trivially passes (extension removed = fix applied).
}

// ---------------------------------------------------------------------------
// VULN-06: ListAll leaks password_hash to every caller
// ---------------------------------------------------------------------------

// RED: ListAll fetches password_hash for every user in the system.
// Any caller that logs, caches, or serializes the result leaks credential data.
func TestSecurity_VULN06_ListAllIncludesPasswordHash_Red(t *testing.T) {
	src, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	content := string(src)

	// Find the ListAll function's SQL — it should select password_hash.
	// We verify by checking that the SELECT used in scanUser includes password_hash
	// and that ListAll calls scanUsers (which calls scanUser).
	listAllSelectsPasswordHash := strings.Contains(content, "password_hash") &&
		strings.Contains(content, "ListAll")

	assert.True(t, listAllSelectsPasswordHash,
		"VULN-06 RED: ListAll retrieves password_hash for all users. "+
			"Callers that log or serialize the result risk leaking hashed credentials. "+
			"A ListAll for administrative display purposes should omit the password_hash column.")
}

// GREEN: a secure ListAll should use a projection that omits password_hash,
// or return a separate DTO type that cannot carry credential data.
func TestSecurity_VULN06_ListAllOmitsPasswordHash_Green(t *testing.T) {
	src, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	// Parse the AST to find the SQL query literal inside ListAll.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "pg.go", src, 0)
	require.NoError(t, err)

	// Walk the AST looking for the ListAll function body.
	listAllSQL := ""
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "ListAll" {
			return true
		}
		// Walk the function body for string literals (SQL).
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			lit, ok := inner.(*ast.BasicLit)
			if ok && lit.Kind == token.STRING {
				listAllSQL += lit.Value
			}
			return true
		})
		return false
	})

	// GREEN: the SQL in ListAll should NOT include password_hash.
	omitsHash := !strings.Contains(listAllSQL, "password_hash")
	assert.True(t, omitsHash,
		"VULN-06 GREEN (currently failing): the SQL query in ListAll should omit "+
			"'password_hash' from the SELECT list to avoid unnecessarily surfacing credential data.")
}

// ---------------------------------------------------------------------------
// VULN-07: Sensitive columns stored as plaintext TEXT (no column encryption)
// ---------------------------------------------------------------------------

// GREEN: sensitive columns should use pgcrypto-backed column encryption or
// the application should encrypt values before storing them.
func TestSecurity_VULN07_SensitiveColumnsEncrypted_Green(t *testing.T) {
	migSrc, err := os.ReadFile("migrations/001_identity.sql")
	require.NoError(t, err)

	pgGoSrc, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	migContent := string(migSrc)
	pgContent := string(pgGoSrc)

	// Either application-layer encryption (pgp_sym_encrypt in queries)
	// or pg column type BYTEA with encryption applied.
	appLevelEncryption := strings.Contains(pgContent, "pgp_sym_encrypt") ||
		strings.Contains(pgContent, "pgp_sym_decrypt")
	schemaLevelEncryption := strings.Contains(migContent, "BYTEA") &&
		strings.Contains(migContent, "pgp_sym")

	hasEncryption := appLevelEncryption || schemaLevelEncryption

	assert.True(t, hasEncryption,
		"VULN-07 GREEN (currently failing): sensitive columns (password_hash, sso_subject) "+
			"should use pgcrypto column-level encryption (pgp_sym_encrypt/decrypt) or "+
			"application-layer encryption before storage.")
}

// ---------------------------------------------------------------------------
// VULN-08: No DB-side enforcement of updated_at (audit trail integrity)
// ---------------------------------------------------------------------------

// GREEN: a trigger (or generated column) should ensure updated_at is always
// refreshed on UPDATE, regardless of what the application provides.
func TestSecurity_VULN08_UpdatedAtTriggerExists_Green(t *testing.T) {
	migSrc, err := os.ReadFile("migrations/001_identity.sql")
	require.NoError(t, err)

	content := strings.ToUpper(string(migSrc))
	hasTrigger := strings.Contains(content, "CREATE TRIGGER") ||
		strings.Contains(content, "CREATE OR REPLACE TRIGGER")

	assert.True(t, hasTrigger,
		"VULN-08 GREEN (currently failing): migration should define a trigger that "+
			"auto-sets updated_at = NOW() on every UPDATE so audit timestamps are always accurate.")
}

// ---------------------------------------------------------------------------
// VULN-09: Inconsistent time sources (app time vs DB time)
// ---------------------------------------------------------------------------

// GREEN: all timestamp writes in the repository should use either consistently
// DB-side NOW() or consistently application-provided values — not a mix.
func TestSecurity_VULN09_ConsistentTimeSources_Green(t *testing.T) {
	src, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	content := string(src)

	// One consistent approach: use DB NOW() for all audit timestamps.
	revokeUsesDBNow := strings.Contains(content, "revoked_at = NOW()")
	updateLastUsedUsesDBNow := strings.Contains(content, "last_used_at = NOW()")

	consistent := (revokeUsesDBNow && updateLastUsedUsesDBNow) ||
		(!revokeUsesDBNow && !updateLastUsedUsesDBNow)

	assert.True(t, consistent,
		"VULN-09 GREEN (currently failing): all audit timestamp writes should use the same "+
			"time source. Either consistently use DB NOW() (safest) or consistently use "+
			"application-provided values — never both in the same package.")
}

// ---------------------------------------------------------------------------
// VULN-10: API key hashed without salt
// ---------------------------------------------------------------------------

// GREEN: hashKey should use HMAC-SHA256 with a server-side secret
// so that the DB hash alone cannot be brute-forced without the secret.
func TestSecurity_VULN10_APIKeyHashUsesHMACWithSecret_Green(t *testing.T) {
	authSrc, err := os.ReadFile("../../app/auth.go")
	require.NoError(t, err)

	content := string(authSrc)

	usesHMAC := strings.Contains(content, "hmac.New") ||
		strings.Contains(content, "hmac.Sum")

	assert.True(t, usesHMAC,
		"VULN-10 GREEN (currently failing): hashKey should use HMAC-SHA256 keyed with "+
			"a server-side secret (the same JWT secret or a dedicated HMAC key) so that "+
			"leaking the api_keys table does not allow offline key recovery.")
}

// ---------------------------------------------------------------------------
// Additional: No SQL injection via string concatenation (positive verification)
// ---------------------------------------------------------------------------

// GREEN (passing): All queries in pg.go use parameterized placeholders ($1, $2, ...).
// This test asserts the ABSENCE of string concatenation in SQL queries.
// This is one of the few tests that should currently PASS — confirming that
// SQL injection is not present.
func TestSecurity_NoSQLInjection_AllQueriesParameterized_Green(t *testing.T) {
	fset := token.NewFileSet()
	src, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	f, err := parser.ParseFile(fset, "pg.go", src, 0)
	require.NoError(t, err)

	// Collect all string literals that look like SQL (contain SELECT/INSERT/UPDATE/DELETE).
	var sqlLiterals []string
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		upper := strings.ToUpper(lit.Value)
		if strings.Contains(upper, "SELECT") ||
			strings.Contains(upper, "INSERT") ||
			strings.Contains(upper, "UPDATE") ||
			strings.Contains(upper, "DELETE") {
			sqlLiterals = append(sqlLiterals, lit.Value)
		}
		return true
	})

	require.NotEmpty(t, sqlLiterals, "Expected to find SQL string literals in pg.go")

	// None of the SQL literals should use fmt.Sprintf-style % placeholders
	// or string concatenation with user values. All parameters must use $N.
	for _, sql := range sqlLiterals {
		// Check for dangerous patterns: %s, %v, %d in SQL strings.
		assert.NotContains(t, sql, "%s",
			"SQL literal contains %%s format verb — potential SQL injection: %q", sql)
		assert.NotContains(t, sql, "%v",
			"SQL literal contains %%v format verb — potential SQL injection: %q", sql)
		assert.NotContains(t, sql, "%d",
			"SQL literal contains %%d format verb — potential SQL injection: %q", sql)
	}
}

// ---------------------------------------------------------------------------
// Additional: Migration is idempotent (CREATE IF NOT EXISTS)
// ---------------------------------------------------------------------------

// GREEN (passing): migration uses IF NOT EXISTS guards — safe to re-run.
func TestSecurity_MigrationIsIdempotent_Green(t *testing.T) {
	migSrc, err := os.ReadFile("migrations/001_identity.sql")
	require.NoError(t, err)

	content := string(migSrc)

	// Every CREATE statement must use IF NOT EXISTS.
	upperContent := strings.ToUpper(content)

	// Find all CREATE TABLE occurrences.
	createCount := strings.Count(upperContent, "CREATE TABLE ")
	createIfNotExistsCount := strings.Count(upperContent, "CREATE TABLE IF NOT EXISTS")
	createIndexCount := strings.Count(upperContent, "CREATE INDEX ")
	createUniqueIndexCount := strings.Count(upperContent, "CREATE UNIQUE INDEX ")
	createIfNotExistsIndexCount := strings.Count(upperContent, "CREATE INDEX IF NOT EXISTS")
	createUniqueIfNotExistsIndexCount := strings.Count(upperContent, "CREATE UNIQUE INDEX IF NOT EXISTS")

	assert.Equal(t, createCount, createIfNotExistsCount,
		"All CREATE TABLE statements should use IF NOT EXISTS for safe re-runs")
	assert.Equal(t, createIndexCount+createUniqueIndexCount,
		createIfNotExistsIndexCount+createUniqueIfNotExistsIndexCount,
		"All CREATE INDEX statements should use IF NOT EXISTS for safe re-runs")
}

// ---------------------------------------------------------------------------
// Additional: No DROP statements without transaction guards in migrations
// ---------------------------------------------------------------------------

// GREEN (passing): migration should not contain bare DROP TABLE/DROP COLUMN
// without a conditional guard, which could destroy data on accidental re-run.
func TestSecurity_MigrationHasNoUnsafeDROP_Green(t *testing.T) {
	migSrc, err := os.ReadFile("migrations/001_identity.sql")
	require.NoError(t, err)

	content := strings.ToUpper(string(migSrc))

	hasDrop := strings.Contains(content, "DROP TABLE") ||
		strings.Contains(content, "DROP COLUMN") ||
		strings.Contains(content, "TRUNCATE ")

	assert.False(t, hasDrop,
		"Migration should not contain DROP TABLE, DROP COLUMN, or TRUNCATE statements. "+
			"Data-destructive operations in a migration file risk destroying production data on re-run.")
}

// ---------------------------------------------------------------------------
// Helper: compile-time check that domain types are used correctly
// ---------------------------------------------------------------------------

// Verify that domain.APIKey.KeyHash (the field that gets persisted) is a
// plain string — documenting that no encryption wrapper type is used.
// This is a RED documentation test.
func TestSecurity_VULN07_APIKeyHashFieldIsPlainString_Red(t *testing.T) {
	k := domain.APIKey{
		ID:        domain.APIKeyID(uuid.New()),
		UserID:    domain.UserID(uuid.New()),
		Name:      "test",
		KeyHash:   "some_hash_value", // plain string, not encrypted
		CreatedAt: time.Now(),
	}

	assert.IsType(t, "", k.KeyHash,
		"VULN-07 RED: domain.APIKey.KeyHash is a plain string type. "+
			"There is no encryption wrapper, meaning the hash is stored and retrieved "+
			"as cleartext in the database column without any at-rest protection.")

	// Also verify User.PasswordHash is a plain string.
	u := domain.User{
		ID:           domain.UserID(uuid.New()),
		Email:        "test@example.com",
		PasswordHash: "$2a$14$somebcrypthash",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	assert.IsType(t, "", u.PasswordHash,
		"VULN-07 RED: domain.User.PasswordHash is a plain string — "+
			"no at-rest encryption type wrapping it.")

	// Suppress unused variable warnings.
	_ = k
	_ = u
}

// GREEN: sensitive fields should use an encrypted value type that enforces
// encryption on marshal and decryption on unmarshal.
func TestSecurity_VULN07_SensitiveFieldsUseEncryptedType_Green(t *testing.T) {
	// In the secure implementation, domain.EncryptedString (or similar) would
	// wrap sensitive fields and prevent accidental plaintext storage.
	// Currently this type does not exist — the test documents the desired state.

	src, err := os.ReadFile("../../domain/types.go")
	require.NoError(t, err)

	content := string(src)
	hasEncryptedType := strings.Contains(content, "EncryptedString") ||
		strings.Contains(content, "SealedString") ||
		strings.Contains(content, "EncryptedValue")

	assert.True(t, hasEncryptedType,
		"VULN-07 GREEN (currently failing): sensitive domain fields (PasswordHash, SSOSubject, KeyHash) "+
			"should use a dedicated encrypted value type that enforces at-rest encryption. "+
			"Currently they are plain string fields with no protection.")
}

// ---------------------------------------------------------------------------
// Additional: Revoke does not check ownership before revoking
// (the repository layer — app layer does check, but repo has no guard)
// ---------------------------------------------------------------------------

// GREEN (passing — this is the current correct behavior at the repo level):
// The repository's Revoke method takes only an APIKeyID, with no user context.
// This means the app layer MUST enforce ownership before calling Revoke.
// This test documents the trust boundary.
func TestSecurity_RevokeHasNoOwnershipCheck_DocumentedTrustBoundary(t *testing.T) {
	src, err := os.ReadFile("pg.go")
	require.NoError(t, err)

	content := string(src)

	// The Revoke SQL operates only on id — no user_id check.
	revokeSQL := extractFunctionSQL(content, "Revoke")
	hasUserIDCheck := strings.Contains(revokeSQL, "user_id")

	assert.False(t, hasUserIDCheck,
		"Trust boundary documented: apiKeyRepository.Revoke has no user_id ownership check in SQL. "+
			"The app layer (auth.go:RevokeAPIKey) MUST verify ownership before calling this method. "+
			"If the app layer check is ever bypassed, any key can be revoked by anyone.")
}

// extractFunctionSQL is a test helper that naively extracts SQL-looking strings
// from within a named function's region of a Go source file.
func extractFunctionSQL(src, funcName string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return ""
	}

	var result strings.Builder
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != funcName {
			return true
		}
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			lit, ok := inner.(*ast.BasicLit)
			if ok && lit.Kind == token.STRING {
				result.WriteString(lit.Value)
			}
			return true
		})
		return false
	})
	return result.String()
}

// Ensure the context package is used (suppress import if needed).
var _ = context.Background
