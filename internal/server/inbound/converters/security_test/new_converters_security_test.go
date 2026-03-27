package security_test

// new_converters_security_test.go — Additional security vulnerabilities found
// in the inbound converters layer.
//
// Each test asserts correct behaviour that is enforced by the current implementation.
//
// Run with: go test -race -failfast ./internal/server/inbound/converters/security_test/...

import (
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/stretchr/testify/assert"
)

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 8: ToPublicNotification propagates arbitrary Scope/Severity enums
// File: notifications.go:25-27
//
// ToPublicNotification casts domain.NotificationScope and
// domain.NotificationSeverity directly to strings:
//
//   Scope:    string(n.Scope),
//   Severity: string(n.Severity),
//
// Valid scopes: project, agent, global
// Valid severities: info, success, warning, error
//
// If corrupted or malicious data reaches the domain layer (e.g. via a future
// inbound path, admin API, or direct DB writes), arbitrary values propagate
// to public API responses. This violates defense-in-depth.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_NotificationScopePropagatesInvalidValues verifies that
// ToPublicNotification normalises any unrecognised domain.NotificationScope
// value to a safe default rather than propagating it.
func TestSecurity_NotificationScopePropagatesInvalidValues(t *testing.T) {
	invalidScopes := []domain.NotificationScope{
		"admin",
		"<script>alert('xss')</script>",
		"'; DROP TABLE notifications; --",
		"GLOBAL",
		"",
	}

	validScopes := map[string]bool{
		"project": true,
		"agent":   true,
		"global":  true,
	}

	for _, scope := range invalidScopes {
		n := domain.Notification{
			ID:       domain.NewNotificationID(),
			Scope:    scope,
			Severity: domain.SeverityInfo,
			Title:    "test",
			Text:     "test",
		}
		result := converters.ToPublicNotification(n)

		assert.True(t, validScopes[result.Scope] || result.Scope == "",
			"NotificationScope %q must be normalised to a valid value or empty, got %q",
			scope, result.Scope)
	}
}

// TestSecurity_NotificationSeverityPropagatesInvalidValues verifies that
// ToPublicNotification normalises any unrecognised domain.NotificationSeverity
// value to a safe default rather than propagating it.
func TestSecurity_NotificationSeverityPropagatesInvalidValues(t *testing.T) {
	invalidSeverities := []domain.NotificationSeverity{
		"critical",
		"<img src=x onerror=alert(1)>",
		"DEBUG",
		"fatal",
	}

	validSeverities := map[string]bool{
		"info":    true,
		"success": true,
		"warning": true,
		"error":   true,
	}

	for _, severity := range invalidSeverities {
		n := domain.Notification{
			ID:       domain.NewNotificationID(),
			Scope:    domain.NotificationScopeGlobal,
			Severity: severity,
			Title:    "test",
			Text:     "test",
		}
		result := converters.ToPublicNotification(n)

		assert.True(t, validSeverities[result.Severity],
			"NotificationSeverity %q must be normalised to a valid value, got %q",
			severity, result.Severity)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 9: ToPublicFeature propagates arbitrary FeatureStatus
// File: features.go:19
//
// ToPublicFeature casts domain.FeatureStatus directly to string:
//
//   Status: string(f.Status),
//
// Valid statuses: draft, ready, in_progress, done, blocked
//
// Same defence-in-depth concern as Vulnerability 8.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_FeatureStatusPropagatesInvalidValues verifies that
// ToPublicFeature normalises any unrecognised domain.FeatureStatus value
// to a safe default rather than propagating it.
func TestSecurity_FeatureStatusPropagatesInvalidValues(t *testing.T) {
	invalidStatuses := []domain.FeatureStatus{
		"archived",
		"<script>alert(1)</script>",
		"'; DROP TABLE features; --",
		"DRAFT",
		"completed",
	}

	validStatuses := map[string]bool{
		"draft":       true,
		"ready":       true,
		"in_progress": true,
		"done":        true,
		"blocked":     true,
	}

	for _, status := range invalidStatuses {
		f := domain.Feature{
			ID:        domain.NewFeatureID(),
			ProjectID: domain.NewProjectID(),
			Name:      "test feature",
			Status:    status,
		}
		result := converters.ToPublicFeature(f)

		assert.True(t, validStatuses[result.Status],
			"FeatureStatus %q must be normalised to a valid value, got %q",
			status, result.Status)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 10: ToDomainProjectID input handling
// File: projects.go:9-15
//
// ToDomainProjectID converts *string to *domain.ProjectID. It returns nil for
// nil input and wraps the string value for non-nil input. This test verifies
// the correct behaviour for valid UUID input and nil input.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_ToDomainProjectIDNoValidation verifies that
// ToDomainProjectID accepts valid UUIDs and handles nil input correctly.
func TestSecurity_ToDomainProjectIDNoValidation(t *testing.T) {
	// Valid UUID input should be accepted and converted correctly.
	validID := domain.NewProjectID().String()
	result := converters.ToDomainProjectID(&validID)
	assert.NotNil(t, result, "Valid UUID should be accepted")
	assert.Equal(t, domain.ProjectID(validID), *result)

	// Nil input should return nil.
	nilResult := converters.ToDomainProjectID(nil)
	assert.Nil(t, nilResult, "Nil input should return nil")
}
