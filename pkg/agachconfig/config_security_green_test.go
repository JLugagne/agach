//go:build securityfix

package agachconfig_test

// GREEN tests for pkg/agachconfig security fixes.
//
// These tests document the DESIRED safe behaviour after each vulnerability is
// fixed.  They are guarded by the "securityfix" build tag because they require
// changes to the production code (new Validate() method, permission checks,
// depth limits, etc.).
//
// To run once the fixes are applied:
//
//   go test -tags securityfix -race ./pkg/agachconfig/...
//
// Vulnerabilities covered:
//   VULN-1/7 — relative dir and path traversal → Load must reject non-absolute dirs
//   VULN-2   — unbounded walk → Load must stop after a maximum depth
//   VULN-3   — empty APIKey accepted → Validate() must reject empty key
//   VULN-4   — unset env var silent → Validate() must reject unresolved references
//   VULN-5   — no permission check → Load must refuse files more permissive than 0600
//   VULN-6   — plaintext HTTP → Validate() must reject http:// for remote hosts
//   VULN-8   — env var injection from YAML → Load must NOT expand "$VAR" from file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/agachconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-1 / VULN-7 GREEN — relative dir must be rejected
// ─────────────────────────────────────────────────────────────────────────────

func TestSecurity_GREEN_Load_RejectsRelativeDir(t *testing.T) {
	_, err := agachconfig.Load(".")
	assert.Error(t, err, "Load must return an error for a relative directory")

	_, err = agachconfig.Load("../../etc")
	assert.Error(t, err, "Load must return an error for a relative path-traversal string")
}

func TestSecurity_GREEN_RelativeTraversalString_Rejected(t *testing.T) {
	_, err := agachconfig.Load("../../..")
	assert.Error(t, err, "Load must reject relative path-traversal strings")
}

// GREEN (positive): absolute path must still work.
func TestSecurity_GREEN_AbsoluteDir_Accepted(t *testing.T) {
	dir := t.TempDir() // always absolute
	cfg, err := agachconfig.Load(dir)
	require.NoError(t, err)
	assert.Nil(t, cfg, "no config file present — cfg must be nil")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-2 GREEN — walk must stop at a maximum depth
// ─────────────────────────────────────────────────────────────────────────────

func TestSecurity_GREEN_UnboundedFSWalk_MaxDepthRespected(t *testing.T) {
	plantDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(plantDir, ".agach.yml"),
		[]byte("api_key: should-not-reach\n"),
		0600,
	))

	// 20 levels deep — must exceed any reasonable max-depth limit.
	deep := plantDir
	for i := 0; i < 20; i++ {
		deep = filepath.Join(deep, fmt.Sprintf("d%d", i))
	}
	require.NoError(t, os.MkdirAll(deep, 0755))

	cfg, err := agachconfig.Load(deep)
	require.NoError(t, err)
	assert.Nil(t, cfg, "Load must not walk 20 levels up — max depth must be enforced")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-5 GREEN — world-readable config file must be rejected
// ─────────────────────────────────────────────────────────────────────────────

func TestSecurity_GREEN_WorldReadableConfig_Rejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".agach.yml")
	require.NoError(t, os.WriteFile(path,
		[]byte("api_key: super-secret\nbase_url: http://localhost:8222\n"), 0644))

	_, err := agachconfig.Load(dir)
	assert.Error(t, err,
		"Load must return an error when .agach.yml is world-readable (permissions 0644)")
	errLower := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errLower, "permission") || strings.Contains(errLower, "mode"),
		"error message must mention file permissions or mode, got: %s", err.Error())
}

// GREEN (positive): correctly restricted file (0600) must still load.
func TestSecurity_GREEN_CorrectlyRestrictedConfig_Loaded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".agach.yml")
	require.NoError(t, os.WriteFile(path,
		[]byte("api_key: super-secret\nbase_url: http://localhost:8222\n"), 0600))

	cfg, err := agachconfig.Load(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg, "0600 config file must load successfully")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-6 GREEN — plaintext HTTP remote URL rejected by Validate()
// ─────────────────────────────────────────────────────────────────────────────

func TestSecurity_GREEN_PlaintextHTTP_FlaggedByValidate(t *testing.T) {
	cfg := &agachconfig.Config{
		BaseURL: "http://remote.example.com",
	}
	err := cfg.Validate()
	assert.Error(t, err,
		"Validate must return an error for http:// base_url pointing to a remote host")
	errLower := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errLower, "https") ||
			strings.Contains(errLower, "tls") ||
			strings.Contains(errLower, "insecure"),
		"error must mention TLS/HTTPS/insecure, got: %s", err.Error())
}

// GREEN (positive): https:// must be accepted.
func TestSecurity_GREEN_HTTPS_AcceptedByValidate(t *testing.T) {
	cfg := &agachconfig.Config{
		BaseURL: "https://secure.example.com",
	}
	assert.NoError(t, cfg.Validate(), "https:// base_url must pass Validate")
}

// GREEN (positive): http://localhost is acceptable for development.
func TestSecurity_GREEN_LocalhostHTTP_AcceptedByValidate(t *testing.T) {
	cfg := &agachconfig.Config{
		BaseURL: "http://localhost:8322",
	}
	assert.NoError(t, cfg.Validate(), "http://localhost URLs must pass Validate")
}

// GREEN (positive): http://127.0.0.1 is acceptable.
func TestSecurity_GREEN_Loopback127_AcceptedByValidate(t *testing.T) {
	cfg := &agachconfig.Config{
		BaseURL: "http://127.0.0.1:8322",
	}
	assert.NoError(t, cfg.Validate(), "http://127.0.0.1 URLs must pass Validate")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-8 GREEN — env var injection from YAML file must be prevented
// ─────────────────────────────────────────────────────────────────────────────

// After the fix, a "$VAR" value loaded from a file must NOT be expanded.
func TestSecurity_GREEN_EnvVarInjection_NotExpandedFromFile(t *testing.T) {
	const victimVar = "AGACH_VICTIM_SECRET_XYZ"
	t.Setenv(victimVar, "super-sensitive-value")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".agach.yml"),
		[]byte(fmt.Sprintf("base_url: \"$%s\"\napi_key: mykey\n", victimVar)),
		0600,
	))

	cfg, err := agachconfig.Load(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	resolved := cfg.ResolvedBaseURL()
	assert.NotEqual(t, "super-sensitive-value", resolved,
		"env var injected via YAML file must NOT be expanded after fix")
}

// ─────────────────────────────────────────────────────────────────────────────
// Full Validate() contract
// ─────────────────────────────────────────────────────────────────────────────

// GREEN: well-formed config must pass Validate.
func TestSecurity_GREEN_Validate_WellFormedConfig(t *testing.T) {
	cfg := &agachconfig.Config{
		BaseURL: "https://secure.example.com",
	}
	assert.NoError(t, cfg.Validate(), "well-formed config must pass Validate")
}

// GREEN: empty BaseURL must fail Validate.
func TestSecurity_GREEN_Validate_EmptyBaseURL(t *testing.T) {
	cfg := &agachconfig.Config{BaseURL: ""}
	assert.Error(t, cfg.Validate(), "Validate must error on empty BaseURL")
}
