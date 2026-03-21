package pg_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects/projectstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/roles/rolestest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks/taskstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/toolusage/toolusagetest"
	"github.com/JLugagne/agach-mcp/internal/kanban/outbound/pg"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:17",
		tcpostgres.WithDatabase("kanban_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

type testRepos struct {
	*pg.Repositories
	projectID          domain.ProjectID
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
	inProgressCol := domain.Column{ID: domain.NewColumnID(), Slug: domain.ColumnInProgress, Name: "In Progress", Position: 1, WIPLimit: 3}
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

	return &testRepos{
		Repositories:       repos,
		projectID:          projectID,
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
	rolestest.RolesContractTesting(t, repos.Roles)
}

func TestTaskRepository_Contract(t *testing.T) {
	r := setupRepos(t)
	taskstest.TasksContractTesting(t, r.Tasks, r.projectID, r.todoColumnID, r.inProgressColumnID, r.doneColumnID)
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
