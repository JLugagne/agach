package app

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies"
	dockerfilesrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dockerfiles"
	featuresrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/features"
	notificationsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	agentsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents"
	skillsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills"
	specializedrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/specialized"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/toolusage"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/sirupsen/logrus"
)

// App implements both Commands and Queries service interfaces via embedded service structs.
type App struct {
	*ProjectService
	*TaskService
	*AgentService
	*CommentService
	*FeatureService
	*SkillService
	*DockerfileService
	*NotificationService
	*StatsService
	*ColumnService
	*DependencyService
	*SpecializedAgentService
	chats *ChatService
}

// Config holds the dependencies for the App
type Config struct {
	Projects      projects.ProjectRepository
	Agents        agentsrepo.AgentRepository
	Features      featuresrepo.FeatureRepository
	Tasks         tasks.TaskRepository
	Columns       columns.ColumnRepository
	Comments      comments.CommentRepository
	Dependencies  dependencies.DependencyRepository
	ToolUsage     toolusage.ToolUsageRepository
	Skills        skillsrepo.SkillRepository
	Dockerfiles   dockerfilesrepo.DockerfileRepository
	Notifications notificationsrepo.NotificationRepository
	Specialized   specializedrepo.SpecializedAgentRepository
	Chats         *ChatService
	Logger        *logrus.Logger
}

// NewApp creates a new App instance
func NewApp(cfg Config) *App {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	return &App{
		ProjectService:          newProjectService(cfg.Projects, cfg.Agents, cfg.Logger),
		TaskService:             newTaskService(cfg.Tasks, cfg.Columns, cfg.Dependencies, cfg.Features, cfg.Projects, cfg.Comments, cfg.Logger),
		AgentService:            newAgentService(cfg.Agents, cfg.Specialized, cfg.Projects, cfg.Tasks, cfg.Skills, cfg.Logger),
		CommentService:          newCommentService(cfg.Comments, cfg.Tasks, cfg.Logger),
		FeatureService:          newFeatureService(cfg.Features, cfg.Projects, cfg.Logger),
		SkillService:            newSkillService(cfg.Skills, cfg.Logger),
		DockerfileService:       newDockerfileService(cfg.Dockerfiles, cfg.Projects, cfg.Logger),
		NotificationService:     newNotificationService(cfg.Notifications, cfg.Logger),
		StatsService:            newStatsService(cfg.ToolUsage, cfg.Tasks, cfg.Projects, cfg.Logger),
		ColumnService:           newColumnService(cfg.Columns, cfg.Projects, cfg.Logger),
		DependencyService:       newDependencyService(cfg.Dependencies, cfg.Tasks, cfg.Logger),
		SpecializedAgentService: newSpecializedAgentService(cfg.Agents, cfg.Skills, cfg.Specialized, cfg.Logger),
		chats:                   cfg.Chats,
	}
}

// Verify that App implements both service interfaces
var (
	_ service.Commands = (*App)(nil)
	_ service.Queries  = (*App)(nil)
)

// ChatService returns the chat service instance
func (a *App) ChatService() *ChatService {
	return a.chats
}
