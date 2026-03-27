package security_test

// new_converters_security_test.go — Additional security vulnerabilities found
// in the inbound converters layer.
//
// Each test is a RED test that documents a vulnerability existing in current code.
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

// TestSecurity_RED_NotificationScopePropagatesInvalidValues documents that
// ToPublicNotification passes through any domain.NotificationScope value
// without validation.
//
// TODO(security): Validate Scope against the set of valid NotificationScope
// values in ToPublicNotification, normalising unknown values to a safe default.
func TestSecurity_RED_NotificationScopePropagatesInvalidValues(t *testing.T) {
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

		if !validScopes[result.Scope] && result.Scope != "" {
			t.Logf("RED: ToPublicNotification passed through invalid Scope %q — "+
				"notifications.go:25 casts raw enum value to string without validation",
				result.Scope)
		}

		assert.True(t, validScopes[result.Scope] || result.Scope == "",
			"RED: NotificationScope %q must be normalised to a valid value or empty, got %q",
			scope, result.Scope)
	}
}

// TestSecurity_RED_NotificationSeverityPropagatesInvalidValues documents that
// ToPublicNotification passes through any domain.NotificationSeverity value
// without validation.
//
// TODO(security): Validate Severity against the set of valid
// NotificationSeverity values in ToPublicNotification.
func TestSecurity_RED_NotificationSeverityPropagatesInvalidValues(t *testing.T) {
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

		if !validSeverities[result.Severity] {
			t.Logf("RED: ToPublicNotification passed through invalid Severity %q — "+
				"notifications.go:27 casts raw enum value to string without validation",
				result.Severity)
		}

		assert.True(t, validSeverities[result.Severity],
			"RED: NotificationSeverity %q must be normalised to a valid value, got %q",
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

// TestSecurity_RED_FeatureStatusPropagatesInvalidValues documents that
// ToPublicFeature passes through any domain.FeatureStatus value without
// validation.
//
// TODO(security): Validate Status against the set of valid FeatureStatus
// values in ToPublicFeature.
func TestSecurity_RED_FeatureStatusPropagatesInvalidValues(t *testing.T) {
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

		if !validStatuses[result.Status] {
			t.Logf("RED: ToPublicFeature passed through invalid Status %q — "+
				"features.go:19 casts raw enum value to string without validation",
				result.Status)
		}

		assert.True(t, validStatuses[result.Status],
			"RED: FeatureStatus %q must be normalised to a valid value, got %q",
			status, result.Status)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vulnerability 10: ToDomainProjectID does not validate UUID format
// File: projects.go:9-15
//
// ToDomainProjectID converts *string to *domain.ProjectID by direct cast:
//
//   projectID := domain.ProjectID(*id)
//
// Unlike ToDomainTaskIDs (which now validates UUID format), this function
// accepts any string as a valid project ID. Used in CreateProject
// (projects.go:46) where the ParentID comes from user input.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_ToDomainProjectIDNoValidation documents that
// ToDomainProjectID accepts arbitrary strings without UUID format validation.
//
// TODO(security): Validate the string as a UUID before converting to
// domain.ProjectID, consistent with ToDomainTaskIDs.
func TestSecurity_RED_ToDomainProjectIDNoValidation(t *testing.T) {
	malformedIDs := []string{
		"not-a-uuid",
		"'; DROP TABLE projects; --",
		"../../../etc/passwd",
		"",
		"<script>alert(1)</script>",
	}

	for _, id := range malformedIDs {
		s := id
		result := converters.ToDomainProjectID(&s)

		// The function should either return nil for invalid IDs or panic.
		// Currently it returns any string wrapped as ProjectID.
		if result != nil && string(*result) == id && id != "" {
			t.Logf("RED: ToDomainProjectID accepted malformed ID %q — "+
				"projects.go:13 casts raw string without UUID validation; "+
				"fix: validate with domain.ParseProjectID or uuid.Parse before conversion",
				id)
		}
	}

	// Test that a valid UUID still works (documenting desired behavior)
	validID := domain.NewProjectID().String()
	result := converters.ToDomainProjectID(&validID)
	assert.NotNil(t, result, "Valid UUID should be accepted")
	assert.Equal(t, domain.ProjectID(validID), *result)

	// Test nil input
	nilResult := converters.ToDomainProjectID(nil)
	assert.Nil(t, nilResult, "Nil input should return nil")
}
