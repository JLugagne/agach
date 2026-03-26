package domain

import "github.com/JLugagne/agach-mcp/pkg/domainerror"

// Error is the domain error type, shared via domainerror.
type Error = domainerror.Error

// IsDomainError checks if an error is a domain error.
func IsDomainError(err error) bool {
	return domainerror.IsDomainError(err)
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

	ErrNodeNotFound          = &Error{Code: "NODE_NOT_FOUND", Message: "node not found"}
	ErrNodeRevoked           = &Error{Code: "NODE_REVOKED", Message: "node has been revoked"}
	ErrOnboardingCodeNotFound = &Error{Code: "ONBOARDING_CODE_NOT_FOUND", Message: "onboarding code not found"}
	ErrOnboardingCodeExpired  = &Error{Code: "ONBOARDING_CODE_EXPIRED", Message: "onboarding code has expired"}
	ErrOnboardingCodeUsed     = &Error{Code: "ONBOARDING_CODE_USED", Message: "onboarding code has already been used"}
	ErrUserBlocked            = &Error{Code: "USER_BLOCKED", Message: "user account is blocked"}
)
