package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// ProjectRoleQueriesHandler handles per-project role read operations
type ProjectRoleQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

func NewProjectRoleQueriesHandler(queries service.Queries, ctrl *controller.Controller) *ProjectRoleQueriesHandler {
	return &ProjectRoleQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

func (h *ProjectRoleQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/roles", h.ListProjectRoles).Methods("GET")
	router.HandleFunc("/api/projects/{id}/roles/{slug}", h.GetProjectRole).Methods("GET")
}

func (h *ProjectRoleQueriesHandler) ListProjectRoles(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	roles, err := h.queries.ListProjectRoles(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicRoles(roles))
}

func (h *ProjectRoleQueriesHandler) GetProjectRole(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	slug := mux.Vars(r)["slug"]

	role, err := h.queries.GetProjectRoleBySlug(r.Context(), projectID, slug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicRole(*role))
}
