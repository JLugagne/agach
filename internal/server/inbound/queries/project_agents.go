package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
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

	h.controller.SendSuccess(w, r, converters.ToPublicAgents(roles))
}
