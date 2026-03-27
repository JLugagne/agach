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

// TestSecurity_RED_NilPointerPanicOnError documents that calling Error()
// on a nil *apierror.Error panics instead of returning a safe fallback.
// TODO(security): add nil receiver checks to Error(), ErrorCode(), ErrorMessage()
func TestSecurity_RED_NilPointerPanicOnError(t *testing.T) {
	var e *apierror.Error

	// Calling Error() directly on a nil pointer panics.
	assert.Panics(t, func() {
		_ = e.Error()
	}, "RED: calling Error() on nil *apierror.Error panics — must return safe fallback")
	t.Log("RED: nil *apierror.Error causes panic on Error() call — add nil receiver guard")
}

// TestSecurity_RED_NilPointerPanicOnErrorCode documents that calling
// ErrorCode() on a nil *apierror.Error panics.
// TODO(security): add nil receiver check to ErrorCode()
func TestSecurity_RED_NilPointerPanicOnErrorCode(t *testing.T) {
	var e *apierror.Error

	assert.Panics(t, func() {
		_ = e.ErrorCode()
	}, "RED: calling ErrorCode() on nil *apierror.Error panics — must return safe fallback")
	t.Log("RED: nil *apierror.Error causes panic on ErrorCode() call")
}

// TestSecurity_RED_NilPointerPanicOnErrorMessage documents that calling
// ErrorMessage() on a nil *apierror.Error panics.
// TODO(security): add nil receiver check to ErrorMessage()
func TestSecurity_RED_NilPointerPanicOnErrorMessage(t *testing.T) {
	var e *apierror.Error

	assert.Panics(t, func() {
		_ = e.ErrorMessage()
	}, "RED: calling ErrorMessage() on nil *apierror.Error panics — must return safe fallback")
	t.Log("RED: nil *apierror.Error causes panic on ErrorMessage() call")
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

// TestSecurity_RED_JSONMarshalingNoControlledOutput documents that
// apierror.Error has no json tags to control serialization.
// TODO(security): add json tags (especially json:"-" on Err) or implement MarshalJSON
func TestSecurity_RED_JSONMarshalingNoControlledOutput(t *testing.T) {
	e := &apierror.Error{
		Code:    "INTERNAL_ERROR",
		Message: "Something went wrong",
		Err:     errors.New("pq: password authentication failed for user \"admin\""),
	}

	data, err := json.Marshal(e)
	assert.NoError(t, err, "marshaling should not fail")

	jsonStr := string(data)

	// Verify the Err field appears in the JSON output (as an empty object,
	// since *errors.errorString has no exported fields).
	// The vulnerability is that there's no json:"-" tag to suppress it,
	// and a custom error type with exported fields would leak data.
	assert.Contains(t, jsonStr, "Err",
		"RED: apierror.Error Err field appears in JSON output — add json:\"-\" tag")
	t.Log("RED: apierror.Error has no json tags — Err field appears in JSON output as uncontrolled key")
}
