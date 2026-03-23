package commands

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
)

// ProjectAgentCommandsHandler handles project-agent assignment write operations
type ProjectAgentCommandsHandler struct {
	commands   service.Commands
	queries    service.Queries
	controller *controller.Controller
	hub        *websocket.Hub
}

func NewProjectAgentCommandsHandler(commands service.Commands, queries service.Queries, ctrl *controller.Controller, hub *websocket.Hub) *ProjectAgentCommandsHandler {
	return &ProjectAgentCommandsHandler{
		commands:   commands,
		queries:    queries,
		controller: ctrl,
		hub:        hub,
	}
}

func (h *ProjectAgentCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{projectId}/agents", h.AssignAgent).Methods("POST")
	router.HandleFunc("/api/projects/{projectId}/agents/bulk-reassign", h.BulkReassign).Methods("POST")
	router.HandleFunc("/api/projects/{projectId}/agents/{slug}", h.RemoveAgent).Methods("DELETE")
}

func (h *ProjectAgentCommandsHandler) AssignAgent(w http.ResponseWriter, r *http.Request) {
	rawID := mux.Vars(r)["projectId"]
	projectID, err := domain.ParseProjectID(rawID)
	if err != nil {
		h.controller.SendFail(w, r, nil, domain.ErrProjectNotFound)
		return
	}

	var req pkgkanban.AssignAgentToProjectRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidAgentAssignmentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidAgentAssignmentRequest, err))
		return
	}

	if err := h.commands.AssignAgentToProject(r.Context(), projectID, req.AgentSlug); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "agent_assigned_to_project",
		Data: map[string]string{
			"project_id": string(projectID),
			"agent_slug": req.AgentSlug,
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "agent assigned"})
}

func (h *ProjectAgentCommandsHandler) RemoveAgent(w http.ResponseWriter, r *http.Request) {
	rawID := mux.Vars(r)["projectId"]
	projectID, err := domain.ParseProjectID(rawID)
	if err != nil {
		h.controller.SendFail(w, r, nil, domain.ErrProjectNotFound)
		return
	}

	slug := mux.Vars(r)["slug"]
	if slug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrAgentSlugRequired)
		return
	}

	var req pkgkanban.RemoveAgentFromProjectRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.controller.SendFail(w, r, nil, pkgkanban.ErrInvalidAgentAssignmentRequest)
			return
		}
	}

	if err := h.commands.RemoveAgentFromProject(r.Context(), projectID, slug, req.ReassignTo, req.ClearAssignment); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "agent_removed_from_project",
		Data: map[string]string{
			"project_id": string(projectID),
			"agent_slug": slug,
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "agent removed"})
}

func (h *ProjectAgentCommandsHandler) BulkReassign(w http.ResponseWriter, r *http.Request) {
	rawID := mux.Vars(r)["projectId"]
	projectID, err := domain.ParseProjectID(rawID)
	if err != nil {
		h.controller.SendFail(w, r, nil, domain.ErrProjectNotFound)
		return
	}

	var req pkgkanban.BulkReassignTasksRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidAgentAssignmentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidAgentAssignmentRequest, err))
		return
	}

	n, err := h.commands.BulkReassignTasks(r.Context(), projectID, req.OldSlug, req.NewSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, pkgkanban.BulkReassignTasksResponse{UpdatedCount: n})
}
