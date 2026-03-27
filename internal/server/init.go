package server

import (
	"encoding/json"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/JLugagne/agach-mcp/internal/pkg/sse"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/queries"
	"github.com/JLugagne/agach-mcp/internal/server/outbound/pg"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Config holds the configuration for the Kanban system
type Config struct {
	Pool             *pgxpool.Pool
	Logger           *logrus.Logger
	AuthQueries      AuthQueries
	DataDir          string // Directory for storing chat session JSONL files
	ResourceManifest *ResourceManifest
	// WSRouter is the router on which to register the /ws endpoint.
	// Use a router without auth middleware, since browsers cannot send
	// Authorization headers on WebSocket connections; auth is done via
	// the ?token= query parameter instead. If nil, router is used.
	WSRouter *mux.Router
}

// InitHTTP initializes the Kanban system with HTTP REST API and WebSocket
func InitHTTP(cfg Config, router *mux.Router) (*websocket.Hub, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.Info("Initializing HTTP server system")

	// Initialize PostgreSQL repositories
	repos, err := pg.NewRepositories(cfg.Pool)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize repositories")
		return nil, err
	}

	logger.Info("Repositories initialized successfully")

	// Initialize app layer with repositories
	appInstance := app.NewApp(app.Config{
		Projects:      repos.Projects,
		Agents:        repos.Agents,
		Features:      repos.Features,
		Tasks:         repos.Tasks,
		Columns:       repos.Columns,
		Comments:      repos.Comments,
		Dependencies:  repos.Dependencies,
		ToolUsage:     repos.ToolUsage,
		Skills:        repos.Skills,
		Dockerfiles:   repos.Dockerfiles,
		Notifications: repos.Notifications,
		Specialized:   repos.SpecializedAgents,
		ProjectAccess: repos.ProjectAccess,
		Chats:         app.NewChatService(repos.Chats),
		Logger:        logger,
	})

	logger.Info("App layer initialized successfully")

	// Initialize WebSocket hub
	hub := websocket.NewHub(logger)
	go hub.Run()

	relayHandler := hub.NewRelayHandler()
	for _, msgType := range []string{
		daemonws.TypeDockerList,
		daemonws.TypeDockerRebuild,
		daemonws.TypeDockerLogs,
		daemonws.TypeDockerPrune,
		daemonws.TypeBuildEvent,
		daemonws.TypePruneEvent,
		daemonws.TypeError,
		daemonws.TypeChatStart,
		daemonws.TypeChatMessage,
		daemonws.TypeChatUserMsg,
		daemonws.TypeChatEnd,
		daemonws.TypeChatError,
		daemonws.TypeChatStats,
		daemonws.TypeChatPing,
		daemonws.TypeChatTTLWarning,
	} {
		hub.RegisterHandler(msgType, relayHandler)
	}

	logger.Info("WebSocket hub initialized")

	// Initialize SSE hub
	sseHub := sse.NewHub(logger)

	// Initialize controller
	ctrl := controller.NewController(logger)

	// Apply RateLimit middleware to the server router to limit resource creation rates.
	router.Use(middleware.RateLimit)

	// Register routes
	chatService := appInstance.ChatService()
	commands.NewRouter(router, appInstance, ctrl, hub, sseHub, cfg.DataDir, chatService)
	queries.NewRouter(router, appInstance, ctrl, sseHub, cfg.DataDir, chatService, cfg.AuthQueries)

	// Register resource download routes
	if cfg.ResourceManifest != nil {
		cfg.ResourceManifest.RegisterRoutes(router)
	}

	wsRouter := router
	if cfg.WSRouter != nil {
		wsRouter = cfg.WSRouter
	}
	var manifestJSON json.RawMessage
	if cfg.ResourceManifest != nil {
		manifestJSON, _ = json.Marshal(cfg.ResourceManifest.Entries())
	}
	wsHandler := commands.NewWSHandler(cfg.AuthQueries, hub, logger, manifestJSON)
	wsRouter.Handle("/ws", wsHandler).Methods("GET")

	logger.Info("REST API and WebSocket initialized successfully")

	return hub, nil
}
