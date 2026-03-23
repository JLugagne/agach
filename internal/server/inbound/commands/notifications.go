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

// NotificationCommandsHandler handles notification write operations
type NotificationCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewNotificationCommandsHandler creates a new notification commands handler
func NewNotificationCommandsHandler(commands service.Commands, ctrl *controller.Controller, hub *websocket.Hub) *NotificationCommandsHandler {
	return &NotificationCommandsHandler{
		commands:   commands,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers notification command routes
func (h *NotificationCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/notifications", h.CreateNotification).Methods("POST")
	router.HandleFunc("/api/notifications", h.CreateGlobalNotification).Methods("POST")
	router.HandleFunc("/api/notifications/{notificationId}/read", h.MarkNotificationRead).Methods("PUT")
	router.HandleFunc("/api/notifications/read-all", h.MarkAllNotificationsRead).Methods("PUT")
	router.HandleFunc("/api/projects/{id}/notifications/read-all", h.MarkAllProjectNotificationsRead).Methods("PUT")
	router.HandleFunc("/api/notifications/{notificationId}", h.DeleteNotification).Methods("DELETE")
}

// CreateNotification creates a new notification for a project
func (h *NotificationCommandsHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProjectID(mux.Vars(r)["id"])
	projectID := &pid

	var req pkgserver.CreateNotificationRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidNotificationRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	scope := domain.NotificationScopeProject
	if req.Scope != "" {
		scope = domain.NotificationScope(req.Scope)
	}

	notification, err := h.commands.CreateNotification(
		r.Context(), projectID, scope, req.AgentSlug,
		domain.NotificationSeverity(req.Severity),
		req.Title, req.Text, req.LinkURL, req.LinkText, req.LinkStyle,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicNotification(notification)
	h.hub.Broadcast(websocket.Event{
		Type:      "notification",
		ProjectID: string(pid),
		Data:      resp,
	})
	h.controller.SendSuccess(w, r, resp)
}

// CreateGlobalNotification creates a new notification without a project
func (h *NotificationCommandsHandler) CreateGlobalNotification(w http.ResponseWriter, r *http.Request) {
	var req pkgserver.CreateNotificationRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidNotificationRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	scope := domain.NotificationScopeGlobal
	if req.Scope != "" {
		scope = domain.NotificationScope(req.Scope)
	}

	notification, err := h.commands.CreateNotification(
		r.Context(), nil, scope, req.AgentSlug,
		domain.NotificationSeverity(req.Severity),
		req.Title, req.Text, req.LinkURL, req.LinkText, req.LinkStyle,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicNotification(notification)
	h.hub.Broadcast(websocket.Event{
		Type: "notification",
		Data: resp,
	})
	h.controller.SendSuccess(w, r, resp)
}

// MarkNotificationRead marks a notification as read
func (h *NotificationCommandsHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	notificationID := domain.NotificationID(mux.Vars(r)["notificationId"])
	if err := h.commands.MarkNotificationRead(r.Context(), notificationID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}
	h.controller.SendSuccess(w, r, map[string]string{"message": "notification marked as read"})
}

// MarkAllNotificationsRead marks all notifications as read
func (h *NotificationCommandsHandler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if err := h.commands.MarkAllNotificationsRead(r.Context(), nil); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}
	h.controller.SendSuccess(w, r, map[string]string{"message": "all notifications marked as read"})
}

// MarkAllProjectNotificationsRead marks all notifications as read for a project
func (h *NotificationCommandsHandler) MarkAllProjectNotificationsRead(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProjectID(mux.Vars(r)["id"])
	if err := h.commands.MarkAllNotificationsRead(r.Context(), &pid); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}
	h.controller.SendSuccess(w, r, map[string]string{"message": "all project notifications marked as read"})
}

// DeleteNotification deletes a notification
func (h *NotificationCommandsHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	notificationID := domain.NotificationID(mux.Vars(r)["notificationId"])
	if err := h.commands.DeleteNotification(r.Context(), notificationID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}
	h.controller.SendSuccess(w, r, map[string]string{"message": "notification deleted"})
}
