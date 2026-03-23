package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// AgentQueriesHandler handles role read operations
type AgentQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewAgentQueriesHandler creates a new role queries handler
func NewAgentQueriesHandler(queries service.Queries, ctrl *controller.Controller) *AgentQueriesHandler {
	return &AgentQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers role query routes
func (h *AgentQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/agents", h.ListAgents).Methods("GET")
	router.HandleFunc("/api/agents/{slug}", h.GetAgent).Methods("GET")
}

// ListAgents lists all roles
func (h *AgentQueriesHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	roles, err := h.queries.ListAgents(r.Context())
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

// GetAgent gets a single role by slug
func (h *AgentQueriesHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]

	role, err := h.queries.GetAgentBySlug(r.Context(), slug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicAgent(*role))
}
