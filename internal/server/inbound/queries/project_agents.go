package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/gorilla/mux"
)

// ProjectAgentQueriesHandler handles project-agent read operations
type ProjectAgentQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

func NewProjectAgentQueriesHandler(queries service.Queries, ctrl *controller.Controller) *ProjectAgentQueriesHandler {
	return &ProjectAgentQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

func (h *ProjectAgentQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{projectId}/agents", h.ListProjectAgents).Methods("GET")
}

func (h *ProjectAgentQueriesHandler) ListProjectAgents(w http.ResponseWriter, r *http.Request) {
	rawID := mux.Vars(r)["projectId"]
	projectID, err := domain.ParseProjectID(rawID)
	if err != nil {
		h.controller.SendFail(w, r, nil, domain.ErrProjectNotFound)
		return
	}

	roles, err := h.queries.ListProjectAgents(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	result := make([]pkgserver.AgentResponse, len(roles))
	for i, role := range roles {
		skillCount := 0
		if skills, err := h.queries.ListAgentSkills(r.Context(), role.Slug); err == nil {
			skillCount = len(skills)
		}
		specializedCount := 0
		if count, err := h.queries.CountSpecializedByParent(r.Context(), role.Slug); err == nil {
			specializedCount = count
		}
		result[i] = converters.ToPublicAgentWithCount(role, skillCount, specializedCount)
	}

	h.controller.SendSuccess(w, r, result)
}
