package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
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

	skillCount := 0
	if skills, err := h.queries.ListAgentSkills(r.Context(), slug); err == nil {
		skillCount = len(skills)
	}
	specializedCount := 0
	if count, err := h.queries.CountSpecializedByParent(r.Context(), slug); err == nil {
		specializedCount = count
	}

	h.controller.SendSuccess(w, r, converters.ToPublicAgentWithCount(*role, skillCount, specializedCount))
}
