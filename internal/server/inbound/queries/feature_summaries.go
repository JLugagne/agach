package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
)

// FeatureSummariesHandler handles feature task summaries read operations
type FeatureSummariesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewFeatureSummariesHandler creates a new feature summaries handler
func NewFeatureSummariesHandler(queries service.Queries, ctrl *controller.Controller) *FeatureSummariesHandler {
	return &FeatureSummariesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers feature summaries query routes
func (h *FeatureSummariesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/features/{featureId}/task-summaries", h.ListFeatureTaskSummaries).Methods("GET")
}

// ListFeatureTaskSummaries lists completed task summaries for a feature
func (h *FeatureSummariesHandler) ListFeatureTaskSummaries(w http.ResponseWriter, r *http.Request) {
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	summaries, err := h.queries.ListFeatureTaskSummaries(r.Context(), featureID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicTaskSummaries(summaries)
	h.controller.SendSuccess(w, r, resp)
}
