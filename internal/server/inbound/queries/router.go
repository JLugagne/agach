package queries

import (
	appservice "github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/sse"
	"github.com/gorilla/mux"
)

// NewRouter wires all query handlers onto the given router.
func NewRouter(router *mux.Router, app service.Queries, ctrl *controller.Controller, sseHub *sse.Hub, dataDir string) {
	NewProjectQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewAgentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewTaskQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewCommentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewDependencyQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewToolUsageQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewTimelineQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewProjectAgentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewColdStartStatsQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewModelStatsQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewSkillQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewDockerfileQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewFeatureQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewNotificationQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewSSEHandler(sseHub).RegisterRoutes(router)

	// Chat queries handler - requires app to be castable to an app type with ChatService method
	if appWithChat, ok := app.(*appservice.App); ok {
		NewChatQueriesHandler(appWithChat.ChatService(), ctrl, dataDir).RegisterRoutes(router)
	}
}
