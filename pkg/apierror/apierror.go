package apierror

// Error is a coded application error surfaced to HTTP/MCP consumers.
// It is defined in pkg so that both pkg/* and internal/*/inbound layers
// can reference it without creating an import cycle.
// internal/server/domain defines its own domain.Error independently;
// the inbound layer is responsible for converting between the two using errors.Is/As.
type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "an error occurred"
}

func (e *Error) Unwrap() error {
	return nil
}
