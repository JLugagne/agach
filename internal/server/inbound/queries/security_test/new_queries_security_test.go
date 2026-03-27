package security_test

// new_queries_security_test.go — Additional security vulnerabilities found in
// the inbound queries layer.
//
// Each test is a RED test that documents a vulnerability existing in current code.
//
// Run with: go test -race -failfast ./internal/server/inbound/queries/security_test/...

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/queries"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func newNotificationRouter(mock *servicetest.MockQueries) *mux.Router {
	ctrl := newTestController()
	router := mux.NewRouter()
	queries.NewNotificationQueriesHandler(mock, ctrl).RegisterRoutes(router)
	return router
}

func newFeatureRouter(mock *servicetest.MockQueries) *mux.Router {
	ctrl := newTestController()
	router := mux.NewRouter()
	queries.NewFeatureQueriesHandler(mock, ctrl).RegisterRoutes(router)
	return router
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-9: ListNotifications — no upper bound on ?limit parameter
// File: notifications.go:43-46
//
// The parseNotificationQueryParams function parses the ?limit= query parameter
// with no maximum cap. A caller can request ?limit=10000000 to force the
// service layer to return an arbitrarily large result set, exhausting memory
// and database resources (DoS).
//
// Unlike SearchTasks (which caps to maxSearchLimit=1000) and ListComments
// (which caps to maxCommentLimit=500), notifications have no upper bound.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_NotificationLimitUnbounded documents that the notification
// list endpoint accepts arbitrarily large ?limit= values without capping.
//
// TODO(security): Add a maxNotificationLimit constant and cap the parsed limit
// in parseNotificationQueryParams (notifications.go:43-46).
func TestSecurity_RED_NotificationLimitUnbounded(t *testing.T) {
	const maxReasonableLimit = 1000

	receivedLimit := 0
	mock := &servicetest.MockQueries{
		ListNotificationsFunc: func(_ context.Context, _ *domain.ProjectID, _ *domain.NotificationScope, _ string, _ bool, limit, _ int) ([]domain.Notification, error) {
			receivedLimit = limit
			return nil, nil
		},
	}

	router := newNotificationRouter(mock)
	req := httptest.NewRequest(http.MethodGet,
		"/api/notifications?limit=5000000",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	if receivedLimit > maxReasonableLimit {
		t.Logf("RED: ListNotifications received limit=%d — no maximum cap is enforced; "+
			"fix: add maxNotificationLimit constant and cap in parseNotificationQueryParams "+
			"(notifications.go:43-46)", receivedLimit)
	}

	assert.LessOrEqual(t, receivedLimit, maxReasonableLimit,
		"RED: notification list limit must be capped to prevent DoS")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-10: ListFeatures — unvalidated FeatureStatus filter values
// File: features.go:41-48
//
// The handler splits the ?status= query param by comma and casts each segment
// directly to domain.FeatureStatus without validation:
//
//   statusFilter = append(statusFilter, domain.FeatureStatus(s))
//
// Valid statuses are: draft, ready, in_progress, done, blocked.
// Any arbitrary string (including SQL fragments or empty strings) is passed
// through to the query layer. While SQL injection depends on the repository
// implementation, passing untrusted enum values violates defense-in-depth.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_FeatureStatusFilterUnvalidated documents that the feature
// listing endpoint passes arbitrary status filter values to the service layer
// without enum validation.
//
// TODO(security): Validate each status value against the set of valid
// FeatureStatus values before passing to the service layer.
func TestSecurity_RED_FeatureStatusFilterUnvalidated(t *testing.T) {
	projectID := newValidProjectID()

	var receivedStatuses []domain.FeatureStatus
	mock := &servicetest.MockQueries{
		ListFeaturesFunc: func(_ context.Context, _ domain.ProjectID, statuses []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
			receivedStatuses = statuses
			return nil, nil
		},
	}

	router := newFeatureRouter(mock)

	// Send an invalid status value that includes bogus entries
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(projectID)+"/features?status=draft,nonexistent,invalid_status",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	validStatuses := map[domain.FeatureStatus]bool{
		"draft": true, "ready": true, "in_progress": true, "done": true, "blocked": true,
	}

	hasInvalid := false
	for _, s := range receivedStatuses {
		if !validStatuses[s] {
			hasInvalid = true
			t.Logf("RED: ListFeatures received invalid status filter %q — "+
				"features.go:43-46 passes raw user input as FeatureStatus enum; "+
				"fix: validate against known status values before passing to service layer", s)
		}
	}

	assert.False(t, hasInvalid,
		"RED: feature status filter must only contain valid FeatureStatus enum values")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-11: GetFeature — ignores projectID from URL, IDOR across projects
// File: features.go:66-80
//
// The handler extracts featureId from the URL but ignores the {id} (projectID)
// path parameter. Any caller can retrieve a feature by its UUID regardless of
// the project specified in the URL path. This enables cross-project data
// exfiltration if an attacker guesses/knows a feature UUID from another project.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_GetFeatureIgnoresProjectID documents that the GetFeature
// query endpoint does not validate that the feature belongs to the stated
// project.
//
// TODO(security): Pass projectID to GetFeature and verify the feature belongs
// to the stated project, or add a cross-check in the handler.
func TestSecurity_RED_GetFeatureIgnoresProjectID(t *testing.T) {
	foreignProjectID := newValidProjectID()
	featureID := domain.NewFeatureID()

	featureFetched := false
	mock := &servicetest.MockQueries{
		GetFeatureFunc: func(_ context.Context, fID domain.FeatureID) (*domain.Feature, error) {
			featureFetched = true
			// Return a feature that belongs to a DIFFERENT project
			otherProjectID := domain.NewProjectID()
			return &domain.Feature{
				ID:        fID,
				ProjectID: otherProjectID,
				Name:      "Secret Feature",
			}, nil
		},
	}

	router := newFeatureRouter(mock)

	// Request the feature via a different project's URL
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(foreignProjectID)+"/features/"+featureID.String(),
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if featureFetched && rr.Code == http.StatusOK {
		t.Log("RED: GetFeature returned a feature that belongs to a different project — " +
			"features.go:67 ignores the {id} parameter; " +
			"an attacker can read any feature by UUID via any project URL")
	}

	// The handler should either return 404 when the feature doesn't belong to
	// the stated project, or pass projectID to the query for scoped retrieval.
	assert.NotEqual(t, http.StatusOK, rr.Code,
		"RED: GetFeature must not return 200 when the feature belongs to a different project")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-12: Notification scope filter accepts arbitrary values
// File: notifications.go:37-39
//
// parseNotificationQueryParams casts the raw ?scope= value directly to
// domain.NotificationScope without validation:
//
//   sc := domain.NotificationScope(s)
//
// Valid scopes are: project, agent, global.
// Arbitrary values are passed through to the query layer.
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_NotificationScopeFilterUnvalidated documents that the
// notification listing endpoint passes arbitrary scope values without
// validation.
//
// TODO(security): Validate the scope parameter against known
// NotificationScope enum values.
func TestSecurity_RED_NotificationScopeFilterUnvalidated(t *testing.T) {
	var receivedScope *domain.NotificationScope
	mock := &servicetest.MockQueries{
		ListNotificationsFunc: func(_ context.Context, _ *domain.ProjectID, scope *domain.NotificationScope, _ string, _ bool, _, _ int) ([]domain.Notification, error) {
			receivedScope = scope
			return nil, nil
		},
	}

	router := newNotificationRouter(mock)

	req := httptest.NewRequest(http.MethodGet,
		"/api/notifications?scope=nonexistent_scope",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	validScopes := map[domain.NotificationScope]bool{
		"project": true, "agent": true, "global": true,
	}

	if receivedScope != nil && !validScopes[*receivedScope] {
		t.Logf("RED: ListNotifications received invalid scope %q — "+
			"notifications.go:37-39 casts raw query param to NotificationScope; "+
			"fix: validate against known scope values before passing to service layer",
			*receivedScope)
	}

	if receivedScope != nil {
		assert.True(t, validScopes[*receivedScope],
			"RED: notification scope filter must only contain valid NotificationScope enum values")
	}
}
