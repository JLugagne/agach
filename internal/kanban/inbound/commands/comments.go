package commands

import (
	"errors"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
)

// CommentCommandsHandler handles comment write operations
type CommentCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewCommentCommandsHandler creates a new comment commands handler
func NewCommentCommandsHandler(commands service.Commands, ctrl *controller.Controller, hub *websocket.Hub) *CommentCommandsHandler {
	return &CommentCommandsHandler{
		commands:   commands,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers comment command routes
func (h *CommentCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/comments", h.CreateComment).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/comments/{commentId}", h.UpdateComment).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/comments/{commentId}", h.DeleteComment).Methods("DELETE")
}

// CreateComment creates a new comment
func (h *CommentCommandsHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	var req pkgkanban.CreateCommentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidCommentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidCommentRequest, err))
		return
	}

	// From web UI, author type is always human
	comment, err := h.commands.CreateComment(
		r.Context(),
		projectID,
		taskID,
		req.AuthorRole,
		req.AuthorName,
		domain.AuthorTypeHuman,
		req.Content,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// If MarkAsWontDo is true and task is in todo, move to wont_do
	if req.MarkAsWontDo {
		// Request won't do with comment content as reason, then approve
		err = h.commands.RequestWontDo(r.Context(), projectID, taskID, req.Content, req.AuthorName)
		if err != nil {
			if domain.IsDomainError(err) {
				h.controller.SendFail(w, r, nil, err)
			} else {
				h.controller.SendError(w, r, err)
			}
			return
		}

		err = h.commands.ApproveWontDo(r.Context(), projectID, taskID)
		if err != nil {
			if domain.IsDomainError(err) {
				h.controller.SendFail(w, r, nil, err)
			} else {
				h.controller.SendError(w, r, err)
			}
			return
		}

		// Broadcast task_wont_do event
		h.hub.Broadcast(websocket.Event{
			Type:      "task_wont_do",
			ProjectID: string(projectID),
			Data: map[string]string{
				"task_id": string(taskID),
				"reason":  req.Content,
			},
		})
	}

	// Broadcast comment_added event
	h.hub.Broadcast(websocket.Event{
		Type:      "comment_added",
		ProjectID: string(projectID),
		Data:      converters.ToPublicComment(comment),
	})

	h.controller.SendSuccess(w, r, converters.ToPublicComment(comment))
}

// UpdateComment updates an existing comment
func (h *CommentCommandsHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	commentID := domain.CommentID(mux.Vars(r)["commentId"])

	var req pkgkanban.UpdateCommentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidCommentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidCommentRequest, err))
		return
	}

	err := h.commands.UpdateComment(r.Context(), projectID, commentID, req.Content)
	if err != nil {
		if domain.IsDomainError(err) {
			statusCode := http.StatusForbidden
			h.controller.SendFail(w, r, &statusCode, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast comment_edited event
	h.hub.Broadcast(websocket.Event{
		Type:      "comment_edited",
		ProjectID: string(projectID),
		Data: map[string]string{
			"comment_id":  string(commentID),
			"new_content": req.Content,
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "comment updated"})
}

// DeleteComment deletes a comment
func (h *CommentCommandsHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	commentID := domain.CommentID(mux.Vars(r)["commentId"])

	err := h.commands.DeleteComment(r.Context(), projectID, commentID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{"message": "comment deleted"})
}
