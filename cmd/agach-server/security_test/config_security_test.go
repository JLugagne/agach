package security_test

// Security tests for cmd/agach-server/config.go — config file security gaps.
//
// These RED tests document vulnerabilities NOT covered by the existing
// server_security_test.go file.

import (
	"os"
	"path/filepath"
	"strings"
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

// TestSecurity_RED_WriteDefaultConfigWorldReadable asserts that writeDefaultConfig
// must create the config file with 0600 permissions (owner-only), not 0644.
// This test FAILS today because the production code uses 0644.
// Fix: change os.WriteFile(path, data, 0644) to os.WriteFile(path, data, 0600).
func TestSecurity_RED_WriteDefaultConfigWorldReadable(t *testing.T) {
	// Read the production source to verify which permission constant is used.
	// The vulnerability is that config.go line 33 passes 0644 to os.WriteFile.
	sourceFile := filepath.Join(
		findProjectRoot(t),
		"cmd", "agach-server", "config.go",
	)
	data, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "must be able to read cmd/agach-server/config.go")
	source := string(data)

	// The production code must NOT use 0644 for the config file.
	assert.False(t, strings.Contains(source, "WriteFile(path, data, 0644)"),
		"config.go must not use 0644 for WriteFile — "+
			"config file may contain SSO secrets; fix: use 0600 (owner-only)")

	// The production code must use 0600 for the config file.
	assert.True(t, strings.Contains(source, "WriteFile(path, data, 0600)"),
		"config.go must use 0600 for WriteFile so the config file is owner-read-only; "+
			"current code uses 0644 which is world-readable")
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

// TestSecurity_RED_WriteDefaultConfigTOCTOU asserts that writeDefaultConfig
// must use atomic O_CREATE|O_EXCL to avoid the TOCTOU race between Stat and
// WriteFile. This test FAILS today because config.go uses the racy Stat+Write
// pattern instead.
// Fix: use os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600).
func TestSecurity_RED_WriteDefaultConfigTOCTOU(t *testing.T) {
	// Read the production source to check which file creation pattern is used.
	sourceFile := filepath.Join(
		findProjectRoot(t),
		"cmd", "agach-server", "config.go",
	)
	data, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "must be able to read cmd/agach-server/config.go")
	source := string(data)

	// The production code must not use the racy Stat-then-WriteFile pattern.
	assert.False(t, strings.Contains(source, "os.Stat(path)") && strings.Contains(source, "os.WriteFile(path"),
		"config.go must not use Stat+WriteFile (TOCTOU race); "+
			"fix: use os.OpenFile with O_CREATE|O_EXCL for atomic file creation")

	// The production code must use O_EXCL for atomic creation.
	assert.True(t, strings.Contains(source, "O_EXCL"),
		"config.go must use O_CREATE|O_EXCL to atomically create the config file; "+
			"the current Stat-then-WriteFile pattern has a TOCTOU race window")
}

// ─── VULNERABILITY: loadConfig does not validate file permissions ────────────
// loadConfig reads the server config file via os.ReadFile without checking file
// permissions. Unlike agachconfig.LoadSecureYAML (which checks for 0600),
// loadConfig accepts world-readable config files.
//
// File: cmd/agach-server/config.go lines 40-55

// TestSecurity_RED_LoadConfigNoPermissionCheck asserts that loadConfig must
// reject config files that are more permissive than 0600. This test FAILS today
// because no permission check exists in the production code.
// Fix: add a permission check using os.Stat before reading, reject if mode&0o077 != 0.
func TestSecurity_RED_LoadConfigNoPermissionCheck(t *testing.T) {
	// Read the production source to check whether a permission check is present.
	sourceFile := filepath.Join(
		findProjectRoot(t),
		"cmd", "agach-server", "config.go",
	)
	data, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "must be able to read cmd/agach-server/config.go")
	source := string(data)

	// loadConfig must check file permissions before reading.
	// The secure pattern checks that mode&0o077 == 0 (only owner can read/write).
	hasPermCheck := strings.Contains(source, "0o077") ||
		strings.Contains(source, "0077") ||
		strings.Contains(source, "LoadSecureYAML") ||
		strings.Contains(source, "Mode().Perm()")
	assert.True(t, hasPermCheck,
		"loadConfig in config.go must check file permissions before reading; "+
			"world-readable config files containing SSO secrets should be rejected; "+
			"fix: add os.Stat check and reject files with mode & 0o077 != 0")
}

// findProjectRoot walks up from the test binary's working directory to find
// the Go module root (the directory containing go.mod).
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod not found)")
		}
		dir = parent
	}
}
