package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// ModelStatsQueriesHandler handles model token statistics and pricing read operations
type ModelStatsQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewModelStatsQueriesHandler creates a new handler
func NewModelStatsQueriesHandler(queries service.Queries, ctrl *controller.Controller) *ModelStatsQueriesHandler {
	return &ModelStatsQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers model stats query routes
func (h *ModelStatsQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/stats/model-tokens", h.handleGetModelTokenStats).Methods("GET")
	router.HandleFunc("/api/model-pricing", h.handleListModelPricing).Methods("GET")
}

func (h *ModelStatsQueriesHandler) handleGetModelTokenStats(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	stats, err := h.queries.GetModelTokenStats(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, stats)
}

func (h *ModelStatsQueriesHandler) handleListModelPricing(w http.ResponseWriter, r *http.Request) {
	pricing, err := h.queries.ListModelPricing(r.Context())
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, pricing)
}
