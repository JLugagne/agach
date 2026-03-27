package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/gorilla/mux"
)

// DependencyQueriesHandler handles dependency read operations
type DependencyQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewDependencyQueriesHandler creates a new dependency queries handler
func NewDependencyQueriesHandler(queries service.Queries, ctrl *controller.Controller) *DependencyQueriesHandler {
	return &DependencyQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers dependency query routes
func (h *DependencyQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/dependencies", h.ListDependencies).Methods("GET")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}/dependents", h.ListDependents).Methods("GET")
}

// ListDependencies returns the task objects that this task depends on
func (h *DependencyQueriesHandler) ListDependencies(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	taskList, err := h.queries.GetDependencyTasks(r.Context(), projectID, taskID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	result := make([]pkgserver.TaskResponse, len(taskList))
	for i, t := range taskList {
		result[i] = converters.ToPublicTask(t)
	}

	h.controller.SendSuccess(w, r, result)
}

// ListDependents returns the task objects that depend on this task
func (h *DependencyQueriesHandler) ListDependents(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	taskList, err := h.queries.GetDependentTasks(r.Context(), projectID, taskID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	result := make([]pkgserver.TaskResponse, len(taskList))
	for i, t := range taskList {
		result[i] = converters.ToPublicTask(t)
	}

	h.controller.SendSuccess(w, r, result)
}
