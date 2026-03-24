package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
	"github.com/JLugagne/agach-mcp/internal/daemon/domain/repositories/builds"
	"github.com/JLugagne/agach-mcp/internal/daemon/outbound/sqlite"
	"github.com/stretchr/testify/require"
)

func newTestRepo(t *testing.T) builds.DockerBuildRepository {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.NewDB(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	err = sqlite.RunMigrations(db)
	require.NoError(t, err)
	return sqlite.NewBuildRepository(db)
}

func TestBuildRepository_Create(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	build := domain.DockerBuild{
		ID:             "build-001",
		DockerfileSlug: "go-agent",
		Version:        "v1.0",
		ImageHash:      "sha256:abc123",
		ImageSize:      1024000,
		Status:         domain.BuildPending,
		BuildLog:       "",
		CreatedAt:      now,
	}

	err := repo.Create(ctx, build)
	require.NoError(t, err)

	got, err := repo.FindByID(ctx, "build-001")
	require.NoError(t, err)
	require.NotNil(t, got, "FindByID must return the created build")
	require.Equal(t, build.ID, got.ID)
	require.Equal(t, build.DockerfileSlug, got.DockerfileSlug)
	require.Equal(t, build.Version, got.Version)
	require.Equal(t, build.ImageHash, got.ImageHash)
	require.Equal(t, build.ImageSize, got.ImageSize)
	require.Equal(t, build.Status, got.Status)
	require.Equal(t, build.CreatedAt, got.CreatedAt)
}

func TestBuildRepository_ListByDockerfile(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	builds := []domain.DockerBuild{
		{ID: "b1", DockerfileSlug: "go-agent", Version: "v1.0", Status: domain.BuildSuccess, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "b2", DockerfileSlug: "go-agent", Version: "v1.1", Status: domain.BuildSuccess, CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "b3", DockerfileSlug: "go-agent", Version: "v1.2", Status: domain.BuildPending, CreatedAt: now},
		{ID: "b4", DockerfileSlug: "python-agent", Version: "v1.0", Status: domain.BuildSuccess, CreatedAt: now},
	}
	for _, b := range builds {
		require.NoError(t, repo.Create(ctx, b))
	}

	got, err := repo.ListByDockerfile(ctx, "go-agent")
	require.NoError(t, err)
	require.Len(t, got, 3, "must return only go-agent builds")
	// Ordered by created_at DESC
	require.Equal(t, domain.BuildID("b3"), got[0].ID)
	require.Equal(t, domain.BuildID("b2"), got[1].ID)
	require.Equal(t, domain.BuildID("b1"), got[2].ID)
}

func TestBuildRepository_UpdateStatus(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	build := domain.DockerBuild{
		ID:             "build-upd",
		DockerfileSlug: "go-agent",
		Version:        "v2.0",
		Status:         domain.BuildPending,
		CreatedAt:      now,
	}
	require.NoError(t, repo.Create(ctx, build))

	err := repo.UpdateStatus(ctx, "build-upd", domain.BuildSuccess, "build completed successfully")
	require.NoError(t, err)

	got, err := repo.FindByID(ctx, "build-upd")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, domain.BuildSuccess, got.Status)
	require.Equal(t, "build completed successfully", got.BuildLog)
}

func TestBuildRepository_DeleteNonLatest(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	builds := []domain.DockerBuild{
		{ID: "d1", DockerfileSlug: "go-agent", Version: "v1.0", Status: domain.BuildSuccess, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: "d2", DockerfileSlug: "go-agent", Version: "v1.1", Status: domain.BuildSuccess, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "d3", DockerfileSlug: "go-agent", Version: "v1.2", Status: domain.BuildSuccess, CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "d4", DockerfileSlug: "go-agent", Version: "v1.3", Status: domain.BuildSuccess, CreatedAt: now},
	}
	for _, b := range builds {
		require.NoError(t, repo.Create(ctx, b))
	}

	count, err := repo.DeleteNonLatest(ctx, "go-agent")
	require.NoError(t, err)
	require.Equal(t, 3, count, "must delete 3 non-latest builds")

	remaining, err := repo.ListByDockerfile(ctx, "go-agent")
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	require.Equal(t, domain.BuildID("d4"), remaining[0].ID)
}
