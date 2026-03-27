package security_test

// Security tests for pkg/server/client/client.go — IPv6 SSRF gaps.
//
// These RED tests document vulnerabilities NOT covered by the existing
// client_security_test.go or client_security_red2_test.go files.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/pkg/server/client"
)

// ─── VULNERABILITY: IPv6 loopback and link-local not blocked ────────────────
// The SSRF check in client.New() only examines IPv4 addresses and specific
// hostnames. IPv6 loopback (::1) and IPv6 link-local addresses (fe80::) are
// not blocked, allowing SSRF via IPv6.
//
// File: pkg/server/client/client.go lines 60-71

// TestSecurity_RED_SSRF_IPv6LoopbackNotBlocked documents that IPv6 loopback
// address [::1] is accepted by client.New().
// TODO(security): check for IPv6 loopback and link-local addresses in SSRF filter
func TestSecurity_RED_SSRF_IPv6LoopbackNotBlocked(t *testing.T) {
	c := client.New("http://[::1]:8080")
	require.NotNil(t, c)

	// The client was created without error — the IPv6 loopback was not blocked.
	_, err := c.ListProjects()
	if err != nil {
		// Connection error is expected (no server), but the point is
		// that New() accepted the address without blocking it.
		t.Logf("RED: IPv6 loopback [::1] accepted by client.New() — request failed with: %v", err)
	}
	t.Log("RED: IPv6 loopback address [::1] is not blocked by SSRF protection")
}

// TestSecurity_RED_SSRF_IPv6LinkLocalNotBlocked documents that IPv6 link-local
// addresses (fe80::) are accepted by client.New().
// TODO(security): block IPv6 link-local (fe80::/10) in SSRF filter
func TestSecurity_RED_SSRF_IPv6LinkLocalNotBlocked(t *testing.T) {
	c := client.New("http://[fe80::1]:8080")
	require.NotNil(t, c)

	_, err := c.ListProjects()
	if err != nil {
		t.Logf("RED: IPv6 link-local [fe80::1] accepted by client.New() — request failed with: %v", err)
	}
	t.Log("RED: IPv6 link-local address [fe80::1] is not blocked by SSRF protection")
}
