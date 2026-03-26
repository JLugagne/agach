// Package qaseed provides deterministic test data for Playwright E2E tests.
// It wipes existing data and inserts a fixed, well-known dataset so tests can
// rely on stable IDs, titles, and states.
package qaseed

import (
	"context"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/outbound/pg"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Result holds the IDs of every entity created during seeding so Playwright
// tests can reference them directly without hard-coded magic strings.
type Result struct {
	// Projects
	MainProjectID    domain.ProjectID
	FeatureProjectID domain.ProjectID

	// Roles
	BackendRoleID  domain.AgentID
	FrontendRoleID domain.AgentID
	QARoleID       domain.AgentID

	// Skills
	GoSkillID         domain.SkillID
	PlaywrightSkillID domain.SkillID

	// Dockerfiles
	DockerfileID domain.DockerfileID

	// Tasks (main project)
	TodoTaskID       domain.TaskID
	InProgressTaskID domain.TaskID
	BlockedTaskID    domain.TaskID
	DoneTaskID       domain.TaskID
	WontDoTaskID     domain.TaskID // in done column, wont_do_requested=1
	BacklogTaskID    domain.TaskID
	FeatureTaskID    domain.TaskID // task assigned to feature sub-project

	// Task with dependency
	DepParentTaskID domain.TaskID
	DepChildTaskID  domain.TaskID
}

// Run wipes all projects/roles and re-seeds the database with a known dataset.
// It uses the service layer (app.App) so all business rules are respected.
func Run(ctx context.Context, pool *pgxpool.Pool, logger *logrus.Logger) (*Result, error) {
	repos, err := pg.NewRepositories(pool)
	if err != nil {
		return nil, fmt.Errorf("initialising repositories: %w", err)
	}

	svc := app.NewApp(app.Config{
		Projects:     repos.Projects,
		Agents:        repos.Agents,
		Tasks:        repos.Tasks,
		Columns:      repos.Columns,
		Comments:     repos.Comments,
		Dependencies: repos.Dependencies,
		ToolUsage:    repos.ToolUsage,
		Skills:       repos.Skills,
		Dockerfiles:  repos.Dockerfiles,
		Logger:       logger,
	})

	if err := wipe(ctx, pool); err != nil {
		return nil, fmt.Errorf("wiping existing data: %w", err)
	}

	return seed(ctx, svc, logger)
}

// wipe truncates all user-created data in dependency order.
func wipe(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		// Leaf tables first (no dependants)
		"task_dependencies",
		"comments",
		"tool_usage",
		"agent_skills",
		// Tasks reference columns, projects, and features (projects)
		"tasks",
		"columns",
		// Join tables referencing projects + roles
		"project_roles",
		"project_agents",
		// Projects reference dockerfiles (FK SET NULL, but wipe anyway)
		"projects",
		// Top-level entities
		"skills",
		"dockerfiles",
		"roles",
	}
	for _, t := range tables {
		if _, err := pool.Exec(ctx, "DELETE FROM "+t); err != nil {
			return fmt.Errorf("truncating %s: %w", t, err)
		}
	}
	return nil
}

func seed(ctx context.Context, svc service.Commands, logger *logrus.Logger) (*Result, error) {
	res := &Result{}

	// ------------------------------------------------------------------ Roles
	backendRole, err := svc.CreateAgent(ctx,
		"backend", "Backend Engineer", "⚙️", "#3B82F6",
		"Implements server-side logic, APIs, and database access.",
		"Focus on correctness, performance, and test coverage.",
		"", "", "",
		[]string{"Go", "PostgreSQL", "REST"}, 1,
	)
	if err != nil {
		return nil, fmt.Errorf("create backend role: %w", err)
	}
	res.BackendRoleID = backendRole.ID
	logger.WithField("id", backendRole.ID).Info("qa-seed: created backend role")

	frontendRole, err := svc.CreateAgent(ctx,
		"frontend", "Frontend Engineer", "🖥️", "#8B5CF6",
		"Builds user interfaces and integrates with HTTP APIs.",
		"Write accessible, responsive components.", "",
		"", "",
		[]string{"TypeScript", "React", "Tailwind"}, 2,
	)
	if err != nil {
		return nil, fmt.Errorf("create frontend role: %w", err)
	}
	res.FrontendRoleID = frontendRole.ID
	logger.WithField("id", frontendRole.ID).Info("qa-seed: created frontend role")

	qaRole, err := svc.CreateAgent(ctx,
		"qa", "QA Engineer", "🧪", "#10B981",
		"Writes automated and manual tests to ensure quality.",
		"Prefer end-to-end coverage over unit tests.", "",
		"", "",
		[]string{"Playwright", "pytest"}, 3,
	)
	if err != nil {
		return nil, fmt.Errorf("create qa role: %w", err)
	}
	res.QARoleID = qaRole.ID
	logger.WithField("id", qaRole.ID).Info("qa-seed: created qa role")

	// ---------------------------------------------------------------- Skills
	goSkill, err := svc.CreateSkill(ctx,
		"go-development", "Go Development",
		"Best practices for writing Go services.",
		"Use table-driven tests. Prefer composition over inheritance.",
		"🔧", "#00ADD8", 1,
	)
	if err != nil {
		return nil, fmt.Errorf("create go skill: %w", err)
	}
	res.GoSkillID = goSkill.ID
	logger.WithField("id", goSkill.ID).Info("qa-seed: created go skill")

	playwrightSkill, err := svc.CreateSkill(ctx,
		"playwright-testing", "Playwright Testing",
		"End-to-end testing with Playwright.",
		"Always use data-qa attributes for selectors.",
		"🎭", "#2EAD33", 2,
	)
	if err != nil {
		return nil, fmt.Errorf("create playwright skill: %w", err)
	}
	res.PlaywrightSkillID = playwrightSkill.ID
	logger.WithField("id", playwrightSkill.ID).Info("qa-seed: created playwright skill")

	// Assign skills to agents
	if err := svc.AddSkillToAgent(ctx, "backend", "go-development"); err != nil {
		return nil, fmt.Errorf("assign go skill to backend: %w", err)
	}
	if err := svc.AddSkillToAgent(ctx, "qa", "playwright-testing"); err != nil {
		return nil, fmt.Errorf("assign playwright skill to qa: %w", err)
	}

	// ------------------------------------------------------------- Dockerfiles
	dockerfile, err := svc.CreateDockerfile(ctx,
		"go-service", "Go Service",
		"Standard Go service with PostgreSQL",
		"1.0.0",
		"FROM golang:1.24\nWORKDIR /app\nCOPY . .\nRUN go build -o /bin/app ./cmd/server\nCMD [\"/bin/app\"]",
		true, 1,
	)
	if err != nil {
		return nil, fmt.Errorf("create dockerfile: %w", err)
	}
	res.DockerfileID = dockerfile.ID
	logger.WithField("id", dockerfile.ID).Info("qa-seed: created dockerfile")

	// --------------------------------------------------------------- Projects
	mainProject, err := svc.CreateProject(ctx,
		"QA Test Project",
		"Seeded project for Playwright tests",
		"",
		"qa", "qa-seed",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create main project: %w", err)
	}
	res.MainProjectID = mainProject.ID
	logger.WithField("id", mainProject.ID).Info("qa-seed: created main project")

	featureProject, err := svc.CreateProject(ctx,
		"QA Feature Branch",
		"Sub-project used to test feature grouping",
		"",
		"qa", "qa-seed",
		&mainProject.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("create feature project: %w", err)
	}
	res.FeatureProjectID = featureProject.ID
	logger.WithField("id", featureProject.ID).Info("qa-seed: created feature project")

	// Assign agents to main project
	if err := svc.AssignAgentToProject(ctx, mainProject.ID, "backend"); err != nil {
		return nil, fmt.Errorf("assign backend to project: %w", err)
	}
	if err := svc.AssignAgentToProject(ctx, mainProject.ID, "frontend"); err != nil {
		return nil, fmt.Errorf("assign frontend to project: %w", err)
	}
	if err := svc.AssignAgentToProject(ctx, mainProject.ID, "qa"); err != nil {
		return nil, fmt.Errorf("assign qa to project: %w", err)
	}

	// Assign dockerfile to main project
	if err := svc.SetProjectDockerfile(ctx, mainProject.ID, dockerfile.ID); err != nil {
		return nil, fmt.Errorf("assign dockerfile to project: %w", err)
	}

	// ----------------------------------------------------------------- Tasks
	todoTask, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Todo task",
		"A task sitting in the todo column.",
		"This task has not been started yet.",
		domain.PriorityMedium,
		"backend", "qa-seed", "backend",
		[]string{}, []string{"qa", "todo"}, "S",
		false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create todo task: %w", err)
	}
	res.TodoTaskID = todoTask.ID
	logger.WithField("id", todoTask.ID).Info("qa-seed: created todo task")

	inProgressTask, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] In-progress task",
		"A task currently being worked on.",
		"Agent has started this task.",
		domain.PriorityHigh,
		"backend", "qa-seed", "backend",
		[]string{}, []string{"qa", "in-progress"}, "M",
		false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create in-progress task: %w", err)
	}
	res.InProgressTaskID = inProgressTask.ID
	if err := svc.StartTask(ctx, mainProject.ID, inProgressTask.ID); err != nil {
		return nil, fmt.Errorf("start in-progress task: %w", err)
	}
	logger.WithField("id", inProgressTask.ID).Info("qa-seed: created in-progress task")

	blockedTask, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Blocked task",
		"A task blocked on external dependency.",
		"Waiting for third-party API credentials.",
		domain.PriorityCritical,
		"backend", "qa-seed", "backend",
		[]string{}, []string{"qa", "blocked"}, "L",
		false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create blocked task: %w", err)
	}
	res.BlockedTaskID = blockedTask.ID
	// Move to in_progress first so BlockTask can act on it
	if err := svc.StartTask(ctx, mainProject.ID, blockedTask.ID); err != nil {
		return nil, fmt.Errorf("start blocked task: %w", err)
	}
	if err := svc.BlockTask(ctx, mainProject.ID, blockedTask.ID,
		"Waiting for third-party API credentials", "qa-seed"); err != nil {
		return nil, fmt.Errorf("block task: %w", err)
	}
	logger.WithField("id", blockedTask.ID).Info("qa-seed: created blocked task")

	doneTask, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Done task",
		"A completed task.",
		"All acceptance criteria were met.",
		domain.PriorityLow,
		"frontend", "qa-seed", "frontend",
		[]string{"internal/server/ux/src/pages/HomePage.tsx"}, []string{"qa", "done"}, "XS",
		false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create done task: %w", err)
	}
	res.DoneTaskID = doneTask.ID
	if err := svc.StartTask(ctx, mainProject.ID, doneTask.ID); err != nil {
		return nil, fmt.Errorf("start done task: %w", err)
	}
	if err := svc.CompleteTask(ctx, mainProject.ID, doneTask.ID,
		"Implemented homepage redesign per spec.",
		[]string{"internal/server/ux/src/pages/HomePage.tsx"},
		"qa-seed", nil,
	); err != nil {
		return nil, fmt.Errorf("complete done task: %w", err)
	}
	logger.WithField("id", doneTask.ID).Info("qa-seed: created done task")

	wontDoTask, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Won't-do task",
		"A task the team decided not to implement.",
		"Scope cut agreed with stakeholders.",
		domain.PriorityLow,
		"backend", "qa-seed", "backend",
		[]string{}, []string{"qa", "wontdo"}, "XL",
		false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create wont-do task: %w", err)
	}
	res.WontDoTaskID = wontDoTask.ID
	if err := svc.StartTask(ctx, mainProject.ID, wontDoTask.ID); err != nil {
		return nil, fmt.Errorf("start wont-do task: %w", err)
	}
	if err := svc.RequestWontDo(ctx, mainProject.ID, wontDoTask.ID,
		"Out of scope for current sprint", "qa-seed"); err != nil {
		return nil, fmt.Errorf("request wont-do: %w", err)
	}
	if err := svc.ApproveWontDo(ctx, mainProject.ID, wontDoTask.ID); err != nil {
		return nil, fmt.Errorf("approve wont-do: %w", err)
	}
	logger.WithField("id", wontDoTask.ID).Info("qa-seed: created wont-do task")

	// ------------------------------------------------------- Backlog task
	backlogTask, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Backlog task",
		"A task waiting in the backlog.",
		"This task is parked in the backlog for future sprint planning.",
		domain.PriorityLow,
		"backend", "qa-seed", "backend",
		[]string{}, []string{"qa", "backlog"}, "S",
		true, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create backlog task: %w", err)
	}
	res.BacklogTaskID = backlogTask.ID
	logger.WithField("id", backlogTask.ID).Info("qa-seed: created backlog task")

	// ----------------------------------------------- Feature task (sub-project)
	featureTask, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Feature task",
		"A task assigned to a feature branch.",
		"This task belongs to the QA Feature Branch sub-project.",
		domain.PriorityMedium,
		"frontend", "qa-seed", "frontend",
		[]string{}, []string{"qa", "feature"}, "M",
		false, func() *domain.FeatureID { fid := domain.FeatureID(featureProject.ID); return &fid }(),
	)
	if err != nil {
		return nil, fmt.Errorf("create feature task: %w", err)
	}
	res.FeatureTaskID = featureTask.ID
	logger.WithField("id", featureTask.ID).Info("qa-seed: created feature task")

	// ---------------------------------------------- Tasks with a dependency
	depParent, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Dependency parent",
		"Must be completed before the child task.",
		"Foundation work that unblocks subsequent tasks.",
		domain.PriorityHigh,
		"backend", "qa-seed", "backend",
		[]string{}, []string{"qa", "dependency"}, "M",
		false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create dep parent: %w", err)
	}
	res.DepParentTaskID = depParent.ID

	depChild, err := svc.CreateTask(ctx,
		mainProject.ID,
		"[QA] Dependency child",
		"Blocked until the parent task is done.",
		"Depends on the parent task being completed first.",
		domain.PriorityMedium,
		"backend", "qa-seed", "backend",
		[]string{}, []string{"qa", "dependency"}, "S",
		false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create dep child: %w", err)
	}
	res.DepChildTaskID = depChild.ID

	if err := svc.AddDependency(ctx, mainProject.ID, depChild.ID, depParent.ID); err != nil {
		return nil, fmt.Errorf("add dependency: %w", err)
	}
	logger.WithFields(logrus.Fields{
		"parent": depParent.ID,
		"child":  depChild.ID,
	}).Info("qa-seed: created dependency pair")

	// ------------------------------------------------------------ Comments
	if _, err := svc.CreateComment(ctx,
		mainProject.ID, todoTask.ID,
		"backend", "qa-seed", domain.AuthorTypeAgent,
		"This task is ready to be picked up.",
	); err != nil {
		return nil, fmt.Errorf("add comment to todo task: %w", err)
	}

	if _, err := svc.CreateComment(ctx,
		mainProject.ID, blockedTask.ID,
		"human", "QA Tester", domain.AuthorTypeHuman,
		"We are waiting on vendor to provide API keys. ETA end of week.",
	); err != nil {
		return nil, fmt.Errorf("add comment to blocked task: %w", err)
	}

	logger.Info("qa-seed: seeding complete")
	return res, nil
}
