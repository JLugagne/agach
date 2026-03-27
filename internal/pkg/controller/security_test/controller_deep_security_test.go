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
// VULNERABILITY: DecodeAndValidate silently ignores trailing JSON
//
// json.NewDecoder reads exactly one JSON value and stops. If the request body
// contains multiple concatenated JSON objects (e.g. `{"a":1}{"b":2}`), only
// the first is decoded and validated. The trailing data is silently ignored.
//
// An attacker can append extra JSON objects that might be consumed by a
// downstream body reader (HTTP request smuggling via JSON) or use the trailing
// data to bypass WAF/signature checks that inspect the full body.
//
// File: controller.go:152-154 — dec.Decode(data) reads one value only.
// TODO(security): After Decode, call dec.More() or dec.Decode(&json.RawMessage{})
// and reject the request if there is any remaining data in the body.
// ---------------------------------------------------------------------------

func TestSecurity_RED_DecodeAndValidate_IgnoresTrailingJSON(t *testing.T) {
	c := newDeepCtrl()

	// Two concatenated JSON objects. The second is silently ignored.
	// json.Decoder internally buffers the stream, so ReadAll after Decode
	// may return empty. The vulnerability is that Decode succeeds without
	// checking whether the body contained exactly one JSON value.
	body := `{"name":"Alice"}{"name":"Eve","admin":true}`
	r := httptest.NewRequest("POST", "/sec-test", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	type payload struct {
		Name string `json:"name" validate:"required"`
	}
	var p payload
	err := c.DecodeAndValidate(r, &p, nil)

	// RED: only the first object is decoded; the second is silently ignored.
	// A secure implementation would call dec.More() after Decode and reject
	// the request if there is trailing data.
	assert.NoError(t, err,
		"RED: DecodeAndValidate accepts a body with trailing JSON objects without error")
	assert.Equal(t, "Alice", p.Name,
		"RED: only the first JSON object was decoded — the second (potentially malicious) one was silently dropped")
	t.Log("RED: DecodeAndValidate silently ignores trailing JSON in the request body — " +
		"dec.More() is never called to reject extra data")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: SendFail status code is not bounded to 4xx range
//
// SendFail accepts an arbitrary *int status code. A caller can pass 200, 301,
// or 500 as a status code to SendFail, producing a misleading HTTP response.
// In particular, passing a 2xx code means the client treats an error as a
// success, and passing a 5xx code triggers incorrect monitoring alerts.
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

			// RED: SendFail uses whatever code is given, even when it makes no sense
			// for a "fail" response.
			assert.Equal(t, tc.statusCode, w.Code,
				"RED: SendFail accepted non-4xx status code %d (%s) — "+
					"clients/monitors will misinterpret the response", tc.statusCode, tc.desc)
		})
	}

	t.Log("RED: SendFail does not validate that the status code is in the 4xx range")
}

// testCodedError satisfies controller.CodedError for testing.
type testCodedError struct {
	code string
	msg  string
}

func (e *testCodedError) Error() string        { return e.msg }
func (e *testCodedError) ErrorCode() string     { return e.code }
func (e *testCodedError) ErrorMessage() string  { return e.msg }

// ---------------------------------------------------------------------------
// VULNERABILITY: SendSuccess serialises arbitrary interface{} data
//
// SendSuccess encodes whatever data is passed into the JSON response. If a
// handler accidentally passes a struct containing unexported-but-marshalable
// fields, database connection objects, or types with custom MarshalJSON that
// panic, the response can leak internal state or crash the server.
//
// More concretely: json.Encoder does not return an error until Write is called,
// and if the data contains a channel, func, or complex type, Encode panics
// (via json.Marshal) with an "unsupported type" error. The caller has no
// protection against this.
//
// File: controller.go:67 — json.NewEncoder(w).Encode(Response{Data: data})
// TODO(security): Wrap the Encode call in a recover() to prevent panics from
// crashing the entire server, and return a generic 500 error instead.
// ---------------------------------------------------------------------------

func TestSecurity_RED_SendSuccess_PanicsOnUnmarshalableData(t *testing.T) {
	c := newDeepCtrl()

	// A channel cannot be JSON-marshaled; json.Encoder will panic.
	type leaky struct {
		Name string      `json:"name"`
		Ch   chan string  `json:"ch"`
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/sec-test", nil)

	// The panic from json.Marshal on a chan type propagates up.
	// A production server without panic recovery will crash.
	var panicked bool
	func() {
		defer func() {
			if rv := recover(); rv != nil {
				panicked = true
			}
		}()
		c.SendSuccess(w, r, leaky{Name: "test", Ch: make(chan string)})
	}()

	// RED: this currently does not panic because json.Encoder.Encode returns
	// an error for unsupported types rather than panicking. However, the error
	// is logged but the response may be partially written (header sent,
	// body truncated), leaving the client with a corrupt response.
	if panicked {
		t.Log("RED: SendSuccess panicked on unmarshalable data — " +
			"no panic recovery in the response path")
	} else {
		// Even without panic, the response is malformed: the header was sent
		// as 200 OK but the body is incomplete/corrupt JSON.
		assert.Equal(t, http.StatusOK, w.Code,
			"RED: status 200 was already written before the encode error was detected")

		var resp controller.Response
		decErr := json.NewDecoder(w.Body).Decode(&resp)
		assert.Error(t, decErr,
			"RED: response body is corrupt/empty after encode failure — "+
				"client receives 200 OK with invalid JSON")
		t.Log("RED: SendSuccess returns 200 with corrupt body when data cannot be marshaled")
	}
}

// ---------------------------------------------------------------------------
// VULNERABILITY: DecodeAndValidate with DisallowUnknownFields leaks field names
//
// When DisallowUnknownFields is enabled (controller.go:153) and the request
// contains an unknown field, the error message from json.Decoder includes the
// exact unknown field name: `json: unknown field "secret_field"`. This error
// is returned to the caller and, if passed to SendFail, may leak the names
// of valid fields to an attacker probing the API schema.
//
// File: controller.go:153-156 — error from Decode leaks field name.
// TODO(security): Catch json.UnmarshalTypeError / unknown field errors and
// return a generic "invalid request body" message without the field name.
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
	assert.Contains(t, err.Error(), "secret_internal_field",
		"RED: error message exposes the unknown field name — "+
			"an attacker can enumerate valid fields by probing with different names")
	t.Log("RED: DecodeAndValidate leaks unknown field names in error messages")
}
