package queries

import (
	"net/http"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
	"github.com/gorilla/mux"
)

// TaskQueriesHandler handles task read operations
type TaskQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewTaskQueriesHandler creates a new task queries handler
func NewTaskQueriesHandler(queries service.Queries, ctrl *controller.Controller) *TaskQueriesHandler {
	return &TaskQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers task query routes
func (h *TaskQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/tasks", h.ListTasks).Methods("GET")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}", h.GetTask).Methods("GET")
	router.HandleFunc("/api/projects/{id}/board", h.GetBoard).Methods("GET")
	router.HandleFunc("/api/projects/{id}/columns", h.ListColumns).Methods("GET")
}

// ListTasks lists tasks with optional filters
func (h *TaskQueriesHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	// Parse query parameters for filters
	filters := tasks.TaskFilters{}

	if col := r.URL.Query().Get("column"); col != "" {
		colSlug := domain.ColumnSlug(col)
		filters.ColumnSlug = &colSlug
	}

	if role := r.URL.Query().Get("assigned_role"); role != "" {
		filters.AssignedRole = &role
	}

	if prio := r.URL.Query().Get("priority"); prio != "" {
		p := domain.Priority(prio)
		filters.Priority = &p
	}

	if tag := r.URL.Query().Get("tag"); tag != "" {
		filters.Tag = &tag
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = search
	}

	taskList, err := h.queries.ListTasks(r.Context(), projectID, filters)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicTasksWithDetails(taskList))
}

// GetTask gets a single task
func (h *TaskQueriesHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	taskID := domain.TaskID(mux.Vars(r)["taskId"])

	task, err := h.queries.GetTask(r.Context(), projectID, taskID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicTask(*task))
}

// GetBoard gets the full kanban board with columns and tasks
func (h *TaskQueriesHandler) GetBoard(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	includeChildren := r.URL.Query().Get("include_children") == "true"
	searchQuery := r.URL.Query().Get("search")

	// Parse optional done_since filter (e.g. "24h", "72h", "168h")
	var doneSince *time.Duration
	if ds := r.URL.Query().Get("done_since"); ds != "" {
		d, err := time.ParseDuration(ds)
		if err == nil {
			doneSince = &d
		}
	}

	columns, err := h.queries.ListColumns(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Build list of projects to aggregate: parent + children
	type projectEntry struct {
		ID   domain.ProjectID
		Name string
	}
	project, err := h.queries.GetProject(r.Context(), projectID)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	projects := []projectEntry{{ID: projectID, Name: project.Name}}

	if includeChildren {
		children, err := h.queries.ListSubProjects(r.Context(), projectID)
		if err == nil {
			for _, child := range children {
				projects = append(projects, projectEntry{ID: child.ID, Name: child.Name})
			}
		}
	}

	// For each column, get tasks from all projects
	boardColumns := make([]pkgkanban.ColumnWithTasksResponse, len(columns))
	for i, col := range columns {
		colSlug := col.Slug
		var allTasks []pkgkanban.TaskWithDetailsResponse

		for _, proj := range projects {
			filters := tasks.TaskFilters{
				ColumnSlug: &colSlug,
				Search:     searchQuery,
			}

			// Apply time filter to done column
			if colSlug == domain.ColumnDone && doneSince != nil {
				since := time.Now().Add(-*doneSince)
				filters.UpdatedSince = &since
			}

			taskList, err := h.queries.ListTasks(r.Context(), proj.ID, filters)
			if err != nil {
				if domain.IsDomainError(err) {
					h.controller.SendFail(w, r, nil, err)
				} else {
					h.controller.SendError(w, r, err)
				}
				return
			}

			publicTasks := converters.ToPublicTasksWithDetails(taskList)
			// Tag tasks with their project info (skip for the parent project itself)
			if proj.ID != projectID {
				for j := range publicTasks {
					publicTasks[j].ProjectID = string(proj.ID)
					publicTasks[j].ProjectName = proj.Name
				}
			}
			allTasks = append(allTasks, publicTasks...)
		}

		boardColumns[i] = pkgkanban.ColumnWithTasksResponse{
			ColumnResponse: converters.ToPublicColumn(col),
			Tasks:          allTasks,
		}
	}

	h.controller.SendSuccess(w, r, pkgkanban.BoardResponse{
		Columns: boardColumns,
	})
}

// ListColumns lists all columns for a project
func (h *TaskQueriesHandler) ListColumns(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	columns, err := h.queries.ListColumns(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicColumns(columns))
}
