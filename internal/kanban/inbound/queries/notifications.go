package queries

import (
	"net/http"
	"strconv"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// NotificationQueriesHandler handles notification read operations
type NotificationQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewNotificationQueriesHandler creates a new notification queries handler
func NewNotificationQueriesHandler(queries service.Queries, ctrl *controller.Controller) *NotificationQueriesHandler {
	return &NotificationQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers notification query routes
func (h *NotificationQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/notifications", h.ListNotifications).Methods("GET")
	router.HandleFunc("/api/projects/{id}/notifications/unread-count", h.GetUnreadCount).Methods("GET")
}

// ListNotifications lists notifications for a project
func (h *NotificationQueriesHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	unreadOnly := r.URL.Query().Get("unread") == "true"

	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v > 0 {
			offset = v
		}
	}

	notifications, err := h.queries.ListNotifications(r.Context(), projectID, unreadOnly, limit, offset)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicNotifications(notifications)
	h.controller.SendSuccess(w, r, resp)
}

// GetUnreadCount returns the count of unread notifications
func (h *NotificationQueriesHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	count, err := h.queries.GetNotificationUnreadCount(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, map[string]int{"unread_count": count})
}
