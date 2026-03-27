package commands_test

// This file tests that the HTTP command handlers correctly broadcast
// wip_slot_available events when tasks transition out of in_progress.
//
// Strategy: we connect a real WebSocket client to the hub, make the HTTP
// request, then collect events from the WS connection and assert on their types.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
)

// newTestTaskHandlerAndHub creates a TaskCommandsHandler together with its Hub,
// allowing tests to connect WebSocket clients and observe broadcast events.
func newTestTaskHandlerAndHub(t *testing.T, mock *servicetest.MockCommands) (*commands.TaskCommandsHandler, *websocket.Hub) {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer(nil))
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()
	handler := commands.NewTaskCommandsHandler(mock, ctrl, hub, nil)
	return handler, hub
}

// connectWSClientToHub spins up a temporary HTTP server that upgrades
// connections to WebSocket and delegates them to hub.ServeWS.
// It returns a connected gorilla WebSocket client and waits for client
// registration to complete before returning.
func connectWSClientToHub(t *testing.T, hub *websocket.Hub, projectIDs ...string) *gorillaws.Conn {
	t.Helper()
	upgrader := gorillaws.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "upgrade failed", http.StatusInternalServerError)
			return
		}
		var opts []websocket.ServeWSOption
		if pid := r.URL.Query().Get("project_id"); pid != "" {
			opts = append(opts, websocket.WithProjectID(pid))
		}
		hub.ServeWS(conn, opts...)
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	if len(projectIDs) > 0 && projectIDs[0] != "" {
		url += "/?project_id=" + projectIDs[0]
	}
	conn, _, err := gorillaws.DefaultDialer.Dial(url, nil)
	require.NoError(t, err, "failed to dial WebSocket test server")
	t.Cleanup(func() { conn.Close() })

	// Allow the hub's Run loop to process the register request.
	time.Sleep(20 * time.Millisecond)
	return conn
}

// collectWSEvents reads WebSocket events until the per-message read deadline
// expires, collecting all events that arrive. This lets a test gather all
// broadcasts triggered by a single HTTP request.
func collectWSEvents(t *testing.T, conn *gorillaws.Conn, perMsgTimeout time.Duration) []websocket.Event {
	t.Helper()
	var events []websocket.Event
	for {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(perMsgTimeout)))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break // deadline exceeded — no more events
		}
		var event websocket.Event
		require.NoError(t, json.Unmarshal(msg, &event))
		events = append(events, event)
	}
	require.NoError(t, conn.SetReadDeadline(time.Time{}))
	return events
}

// hasEventOfType returns true if any event in the slice has the given type.
func hasEventOfType(events []websocket.Event, eventType string) bool {
	for _, e := range events {
		if e.Type == eventType {
			return true
		}
	}
	return false
}

// TestMoveTask_ToBacklog_EmitsWIPSlotAvailable verifies that moving a task to
// backlog (i.e., out of in_progress) causes the HTTP handler to broadcast a
// wip_slot_available event in addition to task_moved.
func TestMoveTask_ToBacklog_EmitsWIPSlotAvailable(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		MoveTaskFunc: func(_ context.Context, pid domain.ProjectID, tid domain.TaskID, targetColumn domain.ColumnSlug, _ string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			assert.Equal(t, domain.ColumnBacklog, targetColumn)
			return nil
		},
	}

	handler, hub := newTestTaskHandlerAndHub(t, mock)
	conn := connectWSClientToHub(t, hub, string(projectID))

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"target_column": "backlog"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/move",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// Collect all broadcast events. Use 300 ms per-message timeout so we catch
	// both task_moved and wip_slot_available which arrive in quick succession.
	events := collectWSEvents(t, conn, 300*time.Millisecond)

	assert.True(t, hasEventOfType(events, "task_moved"),
		"expected task_moved event, got: %v", events)
	assert.True(t, hasEventOfType(events, "wip_slot_available"),
		"expected wip_slot_available event when moving out of in_progress, got: %v", events)
}

// TestCompleteTask_EmitsWIPSlotAvailable verifies that completing a task
// broadcasts a wip_slot_available event alongside task_completed.
func TestCompleteTask_EmitsWIPSlotAvailable(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		CompleteTaskFunc: func(_ context.Context, pid domain.ProjectID, tid domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, _ string) error {
			assert.Equal(t, projectID, pid)
			assert.Equal(t, taskID, tid)
			return nil
		},
	}

	handler, hub := newTestTaskHandlerAndHub(t, mock)
	conn := connectWSClientToHub(t, hub, string(projectID))

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	summary := strings.Repeat("x", 100)
	body := `{"completion_summary": "` + summary + `", "completed_by_agent": "agent-007", "files_modified": []}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/complete",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	events := collectWSEvents(t, conn, 300*time.Millisecond)

	assert.True(t, hasEventOfType(events, "task_completed"),
		"expected task_completed event, got: %v", events)
	assert.True(t, hasEventOfType(events, "wip_slot_available"),
		"expected wip_slot_available event after CompleteTask, got: %v", events)
}

// TestMoveTask_ToInProgress_DoesNotEmitWIPSlotAvailable verifies that moving a
// task INTO in_progress does NOT broadcast wip_slot_available — only task_moved.
//
// Note: BlockTask has no HTTP handler in the commands package; the MCP-layer
// equivalent is covered by TestBlockTask_EmitsBroadcast in
// internal/server/inbound/mcp/tools_ws_internal_test.go.
func TestMoveTask_ToInProgress_DoesNotEmitWIPSlotAvailable(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mock := &servicetest.MockCommands{
		MoveTaskFunc: func(_ context.Context, pid domain.ProjectID, tid domain.TaskID, targetColumn domain.ColumnSlug, _ string) error {
			assert.Equal(t, domain.ColumnInProgress, targetColumn)
			return nil
		},
	}

	handler, hub := newTestTaskHandlerAndHub(t, mock)
	conn := connectWSClientToHub(t, hub, string(projectID))

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := `{"target_column": "in_progress"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/move",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// 100 ms is enough to catch any spurious extra event: task_moved arrives
	// almost instantly; if wip_slot_available were emitted it would also be here.
	events := collectWSEvents(t, conn, 100*time.Millisecond)

	assert.True(t, hasEventOfType(events, "task_moved"),
		"expected task_moved event, got: %v", events)
	assert.False(t, hasEventOfType(events, "wip_slot_available"),
		"must NOT emit wip_slot_available when moving into in_progress, got: %v", events)
}
