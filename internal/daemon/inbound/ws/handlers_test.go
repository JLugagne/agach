package ws_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/app"
	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
	"github.com/JLugagne/agach-mcp/internal/daemon/inbound/ws"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type mockDockerService struct {
	listResult    []app.DockerfileWithBuilds
	listErr       error
	rebuildErr    error
	rebuildEvents []daemonws.BuildEvent
	logsResult    *app.BuildLogResult
	logsErr       error
	pruneResult   app.PruneResult
	pruneErr      error
	pruneEvents   []daemonws.PruneEvent
}

func (m *mockDockerService) ListImages(_ context.Context) ([]app.DockerfileWithBuilds, error) {
	return m.listResult, m.listErr
}

func (m *mockDockerService) Rebuild(_ context.Context, slug string, eventCh chan<- daemonws.BuildEvent) error {
	for _, e := range m.rebuildEvents {
		eventCh <- e
	}
	return m.rebuildErr
}

func (m *mockDockerService) GetBuildLogs(_ context.Context, buildID string) (*app.BuildLogResult, error) {
	return m.logsResult, m.logsErr
}

func (m *mockDockerService) PruneNonLatest(_ context.Context, eventCh chan<- daemonws.PruneEvent) (app.PruneResult, error) {
	for _, e := range m.pruneEvents {
		eventCh <- e
	}
	return m.pruneResult, m.pruneErr
}

func newTestHandlers(mock *mockDockerService) *ws.Handlers {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return ws.NewHandlers(mock, nil, logger)
}

func TestHandleListDockerfiles(t *testing.T) {
	now := time.Date(2026, 3, 22, 14, 30, 0, 0, time.UTC)
	mock := &mockDockerService{
		listResult: []app.DockerfileWithBuilds{
			{
				Slug:      "agent-base",
				LatestTag: "v1.3.2",
				IsHealthy: true,
				Builds: []domain.DockerBuild{
					{ID: "b1", DockerfileSlug: "agent-base", Version: "v1.3.2", ImageHash: "sha256:a3f8c2e91b", ImageSize: 245000000, Status: domain.BuildSuccess, CreatedAt: now},
					{ID: "b2", DockerfileSlug: "agent-base", Version: "v1.3.1", ImageHash: "sha256:prev1", ImageSize: 240000000, Status: domain.BuildSuccess, CreatedAt: now.Add(-24 * time.Hour)},
					{ID: "b3", DockerfileSlug: "agent-base", Version: "v1.3.0", ImageHash: "sha256:prev2", ImageSize: 238000000, Status: domain.BuildSuccess, CreatedAt: now.Add(-48 * time.Hour)},
					{ID: "b4", DockerfileSlug: "agent-base", Version: "v1.2.0", ImageHash: "sha256:prev3", ImageSize: 230000000, Status: domain.BuildSuccess, CreatedAt: now.Add(-72 * time.Hour)},
				},
			},
			{
				Slug:      "tools",
				LatestTag: "v2.1.0",
				IsHealthy: true,
				Builds: []domain.DockerBuild{
					{ID: "t1", DockerfileSlug: "tools", Version: "v2.1.0", ImageHash: "sha256:tool1", ImageSize: 120000000, Status: domain.BuildSuccess, CreatedAt: now},
					{ID: "t2", DockerfileSlug: "tools", Version: "v2.0.0", ImageHash: "sha256:tool2", ImageSize: 115000000, Status: domain.BuildSuccess, CreatedAt: now.Add(-48 * time.Hour)},
					{ID: "t3", DockerfileSlug: "tools", Version: "v1.0.0", ImageHash: "sha256:tool3", ImageSize: 100000000, Status: domain.BuildSuccess, CreatedAt: now.Add(-96 * time.Hour)},
				},
			},
			{
				Slug:      "gpu-worker",
				LatestTag: "v1.0.1",
				IsHealthy: false,
				Builds: []domain.DockerBuild{
					{ID: "g1", DockerfileSlug: "gpu-worker", Version: "v1.0.1", ImageHash: "", ImageSize: 0, Status: domain.BuildFailed, CreatedAt: now},
					{ID: "g2", DockerfileSlug: "gpu-worker", Version: "v1.0.0", ImageHash: "sha256:gpu1", ImageSize: 500000000, Status: domain.BuildSuccess, CreatedAt: now.Add(-24 * time.Hour)},
				},
			},
		},
	}

	h := newTestHandlers(mock)
	resp, err := h.HandleListDockerfiles(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerList,
		RequestID: "req-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload, "response payload must not be empty")

	var listResp ws.ListDockerfilesResponse
	err = json.Unmarshal(resp.Payload, &listResp)
	require.NoError(t, err)
	require.Len(t, listResp.Dockerfiles, 3)

	// Check agent-base
	ab := listResp.Dockerfiles[0]
	require.Equal(t, "agent-base", ab.Slug)
	require.Equal(t, "v1.3.2", ab.LatestVersion)
	require.Equal(t, 4, ab.VersionCount)
	require.True(t, ab.IsHealthy)
	require.Len(t, ab.Builds, 4)
	require.Equal(t, "v1.3.2", ab.Builds[0].Version)
	require.True(t, ab.Builds[0].IsLatest)
	require.Equal(t, int64(245000000), ab.Builds[0].SizeBytes)

	// Check gpu-worker (unhealthy)
	gw := listResp.Dockerfiles[2]
	require.Equal(t, "gpu-worker", gw.Slug)
	require.False(t, gw.IsHealthy)
}

func TestHandleRebuild_Success(t *testing.T) {
	mock := &mockDockerService{
		rebuildEvents: []daemonws.BuildEvent{
			{DockerfileSlug: "agent-base", Status: "started"},
			{DockerfileSlug: "agent-base", Status: "success"},
		},
	}

	h := newTestHandlers(mock)
	payload, _ := json.Marshal(map[string]string{"slug": "agent-base"})
	resp, err := h.HandleRebuild(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerRebuild,
		RequestID: "req-rebuild",
		Payload:   payload,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload, "must return acknowledgment payload")

	var ack map[string]string
	require.NoError(t, json.Unmarshal(resp.Payload, &ack))
	require.Equal(t, "started", ack["status"])
	require.Equal(t, "agent-base", ack["slug"])
}

func TestHandleRebuild_InvalidPayload(t *testing.T) {
	mock := &mockDockerService{}
	h := newTestHandlers(mock)

	resp, err := h.HandleRebuild(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerRebuild,
		RequestID: "req-bad",
		Payload:   json.RawMessage(`{invalid`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Error, "must return error for invalid payload")
}

func TestHandleRebuild_EmptySlug(t *testing.T) {
	mock := &mockDockerService{}
	h := newTestHandlers(mock)

	payload, _ := json.Marshal(map[string]string{"slug": ""})
	resp, err := h.HandleRebuild(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerRebuild,
		RequestID: "req-empty",
		Payload:   payload,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Error, "must return error for empty slug")
}

func TestHandleGetLogs_Success(t *testing.T) {
	mock := &mockDockerService{
		logsResult: &app.BuildLogResult{
			BuildID: "abc123",
			Slug:    "agent-base",
			Version: "v1.3.2",
			Status:  "success",
			Log:     "Step 1/10 : FROM golang:1.21\n---> abc123\nStep 2/10 : WORKDIR /app\n",
		},
	}

	h := newTestHandlers(mock)
	payload, _ := json.Marshal(map[string]string{"build_id": "abc123"})
	resp, err := h.HandleGetLogs(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerLogs,
		RequestID: "req-logs",
		Payload:   payload,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload)

	var logsResp ws.GetLogsResponse
	require.NoError(t, json.Unmarshal(resp.Payload, &logsResp))
	require.Equal(t, "abc123", logsResp.BuildID)
	require.Equal(t, "agent-base", logsResp.Slug)
	require.Equal(t, "v1.3.2", logsResp.Version)
	require.Equal(t, "success", logsResp.Status)
	require.Contains(t, logsResp.Log, "FROM golang:1.21")
	require.False(t, logsResp.InProgress)
}

func TestHandleGetLogs_NotFound(t *testing.T) {
	mock := &mockDockerService{logsResult: nil}
	h := newTestHandlers(mock)

	payload, _ := json.Marshal(map[string]string{"build_id": "nonexistent"})
	resp, err := h.HandleGetLogs(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerLogs,
		RequestID: "req-notfound",
		Payload:   payload,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Error, "must return error for not found")
}

func TestHandleGetLogs_BuildInProgress(t *testing.T) {
	mock := &mockDockerService{
		logsResult: &app.BuildLogResult{
			BuildID: "inprog1",
			Slug:    "agent-base",
			Version: "v1.4.0",
			Status:  "building",
			Log:     "Step 1/10 : FROM golang:1.21\n",
		},
	}

	h := newTestHandlers(mock)
	payload, _ := json.Marshal(map[string]string{"build_id": "inprog1"})
	resp, err := h.HandleGetLogs(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerLogs,
		RequestID: "req-inprog",
		Payload:   payload,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload)

	var logsResp ws.GetLogsResponse
	require.NoError(t, json.Unmarshal(resp.Payload, &logsResp))
	require.True(t, logsResp.InProgress, "must indicate in progress")
	require.Equal(t, "building", logsResp.Status)
}

func TestHandlePrune_Success(t *testing.T) {
	mock := &mockDockerService{
		pruneResult: app.PruneResult{Removed: 5},
		pruneEvents: []daemonws.PruneEvent{
			{Status: "started"},
			{DockerfileSlug: "agent-base", Status: "progress", Removed: 1},
			{Status: "completed", Removed: 5},
		},
	}

	h := newTestHandlers(mock)
	resp, err := h.HandlePrune(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerPrune,
		RequestID: "req-prune",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload, "must return acknowledgment payload")

	var ack map[string]string
	require.NoError(t, json.Unmarshal(resp.Payload, &ack))
	require.Equal(t, "started", ack["status"])
}

func TestHandlePrune_NoImagesToDelete(t *testing.T) {
	mock := &mockDockerService{
		pruneResult: app.PruneResult{Removed: 0},
		pruneEvents: []daemonws.PruneEvent{
			{Status: "started"},
			{Status: "completed", Removed: 0},
		},
	}

	h := newTestHandlers(mock)
	resp, err := h.HandlePrune(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerPrune,
		RequestID: "req-prune-empty",
	})
	require.NoError(t, err)
	require.Empty(t, resp.Error, "no error for 0 images")
	require.NotEmpty(t, resp.Payload)
}

func TestHandlePrune_PartialFailure(t *testing.T) {
	mock := &mockDockerService{
		pruneResult: app.PruneResult{
			Removed: 3,
			Errors:  []string{"failed to remove sha256:abc"},
		},
		pruneEvents: []daemonws.PruneEvent{
			{Status: "started"},
			{Status: "completed", Removed: 3},
		},
	}

	h := newTestHandlers(mock)
	resp, err := h.HandlePrune(context.Background(), daemonws.Message{
		Type:      daemonws.TypeDockerPrune,
		RequestID: "req-prune-partial",
	})
	require.NoError(t, err)
	require.Empty(t, resp.Error, "partial success is still success")
	require.NotEmpty(t, resp.Payload)
}
