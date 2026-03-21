package kanban

import (
	"context"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/app"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/mcp"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/queries"
	"github.com/JLugagne/agach-mcp/internal/kanban/outbound/pg"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/sse"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
	gorillaws "github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Config holds the configuration for the Kanban system
type Config struct {
	Pool   *pgxpool.Pool
	Logger *logrus.Logger
}

// RunMCPStdio initializes the Kanban system and runs the MCP server over stdio.
// This blocks until the connection is closed or ctx is cancelled.
func RunMCPStdio(ctx context.Context, cfg Config) error {
	logger := cfg.Logger
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	// Redirect logrus output to stderr so it doesn't interfere with stdio MCP
	logger.SetOutput(logrus.StandardLogger().Out)

	logger.Info("Initializing Kanban MCP stdio server")

	repos, err := pg.NewRepositories(cfg.Pool)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize repositories")
		return err
	}

	appInstance := app.NewApp(app.Config{
		Projects:     repos.Projects,
		Roles:        repos.Roles,
		Tasks:        repos.Tasks,
		Columns:      repos.Columns,
		Comments:     repos.Comments,
		Dependencies: repos.Dependencies,
		ToolUsage:    repos.ToolUsage,
		Logger:       logger,
	})

	mcpServer, err := mcp.NewServer(appInstance, appInstance, nil, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize MCP server")
		return err
	}

	logger.Info("MCP stdio server starting")
	return mcpServer.Run(ctx)
}

// InitKanbanHTTP initializes the Kanban system with HTTP REST API and WebSocket
func InitKanbanHTTP(cfg Config, router *mux.Router) (*websocket.Hub, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.Info("Initializing Kanban HTTP system")

	// Initialize PostgreSQL repositories
	repos, err := pg.NewRepositories(cfg.Pool)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize repositories")
		return nil, err
	}

	logger.Info("Repositories initialized successfully")

	// Initialize app layer with repositories
	appInstance := app.NewApp(app.Config{
		Projects:     repos.Projects,
		Roles:        repos.Roles,
		Tasks:        repos.Tasks,
		Columns:      repos.Columns,
		Comments:     repos.Comments,
		Dependencies: repos.Dependencies,
		ToolUsage:    repos.ToolUsage,
		Logger:       logger,
	})

	logger.Info("App layer initialized successfully")

	// Initialize WebSocket hub
	hub := websocket.NewHub(logger)
	go hub.Run()

	logger.Info("WebSocket hub initialized")

	// Initialize SSE hub
	sseHub := sse.NewHub()

	// Initialize controller
	ctrl := controller.NewController(logger)

	// Initialize command handlers
	projectCommands := commands.NewProjectCommandsHandler(appInstance, ctrl, hub)
	roleCommands := commands.NewRoleCommandsHandler(appInstance, ctrl, hub)
	taskCommands := commands.NewTaskCommandsHandler(appInstance, ctrl, hub, sseHub)
	commentCommands := commands.NewCommentCommandsHandlerWithQueries(appInstance, appInstance, ctrl, hub)
	imageCommands := commands.NewImageCommandsHandler(appInstance, ctrl)
	seenCommands := commands.NewSeenCommandsHandler(appInstance, ctrl, hub)
	columnCommands := commands.NewColumnCommandsHandler(appInstance, ctrl)
	projectRoleCommands := commands.NewProjectRoleCommandsHandler(appInstance, appInstance, ctrl)

	// Initialize query handlers
	projectQueries := queries.NewProjectQueriesHandler(appInstance, ctrl)
	roleQueries := queries.NewRoleQueriesHandler(appInstance, ctrl)
	taskQueries := queries.NewTaskQueriesHandler(appInstance, ctrl)
	commentQueries := queries.NewCommentQueriesHandler(appInstance, ctrl)
	dependencyQueries := queries.NewDependencyQueriesHandler(appInstance, ctrl)
	toolUsageQueries := queries.NewToolUsageQueriesHandler(appInstance, ctrl)
	timelineQueries := queries.NewTimelineQueriesHandler(appInstance, ctrl)
	projectRoleQueries := queries.NewProjectRoleQueriesHandler(appInstance, ctrl)
	coldStartStatsQueries := queries.NewColdStartStatsQueriesHandler(appInstance, ctrl)
	sseHandler := queries.NewSSEHandler(sseHub)

	// Register routes
	projectCommands.RegisterRoutes(router)
	roleCommands.RegisterRoutes(router)
	taskCommands.RegisterRoutes(router)
	commentCommands.RegisterRoutes(router)
	imageCommands.RegisterRoutes(router)
	seenCommands.RegisterRoutes(router)
	columnCommands.RegisterRoutes(router)
	projectRoleCommands.RegisterRoutes(router)
	projectQueries.RegisterRoutes(router)
	roleQueries.RegisterRoutes(router)
	taskQueries.RegisterRoutes(router)
	commentQueries.RegisterRoutes(router)
	dependencyQueries.RegisterRoutes(router)
	toolUsageQueries.RegisterRoutes(router)
	timelineQueries.RegisterRoutes(router)
	projectRoleQueries.RegisterRoutes(router)
	coldStartStatsQueries.RegisterRoutes(router)
	sseHandler.RegisterRoutes(router)

	// WebSocket endpoint
	upgrader := gorillaws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.WithError(err).Error("Failed to upgrade WebSocket")
			return
		}
		hub.ServeWS(conn)
	}).Methods("GET")

	logger.Info("REST API and WebSocket initialized successfully")

	return hub, nil
}

// InitKanbanMCP initializes the Kanban MCP system and mounts the SSE handler on the provided router.
// If hub is provided, the MCP server will broadcast events to it; otherwise a new hub is created.
func InitKanbanMCP(cfg Config, router *mux.Router, hub *websocket.Hub) error {
	logger := cfg.Logger
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.Info("Initializing Kanban MCP SSE system")

	repos, err := pg.NewRepositories(cfg.Pool)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize repositories")
		return err
	}

	appInstance := app.NewApp(app.Config{
		Projects:     repos.Projects,
		Roles:        repos.Roles,
		Tasks:        repos.Tasks,
		Columns:      repos.Columns,
		Comments:     repos.Comments,
		Dependencies: repos.Dependencies,
		ToolUsage:    repos.ToolUsage,
		Logger:       logger,
	})

	mcpServer, err := mcp.NewServer(appInstance, appInstance, hub, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize MCP server")
		return err
	}

	router.PathPrefix("/mcp").Handler(mcpServer.HTTPHandler())

	logger.Info("MCP server mounted at /mcp (streamable HTTP)")
	return nil
}
