package security_test

// Security tests for pkg/apierror — nil pointer and JSON serialization gaps.
//
// These RED tests document vulnerabilities NOT covered by the existing
// apierror_security_test.go file.

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/stretchr/testify/assert"
)

// ─── VULNERABILITY: nil receiver causes panic ───────────────────────────────
// Error(), ErrorCode(), and ErrorMessage() are pointer receiver methods on
// *Error with no nil guard. Calling them on a nil *apierror.Error panics
// (nil pointer dereference on e.Message). This can happen when a function
// returns *apierror.Error and the caller invokes methods on it without a
// nil check.
//
// File: pkg/apierror/apierror.go lines 14-36

// TestSecurity_RED_NilPointerPanicOnError asserts that calling Error()
// on a nil *apierror.Error does NOT panic — it should return a safe fallback.
// This test currently FAILS because the production code has no nil receiver guard.
// TODO(security): add nil receiver checks to Error(), ErrorCode(), ErrorMessage()
func TestSecurity_RED_NilPointerPanicOnError(t *testing.T) {
	var e *apierror.Error

	// Correct behavior: calling Error() on a nil pointer should NOT panic.
	assert.NotPanics(t, func() {
		_ = e.Error()
	}, "RED: calling Error() on nil *apierror.Error must not panic — should return safe fallback")
}

// TestSecurity_RED_NilPointerPanicOnErrorCode asserts that calling
// ErrorCode() on a nil *apierror.Error does NOT panic.
// This test currently FAILS because the production code has no nil receiver guard.
// TODO(security): add nil receiver check to ErrorCode()
func TestSecurity_RED_NilPointerPanicOnErrorCode(t *testing.T) {
	var e *apierror.Error

	assert.NotPanics(t, func() {
		_ = e.ErrorCode()
	}, "RED: calling ErrorCode() on nil *apierror.Error must not panic — should return safe fallback")
}

// TestSecurity_RED_NilPointerPanicOnErrorMessage asserts that calling
// ErrorMessage() on a nil *apierror.Error does NOT panic.
// This test currently FAILS because the production code has no nil receiver guard.
// TODO(security): add nil receiver check to ErrorMessage()
func TestSecurity_RED_NilPointerPanicOnErrorMessage(t *testing.T) {
	var e *apierror.Error

	assert.NotPanics(t, func() {
		_ = e.ErrorMessage()
	}, "RED: calling ErrorMessage() on nil *apierror.Error must not panic — should return safe fallback")
}

// ─── VULNERABILITY: JSON marshaling does not control output shape ────────────
// apierror.Error has no json tags and no custom MarshalJSON. When the struct
// is serialized (e.g., for logging or API responses outside the controller),
// the output shape is uncontrolled. The exported Err field (type error) will
// serialize as an empty object {} for most error types (since *errors.errorString
// has no exported fields), but custom error types with exported fields could
// leak internal data.
//
// File: pkg/apierror/apierror.go lines 8-12

// TestSecurity_RED_JSONMarshalingNoControlledOutput asserts that
// apierror.Error's Err field does NOT appear in JSON output.
// This test currently FAILS because there is no json:"-" tag suppressing it.
// TODO(security): add json tags (especially json:"-" on Err) or implement MarshalJSON
func TestSecurity_RED_JSONMarshalingNoControlledOutput(t *testing.T) {
	e := &apierror.Error{
		Code:    "INTERNAL_ERROR",
		Message: "Something went wrong",
		Err:     apierror.WrapErr(errors.New("pq: password authentication failed for user \"admin\"")),
	}

	data, err := json.Marshal(e)
	assert.NoError(t, err, "marshaling should not fail")

	jsonStr := string(data)

	// Correct behavior: the Err field must NOT appear in the JSON output.
	// It should be suppressed via json:"-" or a custom MarshalJSON to prevent
	// internal error details from leaking through serialization.
	assert.NotContains(t, jsonStr, "Err",
		"RED: apierror.Error Err field must not appear in JSON output — add json:\"-\" tag")
}
