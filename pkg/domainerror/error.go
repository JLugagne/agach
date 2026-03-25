package domainerror

import "errors"

// Error represents a domain error with a code and message.
type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// ErrorCode returns the domain error code, satisfying controller.CodedError.
func (e *Error) ErrorCode() string {
	return e.Code
}

// ErrorMessage returns the domain error message, satisfying controller.CodedError.
func (e *Error) ErrorMessage() string {
	return e.Message
}

// IsDomainError checks if an error is a domain error.
func IsDomainError(err error) bool {
	var domainErr *Error
	return errors.As(err, &domainErr)
}
