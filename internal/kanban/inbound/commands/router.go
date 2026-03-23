package commands

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/sse"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
)

// App combines Commands and Queries so that handlers needing both interfaces
// can be constructed from a single app instance.
type App interface {
	service.Commands
	service.Queries
}

// NewRouter wires all command handlers onto the given router.
func NewRouter(router *mux.Router, app App, ctrl *controller.Controller, hub *websocket.Hub, sseHub *sse.Hub) {
	NewProjectCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewAgentCommandsHandler(app, app, ctrl, hub).RegisterRoutes(router)
	NewTaskCommandsHandler(app, ctrl, hub, sseHub).RegisterRoutes(router)
	NewCommentCommandsHandlerWithQueries(app, app, ctrl, hub).RegisterRoutes(router)
	NewImageCommandsHandler(app, ctrl).RegisterRoutes(router)
	NewSeenCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewProjectAgentCommandsHandler(app, app, ctrl, hub).RegisterRoutes(router)
	NewSkillCommandsHandler(app, app, ctrl, hub).RegisterRoutes(router)
	NewDockerfileCommandsHandler(app, ctrl).RegisterRoutes(router)
	NewFeatureCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
}
