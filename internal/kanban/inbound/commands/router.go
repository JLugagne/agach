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

// RegisterAllRoutes wires all command handlers onto the given router.
func RegisterAllRoutes(router *mux.Router, app App, ctrl *controller.Controller, hub *websocket.Hub, sseHub *sse.Hub) {
	NewProjectCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewRoleCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewTaskCommandsHandler(app, ctrl, hub, sseHub).RegisterRoutes(router)
	NewCommentCommandsHandlerWithQueries(app, app, ctrl, hub).RegisterRoutes(router)
	NewImageCommandsHandler(app, ctrl).RegisterRoutes(router)
	NewSeenCommandsHandler(app, ctrl, hub).RegisterRoutes(router)
	NewColumnCommandsHandler(app, ctrl).RegisterRoutes(router)
	NewProjectRoleCommandsHandler(app, app, ctrl).RegisterRoutes(router)
}
