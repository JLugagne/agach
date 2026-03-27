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

type SpecializedAgentQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

func NewSpecializedAgentQueriesHandler(queries service.Queries, ctrl *controller.Controller) *SpecializedAgentQueriesHandler {
	return &SpecializedAgentQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

func (h *SpecializedAgentQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/agents/{slug}/specialized", h.HandleListSpecializedAgents).Methods("GET")
	router.HandleFunc("/api/agents/{slug}/specialized/{specSlug}", h.HandleGetSpecializedAgent).Methods("GET")
	router.HandleFunc("/api/agents/{slug}/specialized/{specSlug}/skills", h.HandleListSpecializedAgentSkills).Methods("GET")
}

func (h *SpecializedAgentQueriesHandler) HandleListSpecializedAgents(w http.ResponseWriter, r *http.Request) {
	parentSlug := mux.Vars(r)["slug"]

	agents, err := h.queries.ListSpecializedAgents(r.Context(), parentSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	result := make([]pkgserver.SpecializedAgentResponse, len(agents))
	for i, agent := range agents {
		skillCount := 0
		if skills, err := h.queries.ListSpecializedAgentSkills(r.Context(), agent.Slug); err == nil {
			skillCount = len(skills)
		}
		result[i] = converters.ToPublicSpecializedAgent(agent, parentSlug, skillCount)
	}

	h.controller.SendSuccess(w, r, result)
}

func (h *SpecializedAgentQueriesHandler) HandleGetSpecializedAgent(w http.ResponseWriter, r *http.Request) {
	parentSlug := mux.Vars(r)["slug"]
	specSlug := mux.Vars(r)["specSlug"]

	agent, err := h.queries.GetSpecializedAgent(r.Context(), specSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	skills, err := h.queries.ListSpecializedAgentSkills(r.Context(), specSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicSpecializedAgent(*agent, parentSlug, len(skills)))
}

func (h *SpecializedAgentQueriesHandler) HandleListSpecializedAgentSkills(w http.ResponseWriter, r *http.Request) {
	specSlug := mux.Vars(r)["specSlug"]

	skills, err := h.queries.ListSpecializedAgentSkills(r.Context(), specSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicSkills(skills))
}
