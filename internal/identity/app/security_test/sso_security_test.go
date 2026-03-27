package security_test

// Security tests for SSO/OIDC vulnerabilities.

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-21: OIDC id_token validation accepts HMAC-signed tokens
// File: internal/identity/app/sso.go:181-183
//
// The validateIDToken key function accepts jwt.SigningMethodHMAC and returns
// the app's own JWT secret as the signing key. This means an attacker who
// knows or guesses the HMAC secret can forge id_tokens that appear valid.
//
// In a proper OIDC flow, id_tokens should only be validated with the IdP's
// public key (RSA or ECDSA from the JWKS endpoint). Accepting HMAC allows
// a "key confusion" attack where the attacker uses the shared secret
// (client_secret or JWT secret) to sign a forged id_token.
//
// See: https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_OIDCValidateIDTokenAcceptsHMAC documents that the OIDC
// id_token validator accepts HMAC-signed tokens using the app's JWT secret.
func TestSecurity_OIDCValidateIDTokenAcceptsHMAC(t *testing.T) {
	// We verify this by source inspection since the SSO service requires
	// an HTTP server for OIDC discovery which is complex to mock.
	_, thisFile, _, _ := runtime.Caller(0)
	ssoFile := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "sso.go")
	src, err := os.ReadFile(ssoFile)
	if err != nil {
		t.Skipf("Cannot read sso.go: %v", err)
	}

	content := string(src)

	// The vulnerability is the acceptance of SigningMethodHMAC in the key function
	// of validateIDToken. A secure implementation should NOT have this.
	acceptsHMAC := strings.Contains(content, "SigningMethodHMAC") &&
		strings.Contains(content, "validateIDToken")

	// RED: The code currently accepts HMAC signing for OIDC tokens.
	assert.False(t, acceptsHMAC,
		"RED: validateIDToken accepts HMAC-signed id_tokens (sso.go:181-183). "+
			"An attacker can forge id_tokens signed with the app's JWT/HMAC secret. "+
			"Remove the SigningMethodHMAC case from the key function.")
	t.Log("RED: OIDC id_token validation accepts HMAC signing — JWT key confusion attack possible")
}
