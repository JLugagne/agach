package security_test

// Additional RED security tests for cmd/agach-server — round 2.
//
// These tests cover vulnerabilities NOT already documented in server_security_test.go.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- SEC-SRV-06 -------------------------------------------------------------
// writeDefaultConfig creates the config file with mode 0644 (world-readable).
// This file can contain SSO configuration, rate limiting parameters, and
// daemon JWT TTL settings. On a shared system, other users can read these
// settings.
//
// File: cmd/agach-server/config.go line 33 — os.WriteFile(path, data, 0644)

// TestSecurity_RED_ConfigFileWrittenWorldReadable asserts that the production
// code must write the config file with 0600 (owner-only) permissions.
// This test FAILS today because config.go uses 0644.
// Fix: change os.WriteFile(path, data, 0644) to os.WriteFile(path, data, 0600).
func TestSecurity_RED_ConfigFileWrittenWorldReadable(t *testing.T) {
	sourceFile := filepath.Join(
		findProjectRoot(t),
		"cmd", "agach-server", "config.go",
	)
	data, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "must be able to read cmd/agach-server/config.go")
	source := string(data)

	// The file must not be written with 0644 (group- and world-readable).
	assert.False(t, strings.Contains(source, "WriteFile(path, data, 0644)"),
		"SEC-SRV-06: config.go must not use 0644 for WriteFile — "+
			"config file may contain SSO configuration; fix: use 0600")

	// The file must be written with 0600 (owner-read/write only).
	isWorldReadable := !strings.Contains(source, "WriteFile(path, data, 0600)")
	assert.False(t, isWorldReadable,
		"SEC-SRV-06: config.go must use 0600 for the config file so only the "+
			"owner can read it; current code uses 0644 which exposes SSO config to "+
			"other local users")
}

// ---- SEC-SRV-07 -------------------------------------------------------------
// loadConfig does not validate the parsed YAML structure. A config file with
// unexpected or malicious fields is silently accepted. Specifically:
// - No validation of DaemonJWTTTL (can be negative or zero)
// - No validation of AuthRateLimitPerSecond (can be zero, disabling rate limiting)
// - No validation of AuthRateLimitBurst (can be zero or negative)
//
// File: cmd/agach-server/config.go lines 40-55

// TestSecurity_RED_ZeroRateLimitDisablesProtection asserts that loadConfig must
// validate AuthRateLimitPerSecond > 0. This test FAILS today because no such
// validation exists in the production code.
// Fix: after Unmarshal, check that cfg.AuthRateLimitPerSecond > 0 when set,
// and return an error or clamp to a safe default.
func TestSecurity_RED_ZeroRateLimitDisablesProtection(t *testing.T) {
	sourceFile := filepath.Join(
		findProjectRoot(t),
		"cmd", "agach-server", "config.go",
	)
	data, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "must be able to read cmd/agach-server/config.go")
	source := string(data)

	// loadConfig must validate that rate limit values are positive.
	hasRateLimitValidation := strings.Contains(source, "AuthRateLimitPerSecond") &&
		(strings.Contains(source, "AuthRateLimitPerSecond > 0") ||
			strings.Contains(source, "AuthRateLimitPerSecond <= 0") ||
			strings.Contains(source, "invalid") && strings.Contains(source, "rate"))
	assert.True(t, hasRateLimitValidation,
		"SEC-SRV-07: loadConfig in config.go must validate that "+
			"AuthRateLimitPerSecond > 0 when set; a zero value disables brute-force "+
			"protection; fix: return error or set safe default when value is zero or negative")
}

// ---- SEC-SRV-08 -------------------------------------------------------------
// SPA handler path traversal — the spaHandler in main.go uses
// strings.TrimPrefix(r.URL.Path, "/") then checks fs.Stat(h.fs, path).
// While the embedded FS should be safe, the path is not cleaned with
// filepath.Clean or path.Clean before the stat call. A request with encoded
// path separators or null bytes could potentially bypass the embedded FS
// boundary check.
//
// File: cmd/agach-server/main.go lines 167-201

// TestSecurity_RED_SPAHandlerNoPathCleaning asserts that the spaHandler must
// call path.Clean (or filepath.Clean) on the URL path before using it with
// fs.Stat. This test FAILS today because the production spaHandler does not
// clean the path.
// Fix: apply path.Clean() to the trimmed path before fs.Stat.
func TestSecurity_RED_SPAHandlerNoPathCleaning(t *testing.T) {
	sourceFile := filepath.Join(
		findProjectRoot(t),
		"cmd", "agach-server", "main.go",
	)
	data, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "must be able to read cmd/agach-server/main.go")
	source := string(data)

	// The spaHandler must sanitize the path with path.Clean or filepath.Clean
	// before passing it to fs.Stat, to prevent path traversal sequences.
	hasPathCleaning := strings.Contains(source, "path.Clean(") ||
		strings.Contains(source, "filepath.Clean(")
	assert.True(t, hasPathCleaning,
		"SEC-SRV-08: spaHandler in main.go must call path.Clean() on the URL path "+
			"before fs.Stat to guard against path traversal sequences like "+
			"'assets/../../../etc/passwd'; fix: add path.Clean() after TrimPrefix")
}
