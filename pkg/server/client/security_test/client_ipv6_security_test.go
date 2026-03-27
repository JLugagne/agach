package security_test

// Security tests for pkg/server/client/client.go — IPv6 SSRF gaps.
//
// These RED tests assert CORRECT safe behaviour that is not yet implemented;
// they are expected to FAIL against the current production code.
// GREEN test for fe80:: is included because that address is already blocked.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/pkg/server/client"
)

// ─── VULNERABILITY: IPv6 loopback and link-local not blocked ────────────────
// The SSRF check in client.New() only examines IPv4 addresses and specific
// hostnames. IPv6 loopback (::1) and IPv6 link-local addresses (fe80::) are
// not blocked, allowing SSRF via IPv6.
//
// File: pkg/server/client/client.go lines 60-71

// TestSecurity_RED_SSRF_IPv6LoopbackNotBlocked asserts that the IPv6 loopback
// address [::1] is rejected by New() and that any subsequent method call
// returns an error.
// Currently [::1] passes the SSRF filter; this test will FAIL until IPv6
// loopback blocking is implemented.
// TODO(security): check for IPv6 loopback and link-local addresses in SSRF filter
func TestSecurity_RED_SSRF_IPv6LoopbackNotBlocked(t *testing.T) {
	c := client.New("http://[::1]:8080")
	require.NotNil(t, c, "client.New always returns a non-nil *Client")

	// The client must reject [::1] at construction; the first method call must
	// return an error indicating the address is blocked.
	_, err := c.ListProjects()
	require.Error(t, err,
		"RED: IPv6 loopback [::1] must be blocked — currently New() accepts it and the request proceeds")
	assert.Contains(t, err.Error(), "blocked",
		"RED: the error for [::1] must mention 'blocked' to confirm SSRF protection is active")
}

// TestSecurity_RED_SSRF_IPv6LinkLocalNotBlocked asserts that IPv6 link-local
// addresses (fe80::) are rejected by New() and that any subsequent method
// call returns an error.
// The production code already blocks fe80:: via IsLinkLocalUnicast(); this
// test documents the required observable behaviour: the error must confirm
// the address was blocked (not merely fail with a connection error).
// TODO(security): block IPv6 link-local (fe80::/10) in SSRF filter
func TestSecurity_RED_SSRF_IPv6LinkLocalNotBlocked(t *testing.T) {
	c := client.New("http://[fe80::1]:8080")
	require.NotNil(t, c, "client.New always returns a non-nil *Client")

	// The client must have set an error for this link-local address; the first
	// method call must surface it.  The error must mention that the address is
	// blocked — a generic connection error is not sufficient because it does
	// not confirm that the SSRF protection fired.
	_, err := c.ListProjects()
	require.Error(t, err,
		"RED: IPv6 link-local [fe80::1] must be blocked at construction — currently client.New() accepts it")
	assert.Contains(t, err.Error(), "blocked",
		"RED: the error for [fe80::1] must mention 'blocked' to confirm SSRF protection — currently the error message does not")
}
