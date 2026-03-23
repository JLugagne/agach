package kanban

import (
	"net/http"

	identityservice "github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/app"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/commands"
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
	Pool        *pgxpool.Pool
	Logger      *logrus.Logger
	AuthQueries identityservice.AuthQueries
	// WSRouter is the router on which to register the /ws endpoint.
	// Use a router without auth middleware, since browsers cannot send
	// Authorization headers on WebSocket connections; auth is done via
	// the ?token= query parameter instead. If nil, router is used.
	WSRouter *mux.Router
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
		Agents:        repos.Agents,
		Features:     repos.Features,
		Tasks:        repos.Tasks,
		Columns:      repos.Columns,
		Comments:     repos.Comments,
		Dependencies: repos.Dependencies,
		ToolUsage:    repos.ToolUsage,
		Skills:       repos.Skills,
		Dockerfiles:    repos.Dockerfiles,
		Notifications: repos.Notifications,
		Logger:         logger,
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

	// Register routes
	commands.NewRouter(router, appInstance, ctrl, hub, sseHub)
	queries.NewRouter(router, appInstance, ctrl, sseHub)

	// WebSocket endpoint
	upgrader := gorillaws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	wsRouter := router
	if cfg.WSRouter != nil {
		wsRouter = cfg.WSRouter
	}
	wsRouter.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Browsers cannot set custom headers on WebSocket connections.
		// Accept the JWT via the ?token= query parameter instead.
		if cfg.AuthQueries != nil {
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Error(w, `{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`, http.StatusUnauthorized)
				return
			}
			if _, err := cfg.AuthQueries.ValidateJWT(r.Context(), token); err != nil {
				http.Error(w, `{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`, http.StatusUnauthorized)
				return
			}
		}
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

