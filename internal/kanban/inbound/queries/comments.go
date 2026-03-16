package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// CommentQueriesHandler handles comment read operations
type CommentQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewCommentQueriesHandler creates a new comment queries handler
func NewCommentQueriesHandler(queries service.Queries, ctrl *controller.Controller) *CommentQueriesHandler {
	return &CommentQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers comment query routes
func (h *CommentQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/comments", h.ListComments).Methods("GET")
}

// ListComments lists all comments for a task
func (h *CommentQueriesHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	comments, err := h.queries.ListComments(r.Context(), projectID, taskID, 0, 0)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicComments(comments))
}
