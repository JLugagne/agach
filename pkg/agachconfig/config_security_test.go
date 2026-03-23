package agachconfig_test

// Security regression tests for pkg/agachconfig — RED (current-state) tests.
//
// A RED test documents a vulnerability that exists TODAY.  The test PASSES
// when run against unfixed code, asserting the broken/insecure behaviour.
// When a vulnerability is fixed, the corresponding RED assertion will flip
// (the test will fail) — that is the intended signal to the developer.
//
// GREEN tests (desired safe behaviour) live in config_security_green_test.go
// behind the "securityfix" build tag; they compile and pass only after the
// fixes are applied:
//
//   go test -tags securityfix -race ./pkg/agachconfig/...

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/agachconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-1 / VULN-7 — Relative directory and path traversal accepted by Load
//
// config.go:41 — Load(dir string) performs no check that dir is absolute or
// clean.  An attacker-controlled dir value can direct the walk to arbitrary
// locations on the filesystem.
// ─────────────────────────────────────────────────────────────────────────────

// RED: Load resolves ".." components in the supplied directory path.
// filepath.Join cleans the path, so "/a/b/c/../../../" → "/a" — still an
// absolute path.  Load then walks upward from that cleaned absolute path.
// The vulnerability is that a caller controlling the initial dir can aim Load
// at any subtree, including one that contains a malicious .agach.yml.
func TestSecurity_RED_PathTraversal_DotDotResolved(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(root, ".agach.yml"),
		[]byte("base_url: http://evil.example.com\napi_key: exfiltrated\n"),
		0600,
	))

	deep := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deep, 0755))

	// filepath.Join cleans ".." so the result is still absolute (points to root).
	traversal := filepath.Join(deep, "..", "..", "..")

	cfg, err := agachconfig.Load(traversal)
	// Vulnerability: Load accepts the cleaned absolute path and loads the
	// attacker-controlled config from root without any boundary check.
	assert.NoError(t, err,
		"RED (vulnerability): Load follows cleaned '..' path to planted config")
	if cfg != nil {
		assert.Equal(t, "http://evil.example.com", cfg.ResolvedBaseURL(),
			"RED (vulnerability): attacker config loaded via path traversal")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-2 — Unbounded upward filesystem walk (no depth limit)
//
// config.go:51-56 — The walk loop has no depth cap and no boundary check.
// An attacker who plants a .agach.yml high in the filesystem hierarchy
// captures any invocation starting Load from a deep subdirectory.
// ─────────────────────────────────────────────────────────────────────────────

// RED: Load walks arbitrarily many levels upward with no limit.
func TestSecurity_RED_UnboundedFSWalk_NoDepthLimit(t *testing.T) {
	plantDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(plantDir, ".agach.yml"),
		[]byte("api_key: root-level-stolen\n"),
		0600,
	))

	// Start Load from 10 levels inside plantDir.
	deep := plantDir
	for i := 0; i < 10; i++ {
		deep = filepath.Join(deep, fmt.Sprintf("level%d", i))
	}
	require.NoError(t, os.MkdirAll(deep, 0755))

	cfg, err := agachconfig.Load(deep)
	// Vulnerability: walks all 10 levels up and finds the planted config.
	assert.NoError(t, err,
		"RED (vulnerability): unbounded walk — no depth limit enforced")
	assert.NotNil(t, cfg,
		"RED (vulnerability): config 10 levels up was loaded")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-3 — Empty API key silently accepted
//
// config.go:36 — ResolvedAPIKey() returns "" with no error. There is no
// Validate() method.  Callers proceed unauthenticated without any indication.
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// VULN-4 — Unset env var silently resolves to empty string
//
// config.go:27 — os.Getenv returns "" when a var is unset.  There is no error
// and no distinction between "intentionally empty" and "var not set".
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// VULN-5 — No file permission check before reading .agach.yml
//
// config.go:45 — os.ReadFile is called without checking os.Stat() mode.
// A world-readable config file exposes the API key to all local users.
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// VULN-6 — Plaintext HTTP base_url accepted without warning
//
// config.go:33 — ResolvedBaseURL() returns any string.  No validation enforces
// HTTPS for remote hosts, so API keys are transmitted in clear text.
// ─────────────────────────────────────────────────────────────────────────────

// RED: http:// URL is returned by ResolvedBaseURL without any complaint.
func TestSecurity_RED_PlaintextHTTP_AcceptedSilently(t *testing.T) {
	cfg := &agachconfig.Config{
		BaseURL: "http://plaintext.example.com",
	}
	url := cfg.ResolvedBaseURL()
	// Vulnerability: no validation; API key will be transmitted in clear text.
	assert.Equal(t, "http://plaintext.example.com", url,
		"RED (vulnerability): plaintext HTTP base_url accepted — credentials sent in clear text")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-8 — Env var injection via YAML value ("$AWS_SECRET_ACCESS_KEY")
//
// config.go:25-29 — resolve() expands ANY string starting with "$" as an env
// var, including values read from .agach.yml.  An attacker who can write to
// the config file can reference arbitrary env vars and exfiltrate them.
// ─────────────────────────────────────────────────────────────────────────────

