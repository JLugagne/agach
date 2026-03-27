package queries

import (
	"net/http"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
)

// checkProjectAccess verifies the actor from the request context has access to
// the given project. Admins are always granted access. When no actor is present
// in context (auth middleware not active), access is allowed — authentication
// is the middleware's responsibility, not ours.
func checkProjectAccess(r *http.Request, projectID domain.ProjectID, queries service.Queries, teamResolver TeamIDResolver) bool {
	if queries == nil {
		return true
	}

	actor, ok := r.Context().Value(middleware.ActorContextKey).(identitydomain.Actor)
	if !ok || actor.IsZero() {
		// No actor in context means auth middleware is not active (e.g. MCP mode).
		return true
	}

	if actor.IsAdmin() {
		return true
	}

	var teamIDs []string
	if teamResolver != nil {
		tids, _ := teamResolver.GetUserTeamIDs(r.Context(), actor.UserID)
		for _, tid := range tids {
			teamIDs = append(teamIDs, tid.String())
		}
	}

	ok, err := queries.HasProjectAccess(r.Context(), projectID, actor.UserID.String(), teamIDs)
	if err != nil || !ok {
		return false
	}
	return true
}
