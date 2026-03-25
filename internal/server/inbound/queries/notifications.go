package queries

import (
	"net/http"
	"strconv"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
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
	router.HandleFunc("/api/notifications", h.ListAllNotifications).Methods("GET")
	router.HandleFunc("/api/notifications/unread-count", h.GetGlobalUnreadCount).Methods("GET")
	router.HandleFunc("/api/projects/{id}/notifications", h.ListProjectNotifications).Methods("GET")
	router.HandleFunc("/api/projects/{id}/notifications/unread-count", h.GetProjectUnreadCount).Methods("GET")
}

func parseNotificationQueryParams(r *http.Request) (scope *domain.NotificationScope, agentSlug string, unreadOnly bool, limit, offset int) {
	if s := r.URL.Query().Get("scope"); s != "" {
		sc := domain.NotificationScope(s)
		scope = &sc
	}
	agentSlug = r.URL.Query().Get("agent_slug")
	unreadOnly = r.URL.Query().Get("unread") == "true"
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v > 0 {
			offset = v
		}
	}
	return
}

// ListAllNotifications lists all notifications with optional filters
func (h *NotificationQueriesHandler) ListAllNotifications(w http.ResponseWriter, r *http.Request) {
	scope, agentSlug, unreadOnly, limit, offset := parseNotificationQueryParams(r)

	notifications, err := h.queries.ListNotifications(r.Context(), nil, scope, agentSlug, unreadOnly, limit, offset)
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

// ListProjectNotifications lists notifications for a project
func (h *NotificationQueriesHandler) ListProjectNotifications(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProjectID(mux.Vars(r)["id"])
	scope, agentSlug, unreadOnly, limit, offset := parseNotificationQueryParams(r)

	notifications, err := h.queries.ListNotifications(r.Context(), &pid, scope, agentSlug, unreadOnly, limit, offset)
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

// GetGlobalUnreadCount returns the count of all unread notifications
func (h *NotificationQueriesHandler) GetGlobalUnreadCount(w http.ResponseWriter, r *http.Request) {
	scope, agentSlug, _, _, _ := parseNotificationQueryParams(r)

	count, err := h.queries.GetNotificationUnreadCount(r.Context(), nil, scope, agentSlug)
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

// GetProjectUnreadCount returns the count of unread notifications for a project
func (h *NotificationQueriesHandler) GetProjectUnreadCount(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProjectID(mux.Vars(r)["id"])
	scope, agentSlug, _, _, _ := parseNotificationQueryParams(r)

	count, err := h.queries.GetNotificationUnreadCount(r.Context(), &pid, scope, agentSlug)
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
