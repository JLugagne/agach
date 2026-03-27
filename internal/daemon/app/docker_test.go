package app_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/app"
	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type mockBuildRepo struct {
	builds               []domain.DockerBuild
	createCalled         bool
	updateCalled         bool
	lastStatus           domain.BuildStatus
	lastLog              string
	deleteNonLatestSlugs []string
}

func (m *mockBuildRepo) Create(_ context.Context, b domain.DockerBuild) error {
	m.builds = append(m.builds, b)
	m.createCalled = true
	return nil
}

func (m *mockBuildRepo) FindByID(_ context.Context, id domain.BuildID) (*domain.DockerBuild, error) {
	for _, b := range m.builds {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, nil
}

func (m *mockBuildRepo) ListByDockerfile(_ context.Context, slug string) ([]domain.DockerBuild, error) {
	var result []domain.DockerBuild
	for _, b := range m.builds {
		if b.DockerfileSlug == slug {
			result = append(result, b)
		}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

func (m *mockBuildRepo) ListAll(_ context.Context) ([]domain.DockerBuild, error) {
	return m.builds, nil
}

func (m *mockBuildRepo) UpdateStatus(_ context.Context, id domain.BuildID, status domain.BuildStatus, log string) error {
	m.updateCalled = true
	m.lastStatus = status
	m.lastLog = log
	for i, b := range m.builds {
		if b.ID == id {
			m.builds[i].Status = status
			m.builds[i].BuildLog = log
		}
	}
	return nil
}

func (m *mockBuildRepo) Delete(_ context.Context, id domain.BuildID) error {
	for i, b := range m.builds {
		if b.ID == id {
			m.builds = append(m.builds[:i], m.builds[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockBuildRepo) DeleteNonLatest(_ context.Context, slug string) (int, error) {
	m.deleteNonLatestSlugs = append(m.deleteNonLatestSlugs, slug)
	// Find latest (most recent) and count non-latest
	var latest *domain.DockerBuild
	count := 0
	for _, b := range m.builds {
		if b.DockerfileSlug == slug {
			if latest == nil || b.CreatedAt.After(latest.CreatedAt) {
				latest = &b
			}
		}
	}
	var remaining []domain.DockerBuild
	for _, b := range m.builds {
		if b.DockerfileSlug == slug && latest != nil && b.ID != latest.ID {
			count++
		} else {
			remaining = append(remaining, b)
		}
	}
	m.builds = remaining
	return count, nil
}

type mockDockerClient struct {
	buildResp    build.ImageBuildResponse
	buildErr     error
	inspectResp  image.InspectResponse
	inspectErr   error
	removeResp   []image.DeleteResponse
	removeErr    error
	removeCalls  []string
	removeErrors map[string]error // per-imageID errors
}

func (m *mockDockerClient) ImageBuild(_ context.Context, _ io.Reader, _ build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	return m.buildResp, m.buildErr
}

func (m *mockDockerClient) ImageInspectWithRaw(_ context.Context, _ string) (image.InspectResponse, []byte, error) {
	return m.inspectResp, nil, m.inspectErr
}

func (m *mockDockerClient) ImageRemove(_ context.Context, imageID string, _ image.RemoveOptions) ([]image.DeleteResponse, error) {
	m.removeCalls = append(m.removeCalls, imageID)
	if m.removeErrors != nil {
		if err, ok := m.removeErrors[imageID]; ok {
			return nil, err
		}
	}
	if m.removeErr != nil {
		return nil, m.removeErr
	}
	return m.removeResp, nil
}

type mockDockerfileFetcher struct {
	content *domain.DockerfileContent
	err     error
}

func (m *mockDockerfileFetcher) GetDockerfileBySlug(_ context.Context, _, _ string) (*domain.DockerfileContent, error) {
	return m.content, m.err
}

func newTestLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func TestDockerService_ListImages(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	mock := &mockBuildRepo{
		builds: []domain.DockerBuild{
			{ID: "a1", DockerfileSlug: "agent-base", Version: "v1.0", Status: domain.BuildSuccess, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "a2", DockerfileSlug: "agent-base", Version: "v1.1", Status: domain.BuildSuccess, CreatedAt: now.Add(-1 * time.Hour)},
			{ID: "a3", DockerfileSlug: "agent-base", Version: "v1.2", Status: domain.BuildSuccess, CreatedAt: now},
			{ID: "t1", DockerfileSlug: "tools", Version: "v1.0", Status: domain.BuildSuccess, CreatedAt: now.Add(-1 * time.Hour)},
			{ID: "t2", DockerfileSlug: "tools", Version: "v1.1", Status: domain.BuildFailed, CreatedAt: now},
		},
	}

	svc := app.NewDockerService(mock, &mockDockerClient{}, nil, "", newTestLogger())

	result, err := svc.ListImages(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result, "ListImages must return non-nil result")
	require.Len(t, result, 2, "must return 2 dockerfile groups")

	groups := make(map[string]app.DockerfileWithBuilds)
	for _, g := range result {
		groups[g.Slug] = g
	}

	agentBase := groups["agent-base"]
	require.Equal(t, "agent-base", agentBase.Slug)
	require.Len(t, agentBase.Builds, 3)
	require.Equal(t, "v1.2", agentBase.LatestTag)
	require.True(t, agentBase.IsHealthy, "agent-base latest build succeeded")
	require.Equal(t, domain.BuildID("a3"), agentBase.Builds[0].ID)
	require.Equal(t, domain.BuildID("a2"), agentBase.Builds[1].ID)
	require.Equal(t, domain.BuildID("a1"), agentBase.Builds[2].ID)

	tools := groups["tools"]
	require.Equal(t, "tools", tools.Slug)
	require.Len(t, tools.Builds, 2)
	require.Equal(t, "v1.1", tools.LatestTag)
	require.False(t, tools.IsHealthy, "tools latest build failed")
}

func TestDockerService_Rebuild_Success(t *testing.T) {
	repo := &mockBuildRepo{}
	docker := &mockDockerClient{
		buildResp: build.ImageBuildResponse{
			Body: io.NopCloser(strings.NewReader(`{"stream":"Step 1/3 : FROM golang:1.21\n"}
{"stream":"Step 2/3 : COPY . .\n"}
{"stream":"Step 3/3 : RUN go build\n"}
{"stream":"Successfully built abc123\n"}`)),
		},
		inspectResp: image.InspectResponse{
			ID:   "sha256:abc123def456",
			Size: 245000000,
		},
	}

	fetcher := &mockDockerfileFetcher{
		content: &domain.DockerfileContent{Slug: "agent-base", Version: "v1.0", Content: "FROM golang:1.21\nCOPY . .\nRUN go build"},
	}
	svc := app.NewDockerService(repo, docker, fetcher, "test-token", newTestLogger())
	eventCh := make(chan daemonws.BuildEvent, 10)

	err := svc.Rebuild(context.Background(), "agent-base", eventCh)
	require.NoError(t, err)
	close(eventCh)

	// Collect events
	var events []daemonws.BuildEvent
	for e := range eventCh {
		events = append(events, e)
	}

	require.GreaterOrEqual(t, len(events), 2, "must receive at least started + completed events")

	// First event: started
	require.Equal(t, "started", events[0].Status)
	require.Equal(t, "agent-base", events[0].DockerfileSlug)

	// Last event: completed with success
	last := events[len(events)-1]
	require.Equal(t, "success", last.Status)

	// Verify repo interactions
	require.True(t, repo.createCalled, "Create must be called")
	require.True(t, repo.updateCalled, "UpdateStatus must be called")
	require.Equal(t, domain.BuildSuccess, repo.lastStatus)
}

func TestDockerService_Rebuild_Failure(t *testing.T) {
	repo := &mockBuildRepo{}
	docker := &mockDockerClient{
		buildErr: errors.New("docker build failed: out of disk"),
	}
	fetcher := &mockDockerfileFetcher{
		content: &domain.DockerfileContent{Slug: "agent-base", Version: "v1.0", Content: "FROM golang:1.21"},
	}

	svc := app.NewDockerService(repo, docker, fetcher, "test-token", newTestLogger())
	eventCh := make(chan daemonws.BuildEvent, 10)

	err := svc.Rebuild(context.Background(), "agent-base", eventCh)
	require.NoError(t, err)
	close(eventCh)

	var events []daemonws.BuildEvent
	for e := range eventCh {
		events = append(events, e)
	}

	require.GreaterOrEqual(t, len(events), 2, "must receive at least started + failed events")

	// First event: started
	require.Equal(t, "started", events[0].Status)

	// Last event: failed
	last := events[len(events)-1]
	require.Equal(t, "failed", last.Status)

	// Verify repo UpdateStatus was called with Failed
	require.True(t, repo.updateCalled, "UpdateStatus must be called")
	require.Equal(t, domain.BuildFailed, repo.lastStatus)
}

func TestDockerService_PruneNonLatest(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	repo := &mockBuildRepo{
		builds: []domain.DockerBuild{
			{ID: "a1", DockerfileSlug: "agent-base", Version: "v1.0", ImageHash: "sha256:aaa", ImageSize: 100000, Status: domain.BuildSuccess, CreatedAt: now.Add(-3 * time.Hour)},
			{ID: "a2", DockerfileSlug: "agent-base", Version: "v1.1", ImageHash: "sha256:bbb", ImageSize: 100000, Status: domain.BuildSuccess, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "a3", DockerfileSlug: "agent-base", Version: "v1.2", ImageHash: "sha256:ccc", ImageSize: 100000, Status: domain.BuildSuccess, CreatedAt: now.Add(-1 * time.Hour)},
			{ID: "a4", DockerfileSlug: "agent-base", Version: "v1.3", ImageHash: "sha256:ddd", ImageSize: 100000, Status: domain.BuildSuccess, CreatedAt: now},
			{ID: "t1", DockerfileSlug: "tools", Version: "v1.0", ImageHash: "sha256:eee", ImageSize: 50000, Status: domain.BuildSuccess, CreatedAt: now.Add(-1 * time.Hour)},
			{ID: "t2", DockerfileSlug: "tools", Version: "v1.1", ImageHash: "sha256:fff", ImageSize: 50000, Status: domain.BuildSuccess, CreatedAt: now},
		},
	}
	docker := &mockDockerClient{}

	svc := app.NewDockerService(repo, docker, nil, "", newTestLogger())
	eventCh := make(chan daemonws.PruneEvent, 20)

	result, err := svc.PruneNonLatest(context.Background(), eventCh)
	require.NoError(t, err)
	close(eventCh)

	var events []daemonws.PruneEvent
	for e := range eventCh {
		events = append(events, e)
	}

	require.GreaterOrEqual(t, len(events), 2, "must have at least started + completed events")
	require.Equal(t, 4, result.Removed, "must remove 4 non-latest images (3 agent-base + 1 tools)")

	// Verify repo DeleteNonLatest was called for each slug
	require.Contains(t, repo.deleteNonLatestSlugs, "agent-base")
	require.Contains(t, repo.deleteNonLatestSlugs, "tools")
}

func TestDockerService_PruneNonLatest_PartialFailure(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	repo := &mockBuildRepo{
		builds: []domain.DockerBuild{
			{ID: "a1", DockerfileSlug: "agent-base", Version: "v1.0", ImageHash: "sha256:aaa", Status: domain.BuildSuccess, CreatedAt: now.Add(-1 * time.Hour)},
			{ID: "a2", DockerfileSlug: "agent-base", Version: "v1.1", ImageHash: "sha256:bbb", Status: domain.BuildSuccess, CreatedAt: now},
		},
	}
	docker := &mockDockerClient{
		removeErrors: map[string]error{
			"sha256:aaa": errors.New("image in use by container"),
		},
	}

	svc := app.NewDockerService(repo, docker, nil, "", newTestLogger())
	eventCh := make(chan daemonws.PruneEvent, 20)

	result, err := svc.PruneNonLatest(context.Background(), eventCh)
	require.NoError(t, err)
	close(eventCh)

	require.Len(t, result.Errors, 1, "must report 1 error")
}
