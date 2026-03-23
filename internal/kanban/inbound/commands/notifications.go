package commands

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
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
	router.HandleFunc("/api/projects/{id}/notifications/{notificationId}/read", h.MarkNotificationRead).Methods("PUT")
	router.HandleFunc("/api/projects/{id}/notifications/read-all", h.MarkAllNotificationsRead).Methods("PUT")
	router.HandleFunc("/api/projects/{id}/notifications/{notificationId}", h.DeleteNotification).Methods("DELETE")
}

// CreateNotification creates a new notification
func (h *NotificationCommandsHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	var req pkgkanban.CreateNotificationRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidNotificationRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	notification, err := h.commands.CreateNotification(
		r.Context(), projectID,
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
		ProjectID: string(projectID),
		Data:      resp,
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

// MarkAllNotificationsRead marks all notifications as read for a project
func (h *NotificationCommandsHandler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	if err := h.commands.MarkAllNotificationsRead(r.Context(), projectID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{"message": "all notifications marked as read"})
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
