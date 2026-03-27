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
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/commands"
	"github.com/gorilla/mux"
	gorillaws "github.com/gorilla/websocket"
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

	router := mux.NewRouter()
	commands.NewRouter(router, app, ctrl, hub, "")
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

// TestSecurity_FeatureUpdateIgnoresProjectID verifies that UpdateFeature rejects
// requests where the featureID does not belong to the projectID in the URL path.
// The handler uses verifyFeatureOwnership to enforce this cross-resource check.
func TestSecurity_FeatureUpdateIgnoresProjectID(t *testing.T) {
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

	assert.False(t, updateCalled,
		"UpdateFeature must not be called when the feature does not belong to the project in the URL path")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-12  IDOR: Feature delete ignores projectID from URL path
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_FeatureDeleteIgnoresProjectID verifies that DeleteFeature rejects
// requests where the featureID does not belong to the projectID in the URL path.
func TestSecurity_FeatureDeleteIgnoresProjectID(t *testing.T) {
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

	assert.False(t, deleteCalled,
		"DeleteFeature must not be called when the feature does not belong to the project in the URL path")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-13  IDOR: Feature status update ignores projectID from URL path
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_FeatureStatusUpdateIgnoresProjectID verifies that UpdateFeatureStatus
// rejects requests where the featureID does not belong to the projectID in the URL path.
func TestSecurity_FeatureStatusUpdateIgnoresProjectID(t *testing.T) {
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

	assert.False(t, updateCalled,
		"UpdateFeatureStatus must not proceed when the feature does not belong to the project in the URL path")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-14  Feature broadcast events lack ProjectID — cross-project event leak
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_FeatureCreateBroadcastMissingProjectID verifies that
// CreateFeature (features.go:84-87) sets ProjectID on the broadcast event so
// the hub only delivers it to clients subscribed to the correct project.
//
// Currently the handler broadcasts:
//
//	h.hub.Broadcast(websocket.Event{
//	    Type: "feature_created",
//	    Data: resp,
//	})
//
// — ProjectID is absent (zero value ""), so the event leaks to ALL connected
// WebSocket clients across all projects.
//
// RED: This test FAILS until features.go sets ProjectID on the broadcast event.
func TestSecurity_RED_FeatureCreateBroadcastMissingProjectID(t *testing.T) {
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
	t.Cleanup(hub.Stop)

	// Register a WebSocket endpoint on the same router so we can receive events.
	router := mux.NewRouter()
	commands.NewFeatureCommandsHandler(cmds, ctrl, hub).RegisterRoutes(router)
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		upgrader := gorillaws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		var opts []websocket.ServeWSOption
		if pid := r.URL.Query().Get("project_id"); pid != "" {
			opts = append(opts, websocket.WithProjectID(pid))
		}
		hub.ServeWS(conn, opts...)
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Connect a WebSocket client that is scoped to the target project.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?project_id=" + projectID.String()
	wsConn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { wsConn.Close() })

	// Allow the client to register with the hub.
	time.Sleep(20 * time.Millisecond)

	// Trigger CreateFeature.
	body, _ := json.Marshal(map[string]string{"name": "secret feature"})
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/api/projects/%s/features", srv.URL, projectID.String()),
		bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()
	require.Equal(t, http.StatusOK, httpResp.StatusCode)

	// Read the broadcast event from the WebSocket client with a short deadline.
	wsConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err := wsConn.ReadMessage()
	require.NoError(t, err, "WebSocket client must receive the feature_created broadcast")

	var event struct {
		Type      string `json:"type"`
		ProjectID string `json:"project_id"`
	}
	require.NoError(t, json.Unmarshal(msg, &event))
	require.Equal(t, "feature_created", event.Type)

	// RED: This assertion fails until features.go sets ProjectID on the broadcast.
	assert.Equal(t, projectID.String(), event.ProjectID,
		"feature_created broadcast event must include ProjectID so the hub can scope "+
			"delivery to clients subscribed to the correct project; "+
			"fix: add ProjectID: string(projectID) to the websocket.Event in features.go:84-87")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-15  CompleteTaskRequest.FilesModified has no max-items constraint
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_CompleteTaskFilesModifiedUnbounded verifies that
// CompleteTaskRequest rejects a files_modified slice with more than the allowed
// number of entries. The validate tag must include a max=N constraint on the
// slice itself, not only a dive constraint on element length.
func TestSecurity_CompleteTaskFilesModifiedUnbounded(t *testing.T) {
	const itemCount = 10_000

	completeCalled := false

	cmds := &servicetest.MockCommands{
		CompleteTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, _ string) error {
			completeCalled = true
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

	assert.False(t, completeCalled,
		"CompleteTask with 10,000 files_modified entries must be rejected by validation; "+
			"add a max=N constraint to CompleteTaskRequest.FilesModified in pkg/server/types.go")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-16  RemoveAgent DELETE body parsing bypasses DecodeAndValidate
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RemoveAgentSkipsValidation verifies that RemoveAgent rejects a
// request body sent with the wrong Content-Type and an overlong reassign_to
// value that violates the max=50 validation tag.
// The handler must use controller.DecodeAndValidate instead of raw json.Decode
// so that Content-Type and struct validation are both enforced.
func TestSecurity_RemoveAgentSkipsValidation(t *testing.T) {
	projectID := domain.NewProjectID()

	removeCalled := false

	cmds := &servicetest.MockCommands{
		RemoveAgentFromProjectFunc: func(ctx context.Context, pID domain.ProjectID, slug string, reassignTo *string, clearAssignment bool) error {
			removeCalled = true
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

	assert.False(t, removeCalled,
		"RemoveAgent with invalid Content-Type and overlong reassign_to must be rejected by validation")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-17  WebSocket CheckOrigin accepts all origins
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_WebSocketCheckOriginAcceptsAll verifies that the WSHandler
// (websocket.go:38-39) rejects WebSocket upgrade requests from foreign origins.
//
// Currently the upgrader uses:
//
//	CheckOrigin: func(r *http.Request) bool { return true }
//
// — any origin is accepted, enabling cross-site WebSocket hijacking (CSWSH).
//
// RED: This test FAILS until WSHandler validates the Origin header.
func TestSecurity_RED_WebSocketCheckOriginAcceptsAll(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetOutput(io.Discard)
	hub := websocket.NewHub(logger)
	go hub.Run()
	t.Cleanup(hub.Stop)

	// Build a WSHandler with nil authQueries to bypass token validation.
	wsHandler := commands.NewWSHandler(nil, hub, logger, nil)

	router := mux.NewRouter()
	router.Handle("/ws", wsHandler).Methods("GET")

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	// Attempt a WebSocket upgrade from an untrusted foreign origin.
	header := http.Header{}
	header.Set("Origin", "http://evil.com")

	conn, resp, err := gorillaws.DefaultDialer.Dial(wsURL, header)
	if conn != nil {
		conn.Close()
	}

	if err == nil {
		// The upgrade succeeded — the origin check is absent.
		t.Fatalf("RED: WSHandler accepted a WebSocket upgrade from foreign origin "+
			"'http://evil.com' (status %d); "+
			"fix: replace CheckOrigin: func(*http.Request) bool { return true } in "+
			"websocket.go with origin validation against r.Host",
			resp.StatusCode)
	}

	// When the fix is in place the dial returns an error and resp carries 403.
	require.NotNil(t, resp, "expected an HTTP response even when the upgrade is rejected")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"RED: WSHandler must reject cross-origin WebSocket upgrades with 403; "+
			"fix: implement origin validation in CheckOrigin (websocket.go:38-39)")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-18  IDOR: Feature changelogs update ignores projectID
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_FeatureChangelogsUpdateIgnoresProjectID verifies that
// UpdateFeatureChangelogs rejects requests where the featureID does not belong
// to the projectID in the URL path.
func TestSecurity_FeatureChangelogsUpdateIgnoresProjectID(t *testing.T) {
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

	assert.False(t, updateCalled,
		"UpdateFeatureChangelogs must not proceed when the feature does not belong to the project in the URL path")
}
