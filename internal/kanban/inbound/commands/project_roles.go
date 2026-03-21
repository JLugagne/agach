package commands

import (
	"errors"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
	"github.com/gorilla/mux"
)

// ProjectRoleCommandsHandler handles per-project role write operations
type ProjectRoleCommandsHandler struct {
	commands   service.Commands
	queries    service.Queries
	controller *controller.Controller
}

func NewProjectRoleCommandsHandler(commands service.Commands, queries service.Queries, ctrl *controller.Controller) *ProjectRoleCommandsHandler {
	return &ProjectRoleCommandsHandler{
		commands:   commands,
		queries:    queries,
		controller: ctrl,
	}
}

func (h *ProjectRoleCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/roles", h.CreateProjectRole).Methods("POST")
	router.HandleFunc("/api/projects/{id}/roles/{slug}", h.UpdateProjectRole).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/roles/{slug}", h.DeleteProjectRole).Methods("DELETE")
}

func (h *ProjectRoleCommandsHandler) CreateProjectRole(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	var req pkgkanban.CreateRoleRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidRoleRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidRoleRequest, err))
		return
	}

	role, err := h.commands.CreateProjectRole(r.Context(), projectID, req.Slug, req.Name, req.Icon, req.Color, req.Description, req.PromptHint, req.PromptTemplate, req.TechStack, req.SortOrder)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicRole(role))
}

func (h *ProjectRoleCommandsHandler) UpdateProjectRole(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	slug := mux.Vars(r)["slug"]

	var req pkgkanban.UpdateRoleRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidRoleRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidRoleRequest, err))
		return
	}

	role, err := h.queries.GetProjectRoleBySlug(r.Context(), projectID, slug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	var name, icon, color, description, promptHint, promptTemplate string
	var techStack []string
	var sortOrder int

	if req.Name != nil {
		name = *req.Name
	}
	if req.Icon != nil {
		icon = *req.Icon
	}
	if req.Color != nil {
		color = *req.Color
	}
	if req.Description != nil {
		description = *req.Description
	}
	if req.PromptHint != nil {
		promptHint = *req.PromptHint
	}
	if req.PromptTemplate != nil {
		promptTemplate = *req.PromptTemplate
	}
	if req.TechStack != nil {
		techStack = *req.TechStack
	}
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	if err := h.commands.UpdateProjectRole(r.Context(), projectID, role.ID, name, icon, color, description, promptHint, promptTemplate, techStack, sortOrder); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{"message": "role updated"})
}

func (h *ProjectRoleCommandsHandler) DeleteProjectRole(w http.ResponseWriter, r *http.Request) {
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

	if err := h.commands.DeleteProjectRole(r.Context(), projectID, role.ID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{"message": "role deleted"})
}
