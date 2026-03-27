package security_test

// Deep security tests for pkg/controller — vulnerabilities NOT covered by
// the existing controller_security_test.go.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newDeepCtrl() *controller.Controller {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return controller.NewController(l)
}

// ---------------------------------------------------------------------------
// SECURITY: DecodeAndValidate must reject trailing JSON
//
// A secure implementation calls dec.More() after Decode and returns an error
// if there is any remaining data in the body. This prevents HTTP request
// smuggling via JSON and bypassing WAF/signature checks.
//
// File: controller.go:152-154 — dec.Decode(data) reads one value only.
// TODO(security): After Decode, call dec.More() or dec.Decode(&json.RawMessage{})
// and reject the request if there is any remaining data in the body.
// ---------------------------------------------------------------------------

func TestSecurity_RED_DecodeAndValidate_IgnoresTrailingJSON(t *testing.T) {
	c := newDeepCtrl()

	// Two concatenated JSON objects. A secure implementation must reject this.
	body := `{"name":"Alice"}{"name":"Eve","admin":true}`
	r := httptest.NewRequest("POST", "/sec-test", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	type payload struct {
		Name string `json:"name" validate:"required"`
	}
	var p payload
	err := c.DecodeAndValidate(r, &p, nil)

	// SECURE: a request with trailing JSON must be rejected.
	// This test will FAIL until dec.More() is called after Decode.
	assert.Error(t, err,
		"SECURE: DecodeAndValidate must reject a body containing trailing JSON objects — "+
			"dec.More() should be called after Decode and return an error if extra data is present")
}

// ---------------------------------------------------------------------------
// SECURITY: SendFail status code must be bounded to 4xx range
//
// SendFail accepts an arbitrary *int status code. A caller can pass 200, 301,
// or 500 as a status code to SendFail, producing a misleading HTTP response.
//
// File: controller.go:84-88 — statusCode is used without range validation.
// TODO(security): Validate that statusCode is in the 4xx range (400-499),
// or fall back to 400 if outside that range.
// ---------------------------------------------------------------------------

func TestSecurity_RED_SendFail_AcceptsArbitraryStatusCode(t *testing.T) {
	c := newDeepCtrl()

	tests := []struct {
		name       string
		statusCode int
		desc       string
	}{
		{"200 OK", http.StatusOK, "error disguised as success"},
		{"301 redirect", http.StatusMovedPermanently, "error disguised as redirect"},
		{"500 server error", http.StatusInternalServerError, "client error disguised as server error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/sec-test", nil)

			code := tc.statusCode
			c.SendFail(w, r, &code, &testCodedError{code: "TEST", msg: "test error"})

			// SECURE: SendFail must not use non-4xx status codes.
			// Invalid codes should be rejected or clamped to a valid 4xx range.
			// This test will FAIL until status code validation is added.
			assert.NotEqual(t, tc.statusCode, w.Code,
				"SECURE: SendFail must not write non-4xx status code %d (%s) — "+
					"the code should be clamped or rejected to prevent clients/monitors "+
					"from misinterpreting the response", tc.statusCode, tc.desc)
			assert.GreaterOrEqual(t, w.Code, 400,
				"SECURE: SendFail must write a 4xx status code, got %d", w.Code)
			assert.Less(t, w.Code, 500,
				"SECURE: SendFail must write a 4xx status code, got %d", w.Code)
		})
	}
}

// testCodedError satisfies controller.CodedError for testing.
type testCodedError struct {
	code string
	msg  string
}

func (e *testCodedError) Error() string       { return e.msg }
func (e *testCodedError) ErrorCode() string    { return e.code }
func (e *testCodedError) ErrorMessage() string { return e.msg }

// ---------------------------------------------------------------------------
// SECURITY: SendSuccess must not return 200 when data cannot be marshaled
//
// SendSuccess encodes whatever data is passed into the JSON response. If the
// data contains a channel, func, or complex type, json.Encoder returns an error
// after the 200 header has already been written, leaving the client with a
// 200 OK response and a corrupt/empty body.
//
// A secure implementation would pre-marshal the data before writing the header,
// and return a 500 error if marshaling fails.
//
// File: controller.go:67 — json.NewEncoder(w).Encode(Response{Data: data})
// TODO(security): Pre-marshal data or wrap Encode in a buffer; return 500 on failure.
// ---------------------------------------------------------------------------

func TestSecurity_RED_SendSuccess_PanicsOnUnmarshalableData(t *testing.T) {
	c := newDeepCtrl()

	// A channel cannot be JSON-marshaled; json.Encoder.Encode returns an error.
	type leaky struct {
		Name string     `json:"name"`
		Ch   chan string `json:"ch"`
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/sec-test", nil)

	var panicked bool
	func() {
		defer func() {
			if rv := recover(); rv != nil {
				panicked = true
			}
		}()
		c.SendSuccess(w, r, leaky{Name: "test", Ch: make(chan string)})
	}()

	if panicked {
		// If it panicked, that's a separate bug — there should be recovery.
		t.Log("SendSuccess panicked on unmarshalable data — no panic recovery in the response path")
	}

	// SECURE: when data cannot be marshaled, the response must NOT be 200 OK.
	// A correct implementation should detect the marshal failure before writing
	// headers, or write a 500 status when the error is detected.
	// This test will FAIL until SendSuccess handles marshal errors correctly.
	assert.NotEqual(t, http.StatusOK, w.Code,
		"SECURE: SendSuccess must not return 200 OK when the response body cannot be marshaled — "+
			"the status should be 500 to prevent clients from treating a corrupt response as success")

	// Additionally, the response body must be valid JSON (not corrupt/empty).
	var resp controller.Response
	decErr := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, decErr,
		"SECURE: response body must be valid JSON even when data marshaling fails — "+
			"client must receive a well-formed error response")
}

// ---------------------------------------------------------------------------
// SECURITY: DecodeAndValidate with DisallowUnknownFields must not leak field names
//
// When an unknown field is sent, the error from json.Decoder includes the exact
// field name: `json: unknown field "secret_field"`. This error, if propagated
// to the client, allows attackers to enumerate valid fields by probing the API.
//
// File: controller.go:153-156 — error from Decode leaks field name.
// TODO(security): Catch unknown field errors and return a generic
// "invalid request body" message without the field name.
// ---------------------------------------------------------------------------

func TestSecurity_RED_DecodeAndValidate_LeaksFieldNamesOnUnknownField(t *testing.T) {
	c := newDeepCtrl()

	body := `{"name":"Alice","secret_internal_field":"probe"}`
	r := httptest.NewRequest("POST", "/sec-test", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	type payload struct {
		Name string `json:"name" validate:"required"`
	}
	var p payload
	err := c.DecodeAndValidate(r, &p, nil)

	require.Error(t, err, "unknown field should cause an error")

	// SECURE: the error message must NOT expose the unknown field name.
	// An attacker could enumerate valid fields by probing with different names
	// and observing whether the field name appears in the error response.
	// This test will FAIL until DecodeAndValidate sanitizes the error message.
	assert.NotContains(t, err.Error(), "secret_internal_field",
		"SECURE: error message must not expose unknown field names — "+
			"return a generic 'invalid request body' message instead to prevent field enumeration attacks")
}
