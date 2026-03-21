package queries

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/sse"
	"github.com/gorilla/mux"
)

// RegisterAllRoutes wires all query handlers onto the given router.
func RegisterAllRoutes(router *mux.Router, app service.Queries, ctrl *controller.Controller, sseHub *sse.Hub) {
	NewProjectQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewRoleQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewTaskQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewCommentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewDependencyQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewToolUsageQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewTimelineQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewProjectRoleQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewColdStartStatsQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewSSEHandler(sseHub).RegisterRoutes(router)
}
