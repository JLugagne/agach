package commands

import (
	"encoding/json"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// ColumnCommandsHandler handles column write operations
type ColumnCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
}

// NewColumnCommandsHandler creates a new column commands handler
func NewColumnCommandsHandler(commands service.Commands, ctrl *controller.Controller) *ColumnCommandsHandler {
	return &ColumnCommandsHandler{
		commands:   commands,
		controller: ctrl,
	}
}

// RegisterRoutes registers column command routes
func (h *ColumnCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/columns/{slug}/wip-limit", h.UpdateWIPLimit).Methods("PATCH")
}

type updateWIPLimitRequest struct {
	WIPLimit int `json:"wip_limit"`
}

// UpdateWIPLimit updates the WIP limit for a column
func (h *ColumnCommandsHandler) UpdateWIPLimit(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	slug := domain.ColumnSlug(mux.Vars(r)["slug"])

	var req updateWIPLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	if req.WIPLimit < 0 {
		h.controller.SendFail(w, r, nil, &domain.Error{Code: "INVALID_WIP_LIMIT", Message: "WIP limit must be >= 0"})
		return
	}

	err := h.commands.UpdateColumnWIPLimit(r.Context(), projectID, slug, req.WIPLimit)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
