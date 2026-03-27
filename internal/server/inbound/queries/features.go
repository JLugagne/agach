package queries

import (
	"net/http"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
)

// FeatureQueriesHandler handles feature read operations
type FeatureQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewFeatureQueriesHandler creates a new feature queries handler
func NewFeatureQueriesHandler(queries service.Queries, ctrl *controller.Controller) *FeatureQueriesHandler {
	return &FeatureQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers feature query routes
func (h *FeatureQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/features", h.ListFeatures).Methods("GET")
	router.HandleFunc("/api/projects/{id}/features/{featureId}", h.GetFeature).Methods("GET")
	router.HandleFunc("/api/projects/{id}/stats/features", h.GetFeatureStats).Methods("GET")
}

// ListFeatures lists features of a project, optionally filtered by status
func (h *FeatureQueriesHandler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	// Parse optional status filter from query param (comma-separated)
	var statusFilter []domain.FeatureStatus
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		statuses := strings.Split(statusParam, ",")
		for _, s := range statuses {
			s = strings.TrimSpace(s)
			fs := domain.FeatureStatus(s)
			if s != "" && domain.ValidFeatureStatuses[fs] {
				statusFilter = append(statusFilter, fs)
			}
		}
	}

	features, err := h.queries.ListFeatures(r.Context(), projectID, statusFilter)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicFeaturesWithSummary(features)
	h.controller.SendSuccess(w, r, resp)
}

// GetFeature gets a single feature
func (h *FeatureQueriesHandler) GetFeature(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	feature, err := h.queries.GetFeature(r.Context(), featureID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	if feature.ProjectID != projectID {
		h.controller.SendFail(w, r, nil, domain.ErrFeatureNotFound)
		return
	}

	resp := converters.ToPublicFeature(*feature)
	h.controller.SendSuccess(w, r, resp)
}

// GetFeatureStats gets feature statistics for a project
func (h *FeatureQueriesHandler) GetFeatureStats(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	stats, err := h.queries.GetFeatureStats(r.Context(), projectID)
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
