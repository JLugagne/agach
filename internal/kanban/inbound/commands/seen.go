package commands

import (
	"net/http"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
)

// SeenCommandsHandler handles task seen write operations
type SeenCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewSeenCommandsHandler creates a new seen commands handler
func NewSeenCommandsHandler(commands service.Commands, ctrl *controller.Controller, hub *websocket.Hub) *SeenCommandsHandler {
	return &SeenCommandsHandler{
		commands:   commands,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers seen command routes
func (h *SeenCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/seen", h.MarkTaskSeen).Methods("POST")
}

// MarkTaskSeen marks a task as seen (idempotent — only records the first view)
func (h *SeenCommandsHandler) MarkTaskSeen(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	err := h.commands.MarkTaskSeen(r.Context(), projectID, taskID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast task_seen event
	h.hub.Broadcast(websocket.Event{
		Type:      "task_seen",
		ProjectID: string(projectID),
		Data: map[string]interface{}{
			"project_id": string(projectID),
			"task_id":    string(taskID),
			"seen_at":    time.Now().UTC(),
		},
	})

	w.WriteHeader(http.StatusNoContent)
}
