package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
)

// SkillQueriesHandler handles skill read operations
type SkillQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewSkillQueriesHandler creates a new skill queries handler
func NewSkillQueriesHandler(queries service.Queries, ctrl *controller.Controller) *SkillQueriesHandler {
	return &SkillQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers skill query routes
func (h *SkillQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/skills", h.ListSkills).Methods("GET")
	router.HandleFunc("/api/skills/{slug}", h.GetSkill).Methods("GET")
	router.HandleFunc("/api/agents/{slug}/skills", h.ListAgentSkills).Methods("GET")
}

// ListSkills returns all global skills
func (h *SkillQueriesHandler) ListSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := h.queries.ListSkills(r.Context())
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

// GetSkill returns a single skill by slug
func (h *SkillQueriesHandler) GetSkill(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]

	skill, err := h.queries.GetSkillBySlug(r.Context(), slug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicSkill(*skill))
}

// ListAgentSkills returns all skills assigned to an agent
func (h *SkillQueriesHandler) ListAgentSkills(w http.ResponseWriter, r *http.Request) {
	agentSlug := mux.Vars(r)["slug"]

	skills, err := h.queries.ListAgentSkills(r.Context(), agentSlug)
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
