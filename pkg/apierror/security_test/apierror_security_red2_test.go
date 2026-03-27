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

// TestSecurity_RED_NilReceiverPanic documents that calling methods on a nil
// *apierror.Error panics instead of returning a safe default.
// TODO(security): add nil receiver checks to Error(), ErrorCode(), ErrorMessage()
func TestSecurity_RED_NilReceiverPanic(t *testing.T) {
	var nilErr *apierror.Error

	assert.Panics(t, func() {
		_ = nilErr.Error()
	}, "RED: calling Error() on nil *apierror.Error panics — should return safe default")

	assert.Panics(t, func() {
		_ = nilErr.ErrorCode()
	}, "RED: calling ErrorCode() on nil *apierror.Error panics — should return safe default")

	assert.Panics(t, func() {
		_ = nilErr.ErrorMessage()
	}, "RED: calling ErrorMessage() on nil *apierror.Error panics — should return safe default")

	t.Log("RED: nil *apierror.Error receiver causes panic on all methods — should handle gracefully")
}

// ---- VULNERABILITY 5 --------------------------------------------------------
// JSON serialization of apierror.Error includes the Err field (exported) which
// may contain internal error details. If an apierror.Error is ever marshaled
// to JSON (e.g., in error logging, debug endpoints), the internal cause is
// exposed.
//
// File: pkg/apierror/apierror.go — Error struct has exported Err field with no json tag

// TestSecurity_RED_JSONMarshalExposesInternalError documents that JSON
// marshaling of apierror.Error may expose the Err field's string content.
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

	// The Err field is exported with no json tag — check if it appears in output.
	// Because error interface doesn't have a direct JSON representation and Err is
	// nil here, it marshals as null. But the field IS present in the struct.
	// The real risk is when Err is non-nil and implements json.Marshaler or
	// when the struct is logged via fmt.Sprintf("%+v", e).
	_, hasCode := decoded["Code"]
	_, hasMessage := decoded["Message"]
	_, hasErr := decoded["Err"]

	assert.True(t, hasCode, "RED: Code field is present in JSON output")
	assert.True(t, hasMessage, "RED: Message field is present in JSON output")
	// The Err field appears in JSON output (as null when nil, or as its value when set)
	assert.True(t, hasErr,
		"RED: Err field is present in JSON output — internal errors can leak via JSON serialization")
	t.Log("RED: apierror.Error JSON serialization includes the Err field — internal error details can be exposed")
}
