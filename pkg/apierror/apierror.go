package apierror

// WrappedErr wraps an internal error so it cannot be directly compared
// with the original via == (preventing external callers from extracting
// the raw internal error by equality check).
type WrappedErr struct {
	inner error
}

// Error implements the error interface.
func (w *WrappedErr) Error() string {
	if w == nil || w.inner == nil {
		return ""
	}
	return w.inner.Error()
}

// Error is a coded application error surfaced to HTTP/MCP consumers.
// It is defined in pkg so that both pkg/* and internal/*/inbound layers
// can reference it without creating an import cycle.
// internal/server/domain defines its own domain.Error independently;
// the inbound layer is responsible for converting between the two using errors.Is/As.
type Error struct {
	Code    string
	Message string
	Err     *WrappedErr `json:"-"`
}

// WrapErr creates a WrappedErr from a raw error.
func WrapErr(err error) *WrappedErr {
	if err == nil {
		return nil
	}
	return &WrappedErr{inner: err}
}

func (e *Error) Error() string {
	if e == nil {
		return "an error occurred"
	}
	if e.Message != "" {
		return e.Message
	}
	return "an error occurred"
}

func (e *Error) Unwrap() error {
	return nil
}

// ErrorCode returns the error code, satisfying controller.CodedError.
func (e *Error) ErrorCode() string {
	if e == nil {
		return ""
	}
	return e.Code
}

// ErrorMessage returns the error message, satisfying controller.CodedError.
func (e *Error) ErrorMessage() string {
	if e == nil {
		return "an error occurred"
	}
	if e.Message != "" {
		return e.Message
	}
	return "an error occurred"
}
