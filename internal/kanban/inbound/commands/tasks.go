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

// TaskCommandsHandler handles task write operations
type TaskCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewTaskCommandsHandler creates a new task commands handler
func NewTaskCommandsHandler(commands service.Commands, ctrl *controller.Controller, hub *websocket.Hub) *TaskCommandsHandler {
	return &TaskCommandsHandler{
		commands:   commands,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers task command routes
func (h *TaskCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/tasks", h.CreateTask).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}", h.UpdateTask).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}", h.DeleteTask).Methods("DELETE")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/move", h.MoveTask).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/move-to-project", h.MoveTaskToProject).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/complete", h.CompleteTask).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/unblock", h.UnblockTask).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/wont-do", h.WontDo).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/approve-wont-do", h.ApproveWontDo).Methods("POST")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/reject-wont-do", h.RejectWontDo).Methods("POST")
}

// CreateTask creates a new task
func (h *TaskCommandsHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	var req pkgkanban.CreateTaskRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidTaskRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidTaskRequest, err))
		return
	}

	priority := converters.ToDomainPriority(req.Priority)

	task, err := h.commands.CreateTask(
		r.Context(),
		projectID,
		req.Title,
		req.Summary,
		req.Description,
		priority,
		req.CreatedByRole,
		req.CreatedByAgent,
		req.AssignedRole,
		req.ContextFiles,
		req.Tags,
		req.EstimatedEffort,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_created event
	h.hub.Broadcast(websocket.Event{
		Type:      "task_created",
		ProjectID: string(projectID),
		Data:      converters.ToPublicTask(task),
	})

	h.controller.SendSuccess(w, r, converters.ToPublicTask(task))
}

// UpdateTask updates an existing task
func (h *TaskCommandsHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	var req pkgkanban.UpdateTaskRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidTaskRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidTaskRequest, err))
		return
	}

	var priority *domain.Priority
	if req.Priority != nil {
		p := converters.ToDomainPriority(*req.Priority)
		priority = &p
	}

	err := h.commands.UpdateTask(
		r.Context(),
		projectID,
		taskID,
		req.Title,
		req.Description,
		req.AssignedRole,
		req.EstimatedEffort,
		req.Resolution,
		priority,
		req.ContextFiles,
		req.Tags,
		nil,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_updated event
	h.hub.Broadcast(websocket.Event{
		Type:      "task_updated",
		ProjectID: string(projectID),
		Data:      map[string]string{"task_id": string(taskID)},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "task updated"})
}

// DeleteTask deletes a task
func (h *TaskCommandsHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	err := h.commands.DeleteTask(r.Context(), projectID, taskID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_deleted event
	h.hub.Broadcast(websocket.Event{
		Type:      "task_deleted",
		ProjectID: string(projectID),
		Data:      map[string]string{"task_id": string(taskID)},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "task deleted"})
}

// MoveTask moves a task to a different column
func (h *TaskCommandsHandler) MoveTask(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	var req pkgkanban.MoveTaskRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidTaskRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidTaskRequest, err))
		return
	}

	targetColumn := domain.ColumnSlug(req.TargetColumn)

	err := h.commands.MoveTask(r.Context(), projectID, taskID, targetColumn)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_moved event
	h.hub.Broadcast(websocket.Event{
		Type:      "task_moved",
		ProjectID: string(projectID),
		Data: map[string]string{
			"task_id":       string(taskID),
			"target_column": req.TargetColumn,
			"reason":        req.Reason,
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "task moved"})
}

// CompleteTask marks a task as completed
func (h *TaskCommandsHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	var req pkgkanban.CompleteTaskRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidTaskRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidTaskRequest, err))
		return
	}

	err := h.commands.CompleteTask(
		r.Context(),
		projectID,
		taskID,
		req.CompletionSummary,
		req.FilesModified,
		req.CompletedByAgent,
		nil,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_completed event
	h.hub.Broadcast(websocket.Event{
		Type:      "task_completed",
		ProjectID: string(projectID),
		Data: map[string]interface{}{
			"task_id":            string(taskID),
			"completion_summary": req.CompletionSummary,
			"files_modified":     req.FilesModified,
			"completed_by_agent": req.CompletedByAgent,
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "task completed"})
}

// UnblockTask unblocks a task (human only)
func (h *TaskCommandsHandler) UnblockTask(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	err := h.commands.UnblockTask(r.Context(), projectID, taskID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_unblocked event
	h.hub.Broadcast(websocket.Event{
		Type:      "task_unblocked",
		ProjectID: string(projectID),
		Data:      map[string]string{"task_id": string(taskID)},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "task unblocked"})
}

// WontDo marks a task as won't do (human directly)
func (h *TaskCommandsHandler) WontDo(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	var req pkgkanban.RequestWontDoRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidTaskRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidTaskRequest, err))
		return
	}

	// Human directly marks won't do - we approve immediately
	err := h.commands.RequestWontDo(r.Context(), projectID, taskID, req.WontDoReason, req.WontDoRequestedBy)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Then approve it
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
			"reason":  req.WontDoReason,
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "task marked as won't do"})
}

// ApproveWontDo approves an agent's won't do request
func (h *TaskCommandsHandler) ApproveWontDo(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	err := h.commands.ApproveWontDo(r.Context(), projectID, taskID)
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
		Data:      map[string]string{"task_id": string(taskID)},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "won't do approved"})
}

// MoveTaskToProject moves a task to a different project
func (h *TaskCommandsHandler) MoveTaskToProject(w http.ResponseWriter, r *http.Request) {
	sourceProjectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	var req pkgkanban.MoveTaskToProjectRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidTaskRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidTaskRequest, err))
		return
	}

	targetProjectID := domain.ProjectID(req.TargetProjectID)

	err := h.commands.MoveTaskToProject(r.Context(), sourceProjectID, taskID, targetProjectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_deleted on source project
	h.hub.Broadcast(websocket.Event{
		Type:      "task_deleted",
		ProjectID: string(sourceProjectID),
		Data:      map[string]string{"task_id": string(taskID)},
	})
	// Broadcast task_created on target project so the UI refreshes
	h.hub.Broadcast(websocket.Event{
		Type:      "task_created",
		ProjectID: string(targetProjectID),
		Data: map[string]string{
			"source_project_id": string(sourceProjectID),
			"source_task_id":    string(taskID),
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "task moved to project"})
}

// RejectWontDo rejects an agent's won't do request
func (h *TaskCommandsHandler) RejectWontDo(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	var req pkgkanban.RejectWontDoRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgkanban.ErrInvalidTaskRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidTaskRequest, err))
		return
	}

	err := h.commands.RejectWontDo(r.Context(), projectID, taskID, req.Reason)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast wont_do_rejected event
	h.hub.Broadcast(websocket.Event{
		Type:      "wont_do_rejected",
		ProjectID: string(projectID),
		Data: map[string]string{
			"task_id":          string(taskID),
			"rejection_reason": req.Reason,
		},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "won't do rejected"})
}
