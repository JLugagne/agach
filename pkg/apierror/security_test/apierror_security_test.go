package security_test

// Security tests for pkg/apierror.
//
// Each "RED" test name signals a currently-exploitable behaviour that PASSES
// with the unpatched code (i.e. the vulnerability is real).
//
// Each "GREEN" test name signals the safe state that SHOULD hold after a fix.
// GREEN tests that exercise a missing control currently FAIL — they act as a
// regression gate: once production code is hardened they turn green.
//
// Vulnerabilities covered:
//
//   VULN-1  Exported Err field exposes raw internal error to any caller
//   VULN-2  Error() falls back to Code when Message is empty, leaking
//             technical/internal strings as the public error message
//   VULN-3  Unwrap() at the public HTTP boundary lets callers traverse the
//             full internal error chain via errors.As / errors.Is

import (
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// VULN-1 — Exported Err field exposes raw internal error directly
// ---------------------------------------------------------------------------

// RED: demonstrates that a caller outside the package can read the raw
// internal cause from the exported Err field without using Unwrap.
// This test PASSES with current code, proving the vulnerability is real.
func TestSecurity_RED_ErrFieldIsPubliclyReadable(t *testing.T) {
	internalCause := errors.New("pq: relation \"users\" does not exist")
	e := &apierror.Error{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
		Err:     internalCause,
	}

	// The vulnerability: the exported Err field hands the caller a reference
	// to the raw internal error without any indirection.
	assert.Equal(t, internalCause, e.Err,
		"RED: Err field is publicly readable — internal DB error is exposed to any caller")
}

// GREEN: the safe contract — Error() must never surface the internal cause text
// regardless of what is stored in Err.
func TestSecurity_GREEN_ErrorStringDoesNotExposeInternalCause(t *testing.T) {
	internalCause := errors.New("pq: relation \"users\" does not exist")
	e := &apierror.Error{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
		Err:     internalCause,
	}

	publicString := e.Error()

	assert.NotContains(t, publicString, "pq:",
		"GREEN: Error() must not contain internal DB driver prefix")
	assert.NotContains(t, publicString, "relation",
		"GREEN: Error() must not contain internal schema detail")
	assert.Equal(t, "An internal error occurred", publicString,
		"GREEN: Error() must return only the safe user-facing Message")
}

// ---------------------------------------------------------------------------
// VULN-2 — Error() falls back to Code when Message is empty,
//           leaking technical / internal codes as the public string
// ---------------------------------------------------------------------------

// GREEN: Error() should return a generic safe fallback — NEVER the raw Code —
// when Message is empty.
// NOTE: This test currently FAILS. It documents the required safe behaviour.
func TestSecurity_GREEN_ErrorReturnsGenericFallbackNotCodeWhenMessageEmpty(t *testing.T) {
	e := &apierror.Error{
		Code:    "sql: no rows in result set",
		Message: "",
	}

	publicString := e.Error()

	assert.NotEqual(t, "sql: no rows in result set", publicString,
		"GREEN: Error() must not expose the Code value when Message is absent")
	assert.NotContains(t, publicString, "sql:",
		"GREEN: Error() must not leak SQL-level detail when Message is empty")
}

// GREEN: when both Code and Message are present, Error() must return Message
// (this is already true — test acts as a non-regression guard).
func TestSecurity_GREEN_ErrorReturnsMessageNotCode(t *testing.T) {
	e := &apierror.Error{
		Code:    "INTERNAL_CONSTRAINT_VIOLATION_users_email_key",
		Message: "that email is already in use",
	}

	assert.Equal(t, "that email is already in use", e.Error(),
		"GREEN: Error() must return Message, not the internal Code string")
}

// ---------------------------------------------------------------------------
// VULN-3 — Unwrap() at the HTTP boundary lets callers traverse the full
//           internal error chain
// ---------------------------------------------------------------------------

// GREEN: at the public HTTP response boundary, Unwrap() should return nil so
// that internal error chains cannot be traversed by callers of the boundary
// error value.
// NOTE: This test currently FAILS because Unwrap() returns the stored Err.
// It documents the required safe state after the fix.
func TestSecurity_GREEN_UnwrapReturnsNilAtPublicBoundary(t *testing.T) {
	rawCause := errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")

	e := &apierror.Error{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
		Err:     rawCause,
	}

	// After the fix: Unwrap() at the public boundary returns nil.
	// Internal errors are captured for logging, not for exposure via the chain.
	assert.Nil(t, e.Unwrap(),
		"GREEN: Unwrap() must return nil to prevent traversal of the internal error chain across the HTTP boundary")
}

// GREEN: errors.Is must NOT be able to reach internal causes through a
// public boundary apierror.Error.
// NOTE: This test currently FAILS because Unwrap() propagates the chain.
func TestSecurity_GREEN_ErrorsIsCannotReachInternalCause(t *testing.T) {
	internalSentinel := errors.New("internal sentinel — must not be visible outside")

	e := &apierror.Error{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
		Err:     internalSentinel,
	}

	assert.False(t, errors.Is(e, internalSentinel),
		"GREEN: errors.Is must not traverse the internal chain through a public boundary apierror.Error")
}
