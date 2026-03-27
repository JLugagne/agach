package queries

import (
	"net/http"
	"strconv"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/gorilla/mux"
)

// TimelineQueriesHandler handles timeline read operations
type TimelineQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewTimelineQueriesHandler creates a new timeline queries handler
func NewTimelineQueriesHandler(queries service.Queries, ctrl *controller.Controller) *TimelineQueriesHandler {
	return &TimelineQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers timeline query routes
func (h *TimelineQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/stats/timeline", h.handleGetTimeline).Methods("GET")
}

// handleGetTimeline returns daily task creation and completion counts for a project.
// Accepts optional ?days= query parameter (default: 30, max: 365).
func (h *TimelineQueriesHandler) handleGetTimeline(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	days := 30
	if daysParam := r.URL.Query().Get("days"); daysParam != "" {
		parsed, err := strconv.Atoi(daysParam)
		if err != nil || parsed < 1 || parsed > 365 {
			h.controller.SendFail(w, r, nil, domain.ErrInvalidTaskData)
			return
		}
		days = parsed
	}

	entries, err := h.queries.GetTimeline(r.Context(), projectID, days)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicTimeline(entries))
}
