package commands

import (
	"context"
	"fmt"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/gorilla/mux"
)

type ProjectAccessHandler struct {
	commands   service.Commands
	queries    service.Queries
	controller *controller.Controller
	hub        *websocket.Hub
}

func NewProjectAccessHandler(commands service.Commands, queries service.Queries, ctrl *controller.Controller, hub *websocket.Hub) *ProjectAccessHandler {
	return &ProjectAccessHandler{commands: commands, queries: queries, controller: ctrl, hub: hub}
}

func (h *ProjectAccessHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/access/users", h.ListUserAccess).Methods("GET")
	router.HandleFunc("/api/projects/{id}/access/users", h.GrantUserAccess).Methods("POST")
	router.HandleFunc("/api/projects/{id}/access/users/{userId}", h.UpdateUserAccessRole).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/access/users/{userId}", h.RevokeUserAccess).Methods("DELETE")
	router.HandleFunc("/api/projects/{id}/access/teams", h.ListTeamAccess).Methods("GET")
	router.HandleFunc("/api/projects/{id}/access/teams", h.GrantTeamAccess).Methods("POST")
	router.HandleFunc("/api/projects/{id}/access/teams/{teamId}", h.RevokeTeamAccess).Methods("DELETE")
}

func (h *ProjectAccessHandler) ListUserAccess(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	access, err := h.queries.ListProjectUserAccess(r.Context(), projectID)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	resp := make([]pkgserver.ProjectUserAccessResponse, len(access))
	for i, a := range access {
		resp[i] = pkgserver.ProjectUserAccessResponse{
			ID:        a.ID,
			ProjectID: string(a.ProjectID),
			UserID:    a.UserID,
			Role:      a.Role,
			CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	h.controller.SendSuccess(w, r, resp)
}

func (h *ProjectAccessHandler) GrantUserAccess(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	var req pkgserver.GrantUserAccessRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidProjectAccessRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}
	if err := h.commands.GrantUserAccess(r.Context(), projectID, req.UserID, req.Role); err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	h.hub.Broadcast(websocket.Event{Type: "project_access_updated", Data: map[string]string{"project_id": string(projectID)}})
	h.notifyAccessChange(r.Context(), projectID, domain.SeverityInfo, "Project shared with you", fmt.Sprintf("Role: %s", req.Role))
	h.controller.SendSuccess(w, r, map[string]string{"message": "user access granted"})
}

func (h *ProjectAccessHandler) UpdateUserAccessRole(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	userID := mux.Vars(r)["userId"]
	var req pkgserver.UpdateUserAccessRoleRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidProjectAccessRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}
	if err := h.commands.UpdateUserAccessRole(r.Context(), projectID, userID, req.Role); err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	h.hub.Broadcast(websocket.Event{Type: "project_access_updated", Data: map[string]string{"project_id": string(projectID)}})
	h.notifyAccessChange(r.Context(), projectID, domain.SeverityInfo, "Project role updated", fmt.Sprintf("New role: %s", req.Role))
	h.controller.SendSuccess(w, r, map[string]string{"message": "user access role updated"})
}

func (h *ProjectAccessHandler) RevokeUserAccess(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	userID := mux.Vars(r)["userId"]
	if err := h.commands.RevokeUserAccess(r.Context(), projectID, userID); err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	h.hub.Broadcast(websocket.Event{Type: "project_access_updated", Data: map[string]string{"project_id": string(projectID)}})
	h.notifyAccessChange(r.Context(), projectID, domain.SeverityWarning, "Project access revoked", "")
	h.controller.SendSuccess(w, r, map[string]string{"message": "user access revoked"})
}

func (h *ProjectAccessHandler) ListTeamAccess(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	access, err := h.queries.ListProjectTeamAccess(r.Context(), projectID)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	resp := make([]pkgserver.ProjectTeamAccessResponse, len(access))
	for i, a := range access {
		resp[i] = pkgserver.ProjectTeamAccessResponse{
			ID:        a.ID,
			ProjectID: string(a.ProjectID),
			TeamID:    a.TeamID,
			CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	h.controller.SendSuccess(w, r, resp)
}

func (h *ProjectAccessHandler) GrantTeamAccess(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	var req pkgserver.GrantTeamAccessRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidProjectAccessRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}
	if err := h.commands.GrantTeamAccess(r.Context(), projectID, req.TeamID); err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	h.hub.Broadcast(websocket.Event{Type: "project_access_updated", Data: map[string]string{"project_id": string(projectID)}})
	h.notifyAccessChange(r.Context(), projectID, domain.SeverityInfo, "Team added to project", "")
	h.controller.SendSuccess(w, r, map[string]string{"message": "team access granted"})
}

func (h *ProjectAccessHandler) RevokeTeamAccess(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	teamID := mux.Vars(r)["teamId"]
	if err := h.commands.RevokeTeamAccess(r.Context(), projectID, teamID); err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	h.hub.Broadcast(websocket.Event{Type: "project_access_updated", Data: map[string]string{"project_id": string(projectID)}})
	h.notifyAccessChange(r.Context(), projectID, domain.SeverityWarning, "Team removed from project", "")
	h.controller.SendSuccess(w, r, map[string]string{"message": "team access revoked"})
}

// notifyAccessChange creates and broadcasts a notification for project access changes.
func (h *ProjectAccessHandler) notifyAccessChange(ctx context.Context, projectID domain.ProjectID, severity domain.NotificationSeverity, title, text string) {
	projectName := string(projectID)
	if p, err := h.queries.GetProject(ctx, projectID); err == nil && p != nil {
		projectName = p.Name
	}

	detail := projectName
	if text != "" {
		detail = fmt.Sprintf("%s — %s", projectName, text)
	}

	pid := projectID
	notification, err := h.commands.CreateNotification(ctx, &pid, domain.NotificationScopeProject, "", severity, title, detail, "", "", "")
	if err != nil {
		return
	}
	h.hub.Broadcast(websocket.Event{
		Type:      "notification",
		ProjectID: string(projectID),
		Data:      converters.ToPublicNotification(notification),
	})
}
