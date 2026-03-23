package commands

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
)

// ProjectCommandsHandler handles project write operations
type ProjectCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewProjectCommandsHandler creates a new project commands handler
func NewProjectCommandsHandler(commands service.Commands, ctrl *controller.Controller, hub *websocket.Hub) *ProjectCommandsHandler {
	return &ProjectCommandsHandler{
		commands:   commands,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers project command routes
func (h *ProjectCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects", h.CreateProject).Methods("POST")
	router.HandleFunc("/api/projects/{id}", h.UpdateProject).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}", h.DeleteProject).Methods("DELETE")
}

// CreateProject creates a new project
func (h *ProjectCommandsHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req pkgserver.CreateProjectRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidProjectRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	parentID := converters.ToDomainProjectID(req.ParentID)

	project, err := h.commands.CreateProject(r.Context(), req.Name, req.Description, req.GitURL, req.CreatedByRole, req.CreatedByAgent, parentID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicProject(project)

	h.hub.Broadcast(websocket.Event{
		Type: "project_created",
		Data: resp,
	})

	h.controller.SendSuccess(w, r, resp)
}

// UpdateProject updates a project
func (h *ProjectCommandsHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	projectID := domain.ProjectID(id)

	var req pkgserver.UpdateProjectRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidProjectRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}

	if err := h.commands.UpdateProject(r.Context(), projectID, name, desc, req.GitURL, req.DefaultRole); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{Type: "project_updated", Data: map[string]string{"project_id": id}})
	h.controller.SendSuccess(w, r, map[string]string{"message": "project updated"})
}

// DeleteProject deletes a project and its SQLite database file
func (h *ProjectCommandsHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.controller.SendFail(w, r, nil, domain.ErrProjectNotFound)
		return
	}

	projectID := domain.ProjectID(id)

	err := h.commands.DeleteProject(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast project_deleted event
	h.hub.Broadcast(websocket.Event{
		Type: "project_deleted",
		Data: map[string]string{"project_id": id},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "project deleted"})
}
