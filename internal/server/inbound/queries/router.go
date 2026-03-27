package queries

import (
	"context"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/gorilla/mux"
)

// TeamIDResolver resolves a user's team IDs for project access filtering.
type TeamIDResolver interface {
	GetUserTeamIDs(ctx context.Context, userID identitydomain.UserID) ([]identitydomain.TeamID, error)
}

// NewRouter wires all query handlers onto the given router.
func NewRouter(router *mux.Router, app service.Queries, ctrl *controller.Controller, dataDir string, chatService service.ChatService, teamResolver ...TeamIDResolver) {
	var resolver TeamIDResolver
	if len(teamResolver) > 0 {
		resolver = teamResolver[0]
	}
	NewProjectQueriesHandler(app, ctrl, resolver).RegisterRoutes(router)
	NewAgentQueriesHandler(app, ctrl).RegisterRoutes(router)
	NewTaskQueriesHandler(app, ctrl, resolver).RegisterRoutes(router)
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

	if chatService != nil {
		NewChatQueriesHandler(chatService, ctrl, dataDir).RegisterRoutes(router)
	}
}
