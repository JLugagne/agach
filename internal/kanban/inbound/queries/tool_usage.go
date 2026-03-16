package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// ToolUsageQueriesHandler handles tool usage read operations
type ToolUsageQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewToolUsageQueriesHandler creates a new tool usage queries handler
func NewToolUsageQueriesHandler(queries service.Queries, ctrl *controller.Controller) *ToolUsageQueriesHandler {
	return &ToolUsageQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers tool usage query routes
func (h *ToolUsageQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/tool-usage", h.handleGetToolUsage).Methods("GET")
}

// handleGetToolUsage returns tool usage statistics for a project
func (h *ToolUsageQueriesHandler) handleGetToolUsage(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	stats, err := h.queries.GetToolUsageForProject(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicToolUsageStats(stats))
}
