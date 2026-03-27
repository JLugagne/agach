package queries

import (
	"net/http"
	"strconv"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/gorilla/mux"
)

const (
	defaultCommentLimit = 100
	maxCommentLimit     = 500
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

// ListComments lists comments for a task with pagination.
func (h *CommentQueriesHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	limit := defaultCommentLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > maxCommentLimit {
				l = maxCommentLimit
			}
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	comments, err := h.queries.ListComments(r.Context(), projectID, taskID, limit, offset)
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
