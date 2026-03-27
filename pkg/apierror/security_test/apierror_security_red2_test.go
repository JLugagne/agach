package security_test

// Additional RED security tests for pkg/apierror — round 2.
//
// These tests cover vulnerabilities NOT already documented in apierror_security_test.go.

import (
	"encoding/json"
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/stretchr/testify/assert"
)

// ---- VULNERABILITY 4 --------------------------------------------------------
// Nil receiver panic — calling Error(), ErrorCode(), or ErrorMessage() on a
// nil *apierror.Error pointer causes a nil pointer dereference panic. Since
// apierror.Error is a public type used across package boundaries, a nil value
// can easily propagate through error interfaces.
//
// File: pkg/apierror/apierror.go lines 14-36

// TestSecurity_RED_NilReceiverPanic asserts that calling methods on a nil
// *apierror.Error does NOT panic — they should return safe defaults.
// This test currently FAILS because the production code has no nil receiver guards.
// TODO(security): add nil receiver checks to Error(), ErrorCode(), ErrorMessage()
func TestSecurity_RED_NilReceiverPanic(t *testing.T) {
	var nilErr *apierror.Error

	assert.NotPanics(t, func() {
		_ = nilErr.Error()
	}, "RED: calling Error() on nil *apierror.Error must not panic — should return safe default")

	assert.NotPanics(t, func() {
		_ = nilErr.ErrorCode()
	}, "RED: calling ErrorCode() on nil *apierror.Error must not panic — should return safe default")

	assert.NotPanics(t, func() {
		_ = nilErr.ErrorMessage()
	}, "RED: calling ErrorMessage() on nil *apierror.Error must not panic — should return safe default")
}

// ---- VULNERABILITY 5 --------------------------------------------------------
// JSON serialization of apierror.Error includes the Err field (exported) which
// may contain internal error details. If an apierror.Error is ever marshaled
// to JSON (e.g., in error logging, debug endpoints), the internal cause is
// exposed.
//
// File: pkg/apierror/apierror.go — Error struct has exported Err field with no json tag

// TestSecurity_RED_JSONMarshalExposesInternalError asserts that JSON
// marshaling of apierror.Error does NOT expose the Err field.
// This test currently FAILS because the Err field has no json:"-" tag.
// TODO(security): add `json:"-"` tag to the Err field, or make it unexported
func TestSecurity_RED_JSONMarshalExposesInternalError(t *testing.T) {
	e := &apierror.Error{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
	}

	data, err := json.Marshal(e)
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(data, &decoded))

	_, hasErr := decoded["Err"]

	// Correct behavior: the Err key must NOT be present in the JSON output.
	// Internal errors must not be serialized to JSON to prevent leaking
	// internal implementation details.
	assert.False(t, hasErr,
		"RED: Err field must not be present in JSON output — internal errors must not leak via JSON serialization")
}
