package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects/projectstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/roles/rolestest"
	taskstest "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks/taskstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/toolusage/toolusagetest"
	"github.com/JLugagne/agach-mcp/internal/kanban/outbound/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a new SQLite repository for testing in a temporary directory
func setupTestRepo(t *testing.T) (*sqlite.Repositories, string) {
	t.Helper()

	// Create a temporary directory for test databases
	tempDir, err := os.MkdirTemp("", "kanban-test-*")
	require.NoError(t, err, "Failed to create temp directory")

	repo, err := sqlite.NewRepositories(tempDir)
	require.NoError(t, err, "Failed to create repository")

	return repo, tempDir
}

// cleanupTestRepo cleans up the test repository and temporary directory
func cleanupTestRepo(t *testing.T, repo *sqlite.Repositories, tempDir string) {
	t.Helper()

	if repo != nil {
		repo.Close()
	}

	if tempDir != "" {
		os.RemoveAll(tempDir)
	}
}

// setupTestProject creates a test project and returns its ID and column IDs
func setupTestProject(t *testing.T, repo *sqlite.Repositories) (domain.ProjectID, domain.ColumnID, domain.ColumnID, domain.ColumnID) {
	t.Helper()

	ctx := context.Background()

	// Create a test project
	projectID := domain.NewProjectID()
	project := domain.Project{
		ID:          projectID,
		Name:        "Test Project",
		Description: "Test project for contract testing",
	}

	err := repo.Projects.Create(ctx, project)
	require.NoError(t, err, "Failed to create test project")

	// Get column IDs from the project database
	columns, err := repo.Columns.List(ctx, projectID)
	require.NoError(t, err, "Failed to list columns")
	require.Len(t, columns, 4, "Should have 4 columns")

	var todoColumnID, inProgressColumnID, doneColumnID domain.ColumnID
	for _, col := range columns {
		switch col.Slug {
		case domain.ColumnTodo:
			todoColumnID = col.ID
		case domain.ColumnInProgress:
			inProgressColumnID = col.ID
		case domain.ColumnDone:
			doneColumnID = col.ID
		}
	}

	require.NotEmpty(t, todoColumnID, "Todo column should exist")
	require.NotEmpty(t, inProgressColumnID, "In Progress column should exist")
	require.NotEmpty(t, doneColumnID, "Done column should exist")

	return projectID, todoColumnID, inProgressColumnID, doneColumnID
}

func TestSQLiteProjectRepository(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	projectstest.ProjectsContractTesting(t, repo.Projects)
}

func TestSQLiteRoleRepository(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	rolestest.RolesContractTesting(t, repo.Roles)
}

func TestSQLiteColumnRepository(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	// Create a test project for column tests
	projectID, _, _, _ := setupTestProject(t, repo)

	columnstest.ColumnsContractTesting(t, repo.Columns, projectID)
}

func TestSQLiteTaskRepository(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	// Create a test project for task tests
	projectID, todoColumnID, inProgressColumnID, doneColumnID := setupTestProject(t, repo)

	taskstest.TasksContractTesting(t, repo.Tasks, projectID, todoColumnID, inProgressColumnID, doneColumnID)
}

func TestSQLiteCommentRepository(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	// Create a test project for comment tests
	projectID, todoColumnID, _, _ := setupTestProject(t, repo)

	// Pass task repository and column ID so contract tests can create their own tasks
	commentstest.CommentsContractTesting(t, repo.Comments, projectID, repo.Tasks, todoColumnID)
}

func TestSQLiteDependencyRepository(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	// Create a test project for dependency tests
	projectID, todoColumnID, _, _ := setupTestProject(t, repo)

	dependenciestest.DependenciesContractTesting(t, repo.Dependencies, projectID, repo.Tasks, todoColumnID)
}

// setupTestProjectWithAllColumns creates a test project and returns its ID and all 4 column IDs.
func setupTestProjectWithAllColumns(t *testing.T, repo *sqlite.Repositories) (domain.ProjectID, domain.ColumnID, domain.ColumnID, domain.ColumnID, domain.ColumnID) {
	t.Helper()

	ctx := context.Background()

	projectID := domain.NewProjectID()
	project := domain.Project{
		ID:          projectID,
		Name:        "Test Project With All Columns",
		Description: "Test project for summary testing",
	}

	err := repo.Projects.Create(ctx, project)
	require.NoError(t, err, "Failed to create test project")

	columns, err := repo.Columns.List(ctx, projectID)
	require.NoError(t, err, "Failed to list columns")
	require.Len(t, columns, 4, "Should have 4 columns")

	var todoColID, inProgressColID, doneColID, blockedColID domain.ColumnID
	for _, col := range columns {
		switch col.Slug {
		case domain.ColumnTodo:
			todoColID = col.ID
		case domain.ColumnInProgress:
			inProgressColID = col.ID
		case domain.ColumnDone:
			doneColID = col.ID
		case domain.ColumnBlocked:
			blockedColID = col.ID
		}
	}

	require.NotEmpty(t, todoColID, "Todo column should exist")
	require.NotEmpty(t, inProgressColID, "In Progress column should exist")
	require.NotEmpty(t, doneColID, "Done column should exist")
	require.NotEmpty(t, blockedColID, "Blocked column should exist")

	return projectID, todoColID, inProgressColID, doneColID, blockedColID
}

// newTestTask is a helper that builds a minimal valid domain.Task for a given column.
func newTestTask(columnID domain.ColumnID, title string) domain.Task {
	return domain.Task{
		ID:            domain.NewTaskID(),
		ColumnID:      columnID,
		Title:         title,
		Summary:       title + " summary",
		Priority:      domain.PriorityMedium,
		PriorityScore: domain.PriorityMedium.Score(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestGetSummaryEmptyProject(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	ctx := context.Background()
	projectID, _, _, _, _ := setupTestProjectWithAllColumns(t, repo)

	summary, err := repo.Projects.GetSummary(ctx, projectID)
	require.NoError(t, err, "GetSummary should succeed for empty project")
	require.NotNil(t, summary, "Summary must not be nil")
	assert.Equal(t, 0, summary.TodoCount, "TodoCount should be 0 for empty project")
	assert.Equal(t, 0, summary.InProgressCount, "InProgressCount should be 0 for empty project")
	assert.Equal(t, 0, summary.DoneCount, "DoneCount should be 0 for empty project")
	assert.Equal(t, 0, summary.BlockedCount, "BlockedCount should be 0 for empty project")
}

func TestGetSummaryWithTasksInAllColumns(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	ctx := context.Background()
	projectID, todoColID, inProgressColID, doneColID, blockedColID := setupTestProjectWithAllColumns(t, repo)

	// Create 2 tasks in todo, 3 in in_progress, 1 in done, 2 in blocked.
	tasks := []domain.Task{
		newTestTask(todoColID, "Todo Task 1"),
		newTestTask(todoColID, "Todo Task 2"),
		newTestTask(inProgressColID, "InProgress Task 1"),
		newTestTask(inProgressColID, "InProgress Task 2"),
		newTestTask(inProgressColID, "InProgress Task 3"),
		newTestTask(doneColID, "Done Task 1"),
		newTestTask(blockedColID, "Blocked Task 1"),
		newTestTask(blockedColID, "Blocked Task 2"),
	}

	for _, task := range tasks {
		err := repo.Tasks.Create(ctx, projectID, task)
		require.NoError(t, err, "Create task should succeed")
	}

	summary, err := repo.Projects.GetSummary(ctx, projectID)
	require.NoError(t, err, "GetSummary should succeed")
	require.NotNil(t, summary, "Summary must not be nil")
	assert.Equal(t, 2, summary.TodoCount, "TodoCount should be 2")
	assert.Equal(t, 3, summary.InProgressCount, "InProgressCount should be 3")
	assert.Equal(t, 1, summary.DoneCount, "DoneCount should be 1")
	assert.Equal(t, 2, summary.BlockedCount, "BlockedCount should be 2")
}

func TestGetSummaryNonExistentProject(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	ctx := context.Background()
	nonExistentID := domain.NewProjectID()

	_, err := repo.Projects.GetSummary(ctx, nonExistentID)
	assert.Error(t, err, "GetSummary should return error for non-existent project")
}

func TestGetSummaryUpdatesAfterTaskMove(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	ctx := context.Background()
	projectID, todoColID, inProgressColID, _, _ := setupTestProjectWithAllColumns(t, repo)

	task := newTestTask(todoColID, "Movable Task")
	err := repo.Tasks.Create(ctx, projectID, task)
	require.NoError(t, err, "Create task should succeed")

	// Verify initial state: 1 todo, 0 in_progress.
	summary, err := repo.Projects.GetSummary(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.TodoCount)
	assert.Equal(t, 0, summary.InProgressCount)

	// Move task to in_progress.
	err = repo.Tasks.Move(ctx, projectID, task.ID, inProgressColID)
	require.NoError(t, err, "Move should succeed")

	// Verify updated state: 0 todo, 1 in_progress.
	summary, err = repo.Projects.GetSummary(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, 0, summary.TodoCount)
	assert.Equal(t, 1, summary.InProgressCount)
}

func TestSQLiteToolUsageRepository(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	projectID, _, _, _ := setupTestProject(t, repo)

	toolusagetest.ToolUsageContractTesting(t, repo.ToolUsage, projectID)
}

func TestRepositoryCreatesGlobalDatabase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kanban-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repo, err := sqlite.NewRepositories(tempDir)
	require.NoError(t, err)
	defer repo.Close()

	// Check that kanban.db was created
	globalDBPath := filepath.Join(tempDir, "kanban.db")
	_, err = os.Stat(globalDBPath)
	require.NoError(t, err, "Global database should exist")
}

func TestRepositoryCreatesProjectDatabase(t *testing.T) {
	repo, tempDir := setupTestRepo(t)
	defer cleanupTestRepo(t, repo, tempDir)

	ctx := context.Background()

	// Create a project
	projectID := domain.NewProjectID()
	project := domain.Project{
		ID:          projectID,
		Name:        "Test Project",
		Description: "Test project",
		WorkDir:     "/home/user/myproject",
	}

	err := repo.Projects.Create(ctx, project)
	require.NoError(t, err)

	// List columns to trigger project database creation
	columns, err := repo.Columns.List(ctx, projectID)
	require.NoError(t, err)
	require.Len(t, columns, 4, "Should have 4 columns")

	// Check that project database was created using project ID
	projectDBPath := filepath.Join(tempDir, sqlite.ProjectDBName(project.ID))
	_, err = os.Stat(projectDBPath)
	require.NoError(t, err, "Project database should exist")
}
