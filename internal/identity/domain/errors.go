package domain

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

// IsDomainError checks if an error is a domain error.
func IsDomainError(err error) bool {
	var domainErr *Error
	return errors.As(err, &domainErr)
}

var (
	ErrUnauthorized = &Error{
		Code:    "UNAUTHORIZED",
		Message: "authentication required",
	}
	ErrForbidden = &Error{
		Code:    "FORBIDDEN",
		Message: "access denied",
	}
	ErrUserNotFound = &Error{
		Code:    "USER_NOT_FOUND",
		Message: "user not found",
	}
	ErrAPIKeyNotFound = &Error{
		Code:    "API_KEY_NOT_FOUND",
		Message: "api key not found",
	}
	ErrAPIKeyInvalid = &Error{
		Code:    "API_KEY_INVALID",
		Message: "invalid api key",
	}
	ErrAPIKeyExpired = &Error{
		Code:    "API_KEY_EXPIRED",
		Message: "api key expired",
	}
	ErrAPIKeyRevoked = &Error{
		Code:    "API_KEY_REVOKED",
		Message: "api key revoked",
	}
	ErrInvalidCredentials = &Error{
		Code:    "INVALID_CREDENTIALS",
		Message: "invalid email or password",
	}
	ErrEmailAlreadyExists = &Error{
		Code:    "EMAIL_ALREADY_EXISTS",
		Message: "email already registered",
	}
	ErrSSOUserNoPassword = &Error{
		Code:    "SSO_USER_NO_PASSWORD",
		Message: "user registered via SSO, use SSO login",
	}
	ErrTeamNotFound = &Error{
		Code:    "TEAM_NOT_FOUND",
		Message: "team not found",
	}
	ErrTeamSlugConflict = &Error{
		Code:    "TEAM_SLUG_CONFLICT",
		Message: "team slug already in use",
	}
	ErrSSOProviderNotFound = &Error{Code: "SSO_PROVIDER_NOT_FOUND", Message: "SSO provider not configured"}
	ErrSSONotSupported     = &Error{Code: "SSO_NOT_SUPPORTED", Message: "SAML not yet supported"}
)
