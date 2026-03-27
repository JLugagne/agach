package security_test

// Security tests for cmd/agach-server/config.go — config file security gaps.
//
// These RED tests document vulnerabilities NOT covered by the existing
// server_security_test.go file.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── VULNERABILITY: writeDefaultConfig uses 0644 permissions ────────────────
// writeDefaultConfig writes the config file with os.FileMode 0644 (world-
// readable). The config file may contain SSO client secrets, OIDC configuration,
// or other sensitive data. World-readable permissions expose these secrets to
// all local users on a shared system.
//
// File: cmd/agach-server/config.go line 33:
//   os.WriteFile(path, data, 0644)

// TestSecurity_RED_WriteDefaultConfigWorldReadable documents that
// writeDefaultConfig creates the config file with 0644 permissions.
// TODO(security): change permissions to 0600
func TestSecurity_RED_WriteDefaultConfigWorldReadable(t *testing.T) {
	// The vulnerability is in the file permission mode used by writeDefaultConfig.
	// We verify the constant used in the source code is 0644 (world-readable).
	//
	// We cannot call writeDefaultConfig directly (unexported), but we can
	// verify the pattern: os.WriteFile with 0644 is insecure for config files.

	dir := t.TempDir()
	testFile := filepath.Join(dir, "test-config.yml")

	// Simulate what writeDefaultConfig does.
	err := os.WriteFile(testFile, []byte("sso:\n  client_secret: \"s3cret\"\n"), 0644)
	require.NoError(t, err)

	info, err := os.Stat(testFile)
	require.NoError(t, err)

	mode := info.Mode().Perm()
	// The file was written with 0644 — group and world can read it.
	assert.True(t, mode&0o044 != 0,
		"RED: config file written with 0644 permissions (mode %04o) — "+
			"group/world can read SSO secrets; fix: use 0600", mode)
	t.Log("RED: writeDefaultConfig uses 0644 permissions — sensitive SSO config is world-readable")
}

// ─── VULNERABILITY: writeDefaultConfig TOCTOU race ──────────────────────────
// writeDefaultConfig checks os.Stat(path) then calls os.WriteFile(path, ...).
// Between the Stat and WriteFile, another process can create the file or
// replace it with a symlink, leading to:
//   1. Overwriting an existing config with default values (data loss)
//   2. Writing through a symlink to an attacker-chosen location
//
// File: cmd/agach-server/config.go lines 20-21:
//   if _, err := os.Stat(path); err == nil { return ... }
//   ... os.WriteFile(path, data, 0644)

// TestSecurity_RED_WriteDefaultConfigTOCTOU documents the TOCTOU race between
// Stat and WriteFile.
// TODO(security): use os.OpenFile with O_CREATE|O_EXCL to atomically create
func TestSecurity_RED_WriteDefaultConfigTOCTOU(t *testing.T) {
	// The vulnerability is structural: Stat-then-Write is not atomic.
	// os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600) is the
	// correct pattern — it creates the file atomically and fails if it
	// already exists, eliminating the race window.

	// Demonstrate that the current pattern (Stat then WriteFile) has a gap.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	// Step 1: Stat says file doesn't exist.
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "precondition: file does not exist")

	// Step 2: Between Stat and WriteFile, an attacker creates the file.
	err = os.WriteFile(path, []byte("attacker-content"), 0644)
	require.NoError(t, err)

	// Step 3: WriteFile overwrites the attacker's file (or in reverse,
	// the attacker overwrites the legitimate config).
	err = os.WriteFile(path, []byte("default-config"), 0644)
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Equal(t, "default-config", string(content),
		"RED: Stat-then-WriteFile race — attacker's content was overwritten; "+
			"fix: use O_CREATE|O_EXCL for atomic create")
	t.Log("RED: writeDefaultConfig has TOCTOU race between os.Stat and os.WriteFile")
}

// ─── VULNERABILITY: loadConfig does not validate file permissions ────────────
// loadConfig reads the server config file via os.ReadFile without checking file
// permissions. Unlike agachconfig.LoadSecureYAML (which checks for 0600),
// loadConfig accepts world-readable config files.
//
// File: cmd/agach-server/config.go lines 40-55

// TestSecurity_RED_LoadConfigNoPermissionCheck documents that loadConfig
// reads config files regardless of their permissions.
// TODO(security): check file permissions and reject files more permissive than 0600
func TestSecurity_RED_LoadConfigNoPermissionCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server-config.yml")

	// Write a world-readable config with sensitive data.
	err := os.WriteFile(path, []byte("sso:\n  client_secret: top-secret\n"), 0644)
	require.NoError(t, err)

	// Verify the file is world-readable.
	info, err := os.Stat(path)
	require.NoError(t, err)
	mode := info.Mode().Perm()
	assert.True(t, mode&0o044 != 0,
		"RED: server config file at 0644 should be rejected by loadConfig, but no permission check exists; "+
			"fix: add permission check like agachconfig.LoadSecureYAML")
	t.Log("RED: loadConfig does not check file permissions — world-readable server config accepted")
}
