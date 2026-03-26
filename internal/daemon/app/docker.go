package app

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
	"github.com/JLugagne/agach-mcp/internal/daemon/domain/repositories/builds"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	dbuild "github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DockerfileWithBuilds pairs a dockerfile slug with its build history.
type DockerfileWithBuilds struct {
	Slug      string
	LatestTag string // version of the most recent build
	Builds    []domain.DockerBuild
	IsHealthy bool // true if latest build succeeded
}

// BuildLogResult holds the result of a build log query.
type BuildLogResult struct {
	BuildID string
	Slug    string
	Version string
	Status  string
	Log     string
}

// PruneResult holds the outcome of a prune operation.
type PruneResult struct {
	Removed int
	Errors  []string
}

// DockerClient defines the Docker API operations used by the service.
type DockerClient interface {
	ImageBuild(ctx context.Context, buildContext io.Reader, options dbuild.ImageBuildOptions) (dbuild.ImageBuildResponse, error)
	ImageInspectWithRaw(ctx context.Context, imageID string) (image.InspectResponse, []byte, error)
	ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error)
}

// dockerBuildMessage represents a line of Docker build output JSON.
type dockerBuildMessage struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

// DockerService manages Docker image builds and lifecycle.
type DockerService struct {
	repo              builds.DockerBuildRepository
	docker            DockerClient
	dockerfileFetcher domain.DockerfileFetcher
	token             string
	logger            *logrus.Logger
}

// NewDockerService creates a new DockerService.
func NewDockerService(repo builds.DockerBuildRepository, docker DockerClient, fetcher domain.DockerfileFetcher, token string, logger *logrus.Logger) *DockerService {
	return &DockerService{
		repo:              repo,
		docker:            docker,
		dockerfileFetcher: fetcher,
		token:             token,
		logger:            logger,
	}
}

// ListImages returns all dockerfiles with their build history.
func (s *DockerService) ListImages(ctx context.Context) ([]DockerfileWithBuilds, error) {
	all, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]domain.DockerBuild)
	var slugOrder []string
	for _, b := range all {
		if _, exists := grouped[b.DockerfileSlug]; !exists {
			slugOrder = append(slugOrder, b.DockerfileSlug)
		}
		grouped[b.DockerfileSlug] = append(grouped[b.DockerfileSlug], b)
	}

	result := make([]DockerfileWithBuilds, 0, len(grouped))
	for _, slug := range slugOrder {
		builds := grouped[slug]
		sort.Slice(builds, func(i, j int) bool {
			return builds[i].CreatedAt.After(builds[j].CreatedAt)
		})
		dwb := DockerfileWithBuilds{
			Slug:   slug,
			Builds: builds,
		}
		if len(builds) > 0 {
			dwb.LatestTag = builds[0].Version
			dwb.IsHealthy = builds[0].Status == domain.BuildSuccess
		}
		result = append(result, dwb)
	}
	return result, nil
}

// Rebuild triggers a forced rebuild of a dockerfile and streams events.
func (s *DockerService) Rebuild(ctx context.Context, slug string, eventCh chan<- daemonws.BuildEvent) error {
	buildID := domain.BuildID(uuid.New().String())
	version := time.Now().UTC().Format("20060102-150405")

	// Send started event
	eventCh <- daemonws.BuildEvent{
		DockerfileSlug: slug,
		BuildID:        string(buildID),
		Status:         "started",
	}

	// Fetch dockerfile content from server
	df, err := s.dockerfileFetcher.GetDockerfileBySlug(ctx, s.token, slug)
	if err != nil {
		eventCh <- daemonws.BuildEvent{
			DockerfileSlug: slug,
			BuildID:        string(buildID),
			Status:         "failed",
			Log:            "failed to fetch dockerfile: " + err.Error(),
		}
		return nil
	}
	if df.Content == "" {
		eventCh <- daemonws.BuildEvent{
			DockerfileSlug: slug,
			BuildID:        string(buildID),
			Status:         "failed",
			Log:            "dockerfile content is empty",
		}
		return nil
	}

	// Use the server version for the build record
	if df.Version != "" {
		version = df.Version
	}

	// Create build record
	buildRecord := domain.DockerBuild{
		ID:             buildID,
		DockerfileSlug: slug,
		Version:        version,
		Status:         domain.BuildPending,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, buildRecord); err != nil {
		return fmt.Errorf("create build record: %w", err)
	}

	// Create tar build context with the Dockerfile
	buildContext, err := createBuildContext(df.Content)
	if err != nil {
		eventCh <- daemonws.BuildEvent{
			DockerfileSlug: slug,
			BuildID:        string(buildID),
			Status:         "failed",
			Log:            "failed to create build context: " + err.Error(),
		}
		_ = s.repo.UpdateStatus(ctx, buildID, domain.BuildFailed, err.Error())
		return nil
	}

	// Build the image
	tag := fmt.Sprintf("%s:%s", slug, version)
	resp, err := s.docker.ImageBuild(ctx, buildContext, dbuild.ImageBuildOptions{
		Tags:       []string{tag, slug + ":latest"},
		Dockerfile: "Dockerfile",
	})
	if err != nil {
		eventCh <- daemonws.BuildEvent{
			DockerfileSlug: slug,
			BuildID:        string(buildID),
			Status:         "failed",
			Log:            err.Error(),
		}
		_ = s.repo.UpdateStatus(ctx, buildID, domain.BuildFailed, err.Error())
		return nil
	}
	defer resp.Body.Close()

	// Stream build output
	var logBuf strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		var msg dockerBuildMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Error != "" {
			eventCh <- daemonws.BuildEvent{
				DockerfileSlug: slug,
				BuildID:        string(buildID),
				Status:         "failed",
				Log:            msg.Error,
			}
			_ = s.repo.UpdateStatus(ctx, buildID, domain.BuildFailed, logBuf.String()+"\n"+msg.Error)
			return nil
		}
		if msg.Stream != "" {
			logBuf.WriteString(msg.Stream)
		}
	}

	// Inspect the built image
	inspect, _, err := s.docker.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		eventCh <- daemonws.BuildEvent{
			DockerfileSlug: slug,
			BuildID:        string(buildID),
			Status:         "failed",
			Log:            err.Error(),
		}
		_ = s.repo.UpdateStatus(ctx, buildID, domain.BuildFailed, logBuf.String())
		return nil
	}

	_ = s.repo.UpdateStatus(ctx, buildID, domain.BuildSuccess, logBuf.String())

	eventCh <- daemonws.BuildEvent{
		DockerfileSlug: slug,
		BuildID:        string(buildID),
		Status:         "success",
		Log:            fmt.Sprintf("hash=%s size=%d", inspect.ID, inspect.Size),
	}

	return nil
}

// GetBuildLogs returns the build log for a given build ID.
func (s *DockerService) GetBuildLogs(ctx context.Context, buildID string) (*BuildLogResult, error) {
	build, err := s.repo.FindByID(ctx, domain.BuildID(buildID))
	if err != nil {
		return nil, err
	}
	if build == nil {
		return nil, nil
	}
	return &BuildLogResult{
		BuildID: string(build.ID),
		Slug:    build.DockerfileSlug,
		Version: build.Version,
		Status:  string(build.Status),
		Log:     build.BuildLog,
	}, nil
}

// PruneNonLatest removes non-latest images and their records.
func (s *DockerService) PruneNonLatest(ctx context.Context, eventCh chan<- daemonws.PruneEvent) (PruneResult, error) {
	eventCh <- daemonws.PruneEvent{Status: "started"}

	all, err := s.repo.ListAll(ctx)
	if err != nil {
		return PruneResult{}, fmt.Errorf("list all builds: %w", err)
	}

	// Group by slug
	grouped := make(map[string][]domain.DockerBuild)
	for _, b := range all {
		grouped[b.DockerfileSlug] = append(grouped[b.DockerfileSlug], b)
	}

	result := PruneResult{}

	for slug, builds := range grouped {
		// Find latest (most recent by created_at)
		var latest *domain.DockerBuild
		for i := range builds {
			if latest == nil || builds[i].CreatedAt.After(latest.CreatedAt) {
				latest = &builds[i]
			}
		}
		if latest == nil {
			continue
		}

		// Remove non-latest Docker images
		for _, b := range builds {
			if b.ID == latest.ID {
				continue
			}
			if b.ImageHash != "" {
				_, removeErr := s.docker.ImageRemove(ctx, b.ImageHash, image.RemoveOptions{Force: true})
				if removeErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s/%s: %v", slug, b.Version, removeErr))
					continue
				}
			}
		}

		// Delete non-latest records from repository
		deleted, delErr := s.repo.DeleteNonLatest(ctx, slug)
		if delErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("delete records for %s: %v", slug, delErr))
			continue
		}
		result.Removed += deleted

		eventCh <- daemonws.PruneEvent{
			DockerfileSlug: slug,
			Status:         "progress",
			Removed:        result.Removed,
		}
	}

	eventCh <- daemonws.PruneEvent{
		Status:  "completed",
		Removed: result.Removed,
	}

	return result, nil
}

// createBuildContext creates a tar archive containing the Dockerfile for Docker image build.
func createBuildContext(dockerfileContent string) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	content := []byte(dockerfileContent)
	if err := tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(len(content)),
		Mode: 0644,
	}); err != nil {
		return nil, fmt.Errorf("write tar header: %w", err)
	}
	if _, err := tw.Write(content); err != nil {
		return nil, fmt.Errorf("write tar content: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close tar: %w", err)
	}

	return &buf, nil
}
