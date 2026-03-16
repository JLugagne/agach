package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// RoleQueriesHandler handles role read operations
type RoleQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewRoleQueriesHandler creates a new role queries handler
func NewRoleQueriesHandler(queries service.Queries, ctrl *controller.Controller) *RoleQueriesHandler {
	return &RoleQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers role query routes
func (h *RoleQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/roles", h.ListRoles).Methods("GET")
	router.HandleFunc("/api/roles/{slug}", h.GetRole).Methods("GET")
}

// ListRoles lists all roles
func (h *RoleQueriesHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.queries.ListRoles(r.Context())
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

// GetRole gets a single role by slug
func (h *RoleQueriesHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]

	role, err := h.queries.GetRoleBySlug(r.Context(), slug)
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
