package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/gorilla/mux"
)

// ColdStartStatsQueriesHandler handles cold start statistics read operations
type ColdStartStatsQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewColdStartStatsQueriesHandler creates a new cold start stats queries handler
func NewColdStartStatsQueriesHandler(queries service.Queries, ctrl *controller.Controller) *ColdStartStatsQueriesHandler {
	return &ColdStartStatsQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers cold start stats query routes
func (h *ColdStartStatsQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/stats/cold-start", h.handleGetColdStartStats).Methods("GET")
}

// handleGetColdStartStats returns aggregated cold-start token statistics grouped by role for a project.
func (h *ColdStartStatsQueriesHandler) handleGetColdStartStats(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	stats, err := h.queries.GetColdStartStats(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicColdStartStats(stats))
}
