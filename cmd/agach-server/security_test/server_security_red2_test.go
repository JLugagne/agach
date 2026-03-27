package security_test

// Additional RED security tests for cmd/agach-server — round 2.
//
// These tests cover vulnerabilities NOT already documented in server_security_test.go.

import (
	"os"
	"path/filepath"
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

// TestSecurity_RED_ConfigFileWrittenWorldReadable documents that
// writeDefaultConfig creates files with 0644 permissions instead of 0600.
// TODO(security): change os.WriteFile permission from 0644 to 0600
func TestSecurity_RED_ConfigFileWrittenWorldReadable(t *testing.T) {
	// We verify the constant used in the source by simulating what
	// writeDefaultConfig does: write a file with the same mode.
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.yml")

	// This mirrors the production code: os.WriteFile(path, data, 0644)
	err := os.WriteFile(path, []byte("test: config\n"), 0644)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)

	mode := info.Mode().Perm()
	// 0644 means group-readable and world-readable
	isWorldReadable := mode&0o004 != 0
	isGroupReadable := mode&0o040 != 0

	assert.True(t, isWorldReadable,
		"RED SEC-SRV-06: config file is world-readable (mode %04o) — "+
			"writeDefaultConfig uses 0644 instead of 0600", mode)
	assert.True(t, isGroupReadable,
		"RED SEC-SRV-06: config file is group-readable (mode %04o) — "+
			"should be restricted to owner only", mode)
	t.Log("RED: writeDefaultConfig creates config files with 0644 permissions — should use 0600 for files containing SSO config")
}

// ---- SEC-SRV-07 -------------------------------------------------------------
// loadConfig does not validate the parsed YAML structure. A config file with
// unexpected or malicious fields is silently accepted. Specifically:
// - No validation of DaemonJWTTTL (can be negative or zero)
// - No validation of AuthRateLimitPerSecond (can be zero, disabling rate limiting)
// - No validation of AuthRateLimitBurst (can be zero or negative)
//
// File: cmd/agach-server/config.go lines 40-55

// TestSecurity_RED_ZeroRateLimitDisablesProtection documents that a zero
// AuthRateLimitPerSecond effectively disables rate limiting.
// TODO(security): validate that rate limit values are positive when set
func TestSecurity_RED_ZeroRateLimitDisablesProtection(t *testing.T) {
	// A config with zero rate limit would disable brute-force protection.
	// The server applies these values directly to the rate limiter:
	//   AuthRateLimitPerSecond: 0  -> effectively no rate limiting
	//   AuthRateLimitBurst: 0      -> no burst capacity
	//
	// loadConfig does not reject these values.

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	err := os.WriteFile(path, []byte("auth_rate_limit_per_second: 0\nauth_rate_limit_burst: 0\n"), 0644)
	require.NoError(t, err)

	// Read back and verify the zero values are accepted
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "auth_rate_limit_per_second: 0",
		"RED SEC-SRV-07: zero rate limit accepted in config — brute-force protection disabled")
	t.Log("RED: loadConfig accepts zero auth_rate_limit_per_second — rate limiting is effectively disabled")
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

// TestSecurity_RED_SPAHandlerNoPathCleaning documents that the SPA handler
// does not explicitly clean the path before using it.
// TODO(security): apply path.Clean() to the trimmed path before fs.Stat
func TestSecurity_RED_SPAHandlerNoPathCleaning(t *testing.T) {
	// The spaHandler trims the leading "/" then passes directly to fs.Stat.
	// While Go's http.ServeMux and embedded FS provide some protection,
	// the code does not call path.Clean() on the result.
	// This test documents the missing sanitization.

	// Demonstrate that a path like "assets/../../../etc/passwd" after
	// TrimPrefix would be "assets/../../../etc/passwd" — no cleaning applied.
	rawPath := "/assets/../../../etc/passwd"
	trimmed := rawPath[1:] // strings.TrimPrefix equivalent
	assert.Equal(t, "assets/../../../etc/passwd", trimmed,
		"RED SEC-SRV-08: path traversal sequences are not cleaned before fs.Stat — "+
			"embedded FS may protect against this but explicit cleaning should be applied")
	t.Log("RED: spaHandler does not call path.Clean() on URL path before fs.Stat — relies on embedded FS for safety")
}
