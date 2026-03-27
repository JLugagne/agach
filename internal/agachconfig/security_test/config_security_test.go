package security_test

// Security regression tests for pkg/agachconfig.
//
// Each test asserts the CORRECT/SECURE behaviour — the test FAILS when
// production code has the vulnerability, and PASSES once the fix is in place.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/agachconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-1 / VULN-7 — Relative directory and path traversal accepted by Load
//
// Load(dir string) must reject non-absolute dir values.  filepath.Join cleans
// ".." components before Load sees the path, so the remaining concern is that
// a relative string (not yet cleaned by the caller) must be rejected outright.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_PathTraversal_RelativeStringRejected verifies that Load rejects
// a raw relative path-traversal string (before any filepath.Join cleaning).
// This complements TestSecurity_GREEN_Load_RejectsRelativeDir in the green suite.
func TestSecurity_PathTraversal_RelativeStringRejected(t *testing.T) {
	// These raw strings are relative — Load must reject them.
	for _, rel := range []string{".", "..", "../../etc", "relative/path"} {
		_, err := agachconfig.Load(rel)
		assert.Error(t, err,
			"Load must reject relative dir %q", rel)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-2 — Unbounded upward filesystem walk (no depth limit)
//
// Load must stop walking upward after a small maximum depth.  A config planted
// many levels above the starting directory must NOT be loaded.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_UnboundedFSWalk_MaxDepthEnforced verifies that Load does not
// walk more than a bounded number of levels upward.  The config is planted at
// the root of the temp tree and Load starts 10 levels deep.  If the max depth
// is smaller than 10, the config must not be found.
func TestSecurity_RED_UnboundedFSWalk_MaxDepthEnforced(t *testing.T) {
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
	require.NoError(t, err)
	assert.Nil(t, cfg,
		"Load must not walk 10 levels up — max depth must be enforced (vulnerability: unbounded walk)")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-6 — Plaintext HTTP base_url accepted without warning
//
// Validate() must reject http:// URLs pointing to remote (non-localhost) hosts.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_PlaintextHTTP_RejectedByValidate verifies that Validate returns
// an error for an http:// base_url targeting a remote host.
func TestSecurity_PlaintextHTTP_RejectedByValidate(t *testing.T) {
	cfg := &agachconfig.Config{
		BaseURL: "http://plaintext.example.com",
	}
	err := cfg.Validate()
	assert.Error(t, err,
		"Validate must reject http:// base_url for a remote host — credentials would be sent in clear text")
}
