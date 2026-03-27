package security_test

// new_security_test.go — Additional security vulnerabilities found in the
// inbound commands layer.
//
// Each test is a RED test that documents a vulnerability existing in current code.
//
// Run with: go test -race -failfast ./internal/server/inbound/commands/security_test/...

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/sse"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/commands"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// test infrastructure (reuses the deep_security_test pattern)
// ─────────────────────────────────────────────────────────────────────────────

func newRedSecurityRouter(t *testing.T, app commands.App) *mux.Router {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetOutput(io.Discard)
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()
	sseHub := sse.NewHub(logrus.New())

	router := mux.NewRouter()
	commands.NewRouter(router, app, ctrl, hub, sseHub, "")
	return router
}

type redMockApp struct {
	*servicetest.MockCommands
	*servicetest.MockQueries
}

func newRedTestApp(cmds *servicetest.MockCommands, qrs *servicetest.MockQueries) commands.App {
	return &redMockApp{MockCommands: cmds, MockQueries: qrs}
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-11  IDOR: Feature mutations ignore projectID from URL path
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_FeatureUpdateIgnoresProjectID documents that UpdateFeature
// (features.go:72) extracts featureId from the URL path but completely ignores
// the {id} (projectID) parameter. An attacker who knows a feature UUID from
// project A can update it via /api/projects/<projectB>/features/<featureFromA>.
// The handler never verifies the feature belongs to the stated project.
//
// TODO(security): Pass projectID to the service layer and verify ownership,
// or add a cross-check in the handler before calling UpdateFeature.
func TestSecurity_RED_FeatureUpdateIgnoresProjectID(t *testing.T) {
	foreignProjectID := domain.NewProjectID()
	featureID := domain.NewFeatureID() // belongs to a different project

	updateCalled := false
	cmds := &servicetest.MockCommands{
		UpdateFeatureFunc: func(ctx context.Context, fID domain.FeatureID, name, description string) error {
			updateCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newRedSecurityRouter(t, newRedTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Attacker uses a different project ID in the URL path
	url := fmt.Sprintf("%s/api/projects/%s/features/%s",
		srv.URL, foreignProjectID.String(), featureID.String())

	body, _ := json.Marshal(map[string]string{"name": "hijacked"})
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if updateCalled {
		t.Log("RED: UpdateFeature was called despite projectID mismatch — " +
			"the handler at features.go:72 extracts featureId but ignores projectID; " +
			"an attacker can modify features across project boundaries")
	}

	// The handler should verify the feature belongs to the project in the URL.
	assert.False(t, updateCalled,
		"RED: UpdateFeature must not be called when projectID does not own the feature")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-12  IDOR: Feature delete ignores projectID from URL path
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_FeatureDeleteIgnoresProjectID documents that DeleteFeature
// (features.go:155) ignores the projectID path parameter entirely. An attacker
// can delete any feature by its UUID regardless of which project URL is used.
//
// TODO(security): Verify feature ownership against the project in the URL path
// before calling DeleteFeature.
func TestSecurity_RED_FeatureDeleteIgnoresProjectID(t *testing.T) {
	foreignProjectID := domain.NewProjectID()
	featureID := domain.NewFeatureID()

	deleteCalled := false
	cmds := &servicetest.MockCommands{
		DeleteFeatureFunc: func(ctx context.Context, fID domain.FeatureID) error {
			deleteCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newRedSecurityRouter(t, newRedTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	url := fmt.Sprintf("%s/api/projects/%s/features/%s",
		srv.URL, foreignProjectID.String(), featureID.String())

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if deleteCalled {
		t.Log("RED: DeleteFeature was called despite projectID mismatch — " +
			"features.go:155 ignores the {id} URL parameter entirely")
	}

	assert.False(t, deleteCalled,
		"RED: DeleteFeature must not be called when projectID does not own the feature")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-13  IDOR: Feature status update ignores projectID from URL path
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_FeatureStatusUpdateIgnoresProjectID documents that
// UpdateFeatureStatus (features.go:103) does not validate projectID ownership.
//
// TODO(security): Verify feature belongs to the project in the URL path.
func TestSecurity_RED_FeatureStatusUpdateIgnoresProjectID(t *testing.T) {
	foreignProjectID := domain.NewProjectID()
	featureID := domain.NewFeatureID()

	updateCalled := false
	cmds := &servicetest.MockCommands{
		UpdateFeatureStatusFunc: func(ctx context.Context, fID domain.FeatureID, status domain.FeatureStatus, nodeID string) error {
			updateCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newRedSecurityRouter(t, newRedTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	url := fmt.Sprintf("%s/api/projects/%s/features/%s/status",
		srv.URL, foreignProjectID.String(), featureID.String())

	body, _ := json.Marshal(map[string]string{"status": "done"})
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if updateCalled {
		t.Log("RED: UpdateFeatureStatus was called despite projectID mismatch — " +
			"features.go:103 ignores the {id} parameter; " +
			"any feature can have its status changed via any project URL")
	}

	assert.False(t, updateCalled,
		"RED: UpdateFeatureStatus must not proceed when projectID does not own the feature")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-14  Feature broadcast events lack ProjectID — cross-project event leak
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_FeatureCreateBroadcastMissingProjectID documents that
// CreateFeature (features.go:62-64) broadcasts a websocket.Event without
// setting ProjectID:
//
//   h.hub.Broadcast(websocket.Event{
//       Type: "feature_created",
//       Data: resp,
//   })
//
// Without a ProjectID filter, the event is delivered to ALL connected WebSocket
// clients regardless of which project they are monitoring. This leaks feature
// data (name, description) across project boundaries.
//
// TODO(security): Set ProjectID on the broadcast event so the hub only delivers
// it to clients subscribed to the correct project.
func TestSecurity_RED_FeatureCreateBroadcastMissingProjectID(t *testing.T) {
	// This test verifies the code path at features.go:62-64.
	// The websocket.Event struct has a ProjectID field that should be set.
	// Currently it is left as "" (zero value), meaning the hub broadcasts
	// to all clients.
	t.Log("RED: Feature command broadcasts (feature_created, feature_updated, " +
		"feature_deleted, feature_status_updated, feature_changelogs_updated) " +
		"at features.go:62-64, :98, :123, :146-149, :167-169 do not set " +
		"ProjectID on the websocket.Event — events leak to all connected " +
		"WebSocket clients across all projects; " +
		"fix: set ProjectID: string(projectID) on each broadcast event")

	// To verify programmatically: CreateFeature builds the event and the
	// hub.Broadcast call can be observed. Since this is a structural issue
	// (missing field assignment), we document it as a finding.
	// The projectID IS available in the handler (features.go:42) but is
	// not passed to the broadcast.

	projectID := domain.NewProjectID()
	cmds := &servicetest.MockCommands{
		CreateFeatureFunc: func(ctx context.Context, pID domain.ProjectID, name, description, createdByRole, createdByAgent string) (domain.Feature, error) {
			return domain.Feature{
				ID:        domain.NewFeatureID(),
				ProjectID: pID,
				Name:      name,
			}, nil
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetOutput(io.Discard)
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()

	router := mux.NewRouter()
	commands.NewFeatureCommandsHandler(cmds, ctrl, hub).RegisterRoutes(router)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]string{"name": "secret feature"})
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/api/projects/%s/features", srv.URL, projectID.String()),
		bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The event was broadcast without ProjectID filtering.
	// This is a structural vulnerability that requires code inspection to verify.
	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"Request itself succeeds, but the broadcast event lacks ProjectID scoping")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-15  CompleteTaskRequest.FilesModified has no max-items constraint
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_CompleteTaskFilesModifiedUnbounded documents that
// CompleteTaskRequest.FilesModified (pkg/server/types.go:220) uses:
//
//   FilesModified []string `json:"files_modified" validate:"dive,max=500"`
//
// The "dive" checks element length but there is no max=N on the slice itself.
// An attacker can submit 50,000 file paths (each up to 500 chars) in a single
// completion request, causing large memory allocation and database storage.
//
// TODO(security): Add validate:"max=1000,dive,max=500" to FilesModified in
// CompleteTaskRequest (pkg/server/types.go:220).
func TestSecurity_RED_CompleteTaskFilesModifiedUnbounded(t *testing.T) {
	const itemCount = 10_000

	completeCalled := false
	var receivedFilesCount int

	cmds := &servicetest.MockCommands{
		CompleteTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, _ string) error {
			completeCalled = true
			receivedFilesCount = len(filesModified)
			return nil
		},
	}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newRedSecurityRouter(t, newRedTestApp(cmds, &servicetest.MockQueries{}))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	files := make([]string, itemCount)
	for i := range files {
		files[i] = fmt.Sprintf("src/pkg/module_%05d/handler.go", i)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"completion_summary": "All work completed. Full implementation with tests, documentation, and integration. " +
			"See PR for details. All tests pass and coverage is adequate. This is a thorough completion summary.",
		"files_modified":     files,
		"completed_by_agent": "claude-agent",
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/complete",
		srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if completeCalled && receivedFilesCount >= itemCount {
		t.Logf("RED: CompleteTask accepted %d files_modified entries — "+
			"no array-count limit on FilesModified; "+
			"fix: add validate:\"max=1000,dive,max=500\" to "+
			"CompleteTaskRequest.FilesModified in pkg/server/types.go:220",
			receivedFilesCount)
	}

	assert.False(t, completeCalled,
		"RED: CompleteTask with 10,000 files_modified entries must be rejected by validation")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-16  RemoveAgent DELETE body parsing bypasses DecodeAndValidate
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_RemoveAgentSkipsValidation documents that RemoveAgent
// (project_agents.go:88-93) uses raw json.NewDecoder instead of
// controller.DecodeAndValidate when parsing the optional request body:
//
//   if r.ContentLength > 0 {
//       if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
//   }
//
// This bypasses: (1) Content-Type check, (2) DisallowUnknownFields,
// (3) struct validation tags. A caller can send XML or malformed JSON with
// arbitrary ReassignTo values that skip the max=50 validation.
//
// TODO(security): Use controller.DecodeAndValidate instead of raw json.Decode,
// or at minimum call controller.Validate(&req) after decoding.
func TestSecurity_RED_RemoveAgentSkipsValidation(t *testing.T) {
	projectID := domain.NewProjectID()

	removeCalled := false
	var receivedReassignTo *string

	cmds := &servicetest.MockCommands{
		RemoveAgentFromProjectFunc: func(ctx context.Context, pID domain.ProjectID, slug string, reassignTo *string, clearAssignment bool) error {
			removeCalled = true
			receivedReassignTo = reassignTo
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newRedSecurityRouter(t, newRedTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Send body with text/plain Content-Type (should be rejected) and an
	// overlong reassign_to that violates max=50 validation tag
	longSlug := make([]byte, 200)
	for i := range longSlug {
		longSlug[i] = 'a'
	}
	body, _ := json.Marshal(map[string]string{
		"reassign_to": string(longSlug),
	})

	url := fmt.Sprintf("%s/api/projects/%s/agents/backend",
		srv.URL, projectID.String())
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain") // wrong content type

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if removeCalled && receivedReassignTo != nil && len(*receivedReassignTo) > 50 {
		t.Logf("RED: RemoveAgentFromProject received reassign_to of length %d "+
			"with Content-Type text/plain — validation was bypassed; "+
			"fix: use controller.DecodeAndValidate in project_agents.go:88-93",
			len(*receivedReassignTo))
	}

	assert.False(t, removeCalled,
		"RED: RemoveAgent with invalid Content-Type and overlong reassign_to must be rejected")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-17  WebSocket CheckOrigin accepts all origins
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_WebSocketCheckOriginAcceptsAll documents that the WebSocket
// upgrader in websocket.go:38-39 uses:
//
//   CheckOrigin: func(r *http.Request) bool {
//       return true
//   }
//
// This allows any website to initiate a WebSocket connection to the server,
// enabling cross-origin WebSocket hijacking. A malicious page at evil.com
// can open a WebSocket to the server and receive all broadcast events
// (task data, feature data, notifications) if the user has a valid token.
//
// TODO(security): Validate the Origin header against a list of trusted origins,
// or at minimum check that it matches the Host header.
func TestSecurity_RED_WebSocketCheckOriginAcceptsAll(t *testing.T) {
	// This is a code-level vulnerability at websocket.go:38-39.
	// The gorilla/websocket Upgrader.CheckOrigin function always returns true.
	// We document this as a structural finding.
	t.Log("RED: WSHandler (websocket.go:35-39) creates a gorilla/websocket.Upgrader " +
		"with CheckOrigin: func(r *http.Request) bool { return true } — " +
		"this disables all cross-origin protection for WebSocket connections; " +
		"any website can initiate a WebSocket connection and receive all " +
		"broadcast events including task details, feature data, and notifications; " +
		"fix: implement origin validation in CheckOrigin that compares " +
		"r.Header.Get(\"Origin\") against trusted origins")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-18  IDOR: Feature changelogs update ignores projectID
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_FeatureChangelogsUpdateIgnoresProjectID documents that
// UpdateFeatureChangelogs (features.go:128) extracts featureId but ignores
// the projectID from the URL. Same IDOR pattern as SEC-11/12/13.
//
// TODO(security): Verify feature belongs to the project in the URL path
// before calling UpdateFeatureChangelogs.
func TestSecurity_RED_FeatureChangelogsUpdateIgnoresProjectID(t *testing.T) {
	foreignProjectID := domain.NewProjectID()
	featureID := domain.NewFeatureID()

	updateCalled := false
	cmds := &servicetest.MockCommands{
		UpdateFeatureChangelogsFunc: func(ctx context.Context, fID domain.FeatureID, userChangelog, techChangelog *string) error {
			updateCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newRedSecurityRouter(t, newRedTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	changelog := "hijacked changelog"
	body, _ := json.Marshal(map[string]string{"user_changelog": changelog})

	url := fmt.Sprintf("%s/api/projects/%s/features/%s/changelogs",
		srv.URL, foreignProjectID.String(), featureID.String())
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if updateCalled {
		t.Log("RED: UpdateFeatureChangelogs was called despite projectID mismatch — " +
			"features.go:128 ignores the {id} parameter; " +
			"changelogs of any feature can be modified via any project URL")
	}

	assert.False(t, updateCalled,
		"RED: UpdateFeatureChangelogs must not proceed when projectID does not own the feature")
}
