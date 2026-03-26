package queries

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/sse"
	"github.com/gorilla/mux"
)

// NewRouter wires all query handlers onto the given router.
func NewRouter(router *mux.Router, app service.Queries, ctrl *controller.Controller, sseHub *sse.Hub, dataDir string, chatService service.ChatService) {
	NewProjectQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewAgentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewTaskQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewCommentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewDependencyQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewToolUsageQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewTimelineQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewProjectAgentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewAgentDownloadHandler(app, ctrl).RegisterRoutes(router)
	NewColdStartStatsQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewModelStatsQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewSkillQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewSpecializedAgentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewDockerfileQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewFeatureQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewFeatureSummariesHandler(app, ctrl).RegisterRoutes(router)
	NewNotificationQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewSSEHandler(sseHub, ctrl).RegisterRoutes(router)

	if chatService != nil {
		NewChatQueriesHandler(chatService, ctrl, dataDir).RegisterRoutes(router)
	}
}
