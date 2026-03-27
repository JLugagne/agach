package security_test

import (
	"github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents/agentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects/projectstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks/taskstest"
	"github.com/sirupsen/logrus"
)

// setupTestApp creates a test app with all mocked repositories.
func setupTestApp() (*app.App, *projectstest.MockProjectRepository, *agentstest.MockRoleRepository, *taskstest.MockTaskRepository, *columnstest.MockColumnRepository, *commentstest.MockCommentRepository, *dependenciestest.MockDependencyRepository) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	mockProjects := &projectstest.MockProjectRepository{}
	mockRoles := &agentstest.MockRoleRepository{}
	mockTasks := &taskstest.MockTaskRepository{}
	mockColumns := &columnstest.MockColumnRepository{}
	mockComments := &commentstest.MockCommentRepository{}
	mockDependencies := &dependenciestest.MockDependencyRepository{}

	a := app.NewApp(app.Config{
		Projects:     mockProjects,
		Agents:       mockRoles,
		Tasks:        mockTasks,
		Columns:      mockColumns,
		Comments:     mockComments,
		Dependencies: mockDependencies,
		Logger:       logger,
	})

	return a, mockProjects, mockRoles, mockTasks, mockColumns, mockComments, mockDependencies
}

// newSecurityApp creates a test app from explicit mock instances.
func newSecurityApp(
	projects *projectstest.MockProjectRepository,
	roles *agentstest.MockRoleRepository,
	tasks *taskstest.MockTaskRepository,
	columns *columnstest.MockColumnRepository,
	comments *commentstest.MockCommentRepository,
	deps *dependenciestest.MockDependencyRepository,
) *app.App {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return app.NewApp(app.Config{
		Projects:     projects,
		Agents:       roles,
		Tasks:        tasks,
		Columns:      columns,
		Comments:     comments,
		Dependencies: deps,
		Logger:       logger,
	})
}

// makeColumns returns a minimal set of columns.
func makeColumns() map[domain.ColumnSlug]*domain.Column {
	return map[domain.ColumnSlug]*domain.Column{
		domain.ColumnBacklog: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnBacklog,
			Name:     "Backlog",
			Position: 0,
		},
		domain.ColumnTodo: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnTodo,
			Name:     "Todo",
			Position: 1,
		},
		domain.ColumnInProgress: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnInProgress,
			Name:     "In Progress",
			Position: 2,
		},
		domain.ColumnDone: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnDone,
			Name:     "Done",
			Position: 3,
		},
		domain.ColumnBlocked: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnBlocked,
			Name:     "Blocked",
			Position: 4,
		},
	}
}
