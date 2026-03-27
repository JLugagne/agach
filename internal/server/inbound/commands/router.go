package commands

import (
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/gorilla/mux"
)

// App combines Commands and Queries so that handlers needing both interfaces
// can be constructed from a single app instance.
type App interface {
	service.Commands
	service.Queries
}

// RouterOptions holds optional dependencies for the command router.
type RouterOptions struct {
	ChatService  service.ChatService
	TeamResolver TeamIDResolver
}

// NewRouter wires all command handlers onto the given router.
// chatSvc is optional: when provided the chat command routes are registered.
func NewRouter(router *mux.Router, app App, ctrl *controller.Controller, hub *websocket.Hub, dataDir string, opts ...RouterOptions) {
	var opt RouterOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	NewProjectCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewAgentCommandsHandler(app, app, ctrl, hub).RegisterRoutes(router)

	taskHandler := NewTaskCommandsHandler(app, ctrl, hub, app)
	taskHandler.SetTeamResolver(opt.TeamResolver)
	taskHandler.RegisterRoutes(router)

	commentHandler := NewCommentCommandsHandlerWithQueries(app, app, ctrl, hub)
	commentHandler.SetTeamResolver(opt.TeamResolver)
	commentHandler.RegisterRoutes(router)

	NewImageCommandsHandler(app, ctrl).RegisterRoutes(router)
	NewSeenCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewProjectAgentCommandsHandler(app, app, ctrl, hub).RegisterRoutes(router)
	NewSkillCommandsHandler(app, app, ctrl, hub).RegisterRoutes(router)
	NewSpecializedAgentCommandsHandler(app, app, ctrl, hub).RegisterRoutes(router)
	NewDockerfileCommandsHandler(app, ctrl).RegisterRoutes(router)

	featureHandler := NewFeatureCommandsHandler(app, ctrl, hub, app)
	featureHandler.SetTeamResolver(opt.TeamResolver)
	featureHandler.RegisterRoutes(router)

	NewNotificationCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewProjectAccessHandler(app, app, ctrl, hub).RegisterRoutes(router)

	if opt.ChatService != nil {
		NewChatsHandler(opt.ChatService, app, ctrl, hub, dataDir).RegisterRoutes(router)
	}
}
