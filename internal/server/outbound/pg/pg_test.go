package pg_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents/agentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications/notificationstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects/projectstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks/taskstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/toolusage/toolusagetest"
	"github.com/JLugagne/agach-mcp/internal/server/outbound/pg"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Shared container for all tests in this package — started once in TestMain.
var sharedConnStr string
var dbCounter atomic.Int64

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx,
		"postgres:17",
		tcpostgres.WithDatabase("server_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		log.Fatalf("failed to get connection string: %v", err)
	}
	sharedConnStr = connStr

	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

// newTestPool creates a fresh database within the shared container for test isolation.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	// Connect to the default database to create a per-test database.
	adminPool, err := pgxpool.New(ctx, sharedConnStr)
	require.NoError(t, err)

	dbName := fmt.Sprintf("test_%d", dbCounter.Add(1))
	_, err = adminPool.Exec(ctx, "CREATE DATABASE "+dbName)
	require.NoError(t, err)
	adminPool.Close()

	// Build connection string for the new database.
	// Replace the database name in the connection string.
	testConnStr := replaceDBName(sharedConnStr, dbName)

	pool, err := pgxpool.New(ctx, testConnStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

// replaceDBName swaps the database name in a postgres connection string.
func replaceDBName(connStr, newDB string) string {
	// connStr looks like: postgres://test:test@host:port/server_test?sslmode=disable
	// Replace /server_test with /newDB
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return connStr
	}
	cfg.ConnConfig.Database = newDB
	return cfg.ConnConfig.ConnString()
}

type testRepos struct {
	*pg.Repositories
	projectID          domain.ProjectID
	featureProjectID   domain.FeatureID
	todoColumnID       domain.ColumnID
	inProgressColumnID domain.ColumnID
	doneColumnID       domain.ColumnID
}

func setupRepos(t *testing.T) *testRepos {
	t.Helper()
	ctx := context.Background()

	pool := newTestPool(t)
	repos, err := pg.NewRepositories(pool)
	require.NoError(t, err)

	// Create a project
	projectID := domain.NewProjectID()
	err = repos.Projects.Create(ctx, domain.Project{
		ID:        projectID,
		Name:      "Test Project",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Ensure columns via EnsureBacklog and seeding
	todoCol := domain.Column{ID: domain.NewColumnID(), Slug: domain.ColumnTodo, Name: "To Do", Position: 0}
	inProgressCol := domain.Column{ID: domain.NewColumnID(), Slug: domain.ColumnInProgress, Name: "In Progress", Position: 1}
	doneCol := domain.Column{ID: domain.NewColumnID(), Slug: domain.ColumnDone, Name: "Done", Position: 2}

	// Use EnsureBacklog or seed columns directly — columns are typically created by migration.
	// For testing, we create them directly if the column repo supports it, otherwise use EnsureBacklog.
	_ = todoCol
	_ = inProgressCol
	_ = doneCol

	// Get columns created by migrations/project init
	cols, err := repos.Columns.List(ctx, projectID)
	require.NoError(t, err)

	var todoID, inProgressID, doneID domain.ColumnID
	for _, c := range cols {
		switch c.Slug {
		case domain.ColumnTodo:
			todoID = c.ID
		case domain.ColumnInProgress:
			inProgressID = c.ID
		case domain.ColumnDone:
			doneID = c.ID
		}
	}

	// Create a feature to use as featureProjectID in contract tests
	featureID := domain.NewFeatureID()
	err = repos.Features.Create(ctx, domain.Feature{
		ID:        featureID,
		ProjectID: projectID,
		Name:      "Test Feature",
		Status:    domain.FeatureStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	return &testRepos{
		Repositories:       repos,
		projectID:          projectID,
		featureProjectID:   featureID,
		todoColumnID:       todoID,
		inProgressColumnID: inProgressID,
		doneColumnID:       doneID,
	}
}

func TestProjectRepository_Contract(t *testing.T) {
	pool := newTestPool(t)
	repos, err := pg.NewRepositories(pool)
	require.NoError(t, err)
	projectstest.ProjectsContractTesting(t, repos.Projects)
}

func TestRoleRepository_Contract(t *testing.T) {
	pool := newTestPool(t)
	repos, err := pg.NewRepositories(pool)
	require.NoError(t, err)
	agentstest.RolesContractTesting(t, repos.Agents)
}

func TestTaskRepository_Contract(t *testing.T) {
	r := setupRepos(t)
	taskstest.TasksContractTesting(t, r.Tasks, r.projectID, r.todoColumnID, r.inProgressColumnID, r.doneColumnID, r.featureProjectID)
}

func TestColumnRepository_Contract(t *testing.T) {
	r := setupRepos(t)
	columnstest.ColumnsContractTesting(t, r.Columns, r.projectID)
}

func TestCommentRepository_Contract(t *testing.T) {
	r := setupRepos(t)
	commentstest.CommentsContractTesting(t, r.Comments, r.projectID, r.Tasks, r.todoColumnID)
}

func TestDependencyRepository_Contract(t *testing.T) {
	r := setupRepos(t)
	dependenciestest.DependenciesContractTesting(t, r.Dependencies, r.projectID, r.Tasks, r.todoColumnID)
}

func TestToolUsageRepository_Contract(t *testing.T) {
	r := setupRepos(t)
	toolusagetest.ToolUsageContractTesting(t, r.ToolUsage, r.projectID)
}

func TestNotificationRepository_Contract(t *testing.T) {
	r := setupRepos(t)
	notificationstest.NotificationsContractTesting(t, r.Notifications, r.projectID)
}

func TestProjectFeaturesRepository(t *testing.T) {
	pool := newTestPool(t)
	repos, err := pg.NewRepositories(pool)
	require.NoError(t, err)

	ctx := context.Background()

	parentProject := domain.Project{
		ID:   domain.NewProjectID(),
		Name: "Test Parent Project",
	}
	require.NoError(t, repos.Projects.Create(ctx, parentProject))

	createTaskInColumn := func(t *testing.T, projectID domain.ProjectID, columnSlug domain.ColumnSlug) {
		t.Helper()
		cols, err := repos.Columns.List(ctx, projectID)
		require.NoError(t, err)

		var col *domain.Column
		for i := range cols {
			if cols[i].Slug == columnSlug {
				col = &cols[i]
				break
			}
		}
		require.NotNil(t, col, "column %s must exist for project %s", columnSlug, projectID)

		task := domain.Task{
			ID:            domain.NewTaskID(),
			ColumnID:      col.ID,
			Title:         "Test Task",
			Summary:       "Test task for feature active filter",
			Priority:      domain.PriorityMedium,
			PriorityScore: domain.PriorityMedium.Score(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		require.NoError(t, repos.Tasks.Create(ctx, projectID, task))
	}

	projectstest.ProjectFeaturesContractTesting(t, repos.Projects, parentProject.ID, createTaskInColumn)
}
