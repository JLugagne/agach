package queries

import (
	"net/http"
	"strconv"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
	"github.com/gorilla/mux"
)

const (
	maxSearchLimit = 1000
	maxNextCount   = 100
	maxDoneSince   = 8760 * time.Hour
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
	router.HandleFunc("/api/projects/{id}/tasks/search", h.SearchTasks).Methods("GET")
	router.HandleFunc("/api/projects/{id}/tasks/{taskId}", h.GetTask).Methods("GET")
	router.HandleFunc("/api/projects/{id}/board", h.GetBoard).Methods("GET")
	router.HandleFunc("/api/projects/{id}/columns", h.ListColumns).Methods("GET")
	router.HandleFunc("/api/projects/{id}/next-tasks", h.GetNextTasks).Methods("GET")
	router.HandleFunc("/api/projects/{id}/wip-slots", h.GetWIPSlots).Methods("GET")
	router.HandleFunc("/api/projects/{projectId}/agents/{slug}/tasks", h.ListTasksByAgent).Methods("GET")
}

// SearchTasks searches tasks with optional filters and a limit parameter.
func (h *TaskQueriesHandler) SearchTasks(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	filters := tasks.TaskFilters{}
	if q := r.URL.Query().Get("q"); q != "" {
		filters.Search = q
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			if limit > maxSearchLimit {
				limit = maxSearchLimit
			}
			filters.Limit = limit
		}
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

	publicTasks := converters.ToPublicTasksWithDetails(taskList)
	h.controller.SendSuccess(w, r, publicTasks)
}

// ListTasks lists tasks with optional filters
func (h *TaskQueriesHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	includeChildren := r.URL.Query().Get("include_children") == "true"

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

	// Fetch tasks for the parent project
	taskList, err := h.queries.ListTasks(r.Context(), projectID, filters)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	publicTasks := converters.ToPublicTasksWithDetails(taskList)
	// Tag parent project tasks with their project_id
	for i := range publicTasks {
		publicTasks[i].ProjectID = string(projectID)
	}

	if includeChildren {
		children, err := h.queries.ListSubProjects(r.Context(), projectID)
		if err != nil {
			h.controller.SendError(w, r, err)
			return
		}
		for _, child := range children {
			childTasks, err := h.queries.ListTasks(r.Context(), child.ID, filters)
			if err != nil {
				continue
			}
			childPublic := converters.ToPublicTasksWithDetails(childTasks)
			for i := range childPublic {
				childPublic[i].ProjectID = string(child.ID)
				childPublic[i].ProjectName = child.Name
			}
			publicTasks = append(publicTasks, childPublic...)
		}
	}

	h.controller.SendSuccess(w, r, publicTasks)
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
			if d > maxDoneSince {
				http.Error(w, `{"status":"fail","data":{"error":"done_since exceeds maximum allowed duration of 8760h"}}`, http.StatusBadRequest)
				return
			}
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

// GetNextTasks returns up to count ready tasks as [{id, title, role, project_id, session_id}]
func (h *TaskQueriesHandler) GetNextTasks(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	count := 1
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if n, err := strconv.Atoi(countStr); err == nil && n > 0 {
			count = n
		}
	}
	if count > maxNextCount {
		count = maxNextCount
	}

	role := r.URL.Query().Get("role")
	includeSubprojects := r.URL.Query().Get("include_subprojects") == "true"

	type nextTaskResult struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Role      string `json:"role"`
		ProjectID string `json:"project_id"`
		SessionID string `json:"session_id"`
	}

	var results []nextTaskResult

	if !includeSubprojects {
		// Single project query
		taskList, err := h.queries.GetNextTasks(r.Context(), projectID, role, count, nil)
		if err != nil {
			if domain.IsDomainError(err) {
				h.controller.SendFail(w, r, nil, err)
			} else {
				h.controller.SendError(w, r, err)
			}
			return
		}

		results = make([]nextTaskResult, len(taskList))
		for i, t := range taskList {
			results[i] = nextTaskResult{
				ID:        string(t.ID),
				Title:     t.Title,
				Role:      t.AssignedRole,
				ProjectID: string(projectID),
				SessionID: t.SessionID,
			}
		}
	} else {
		// Multi-project query: main project + all sub-projects
		allTasks := make(map[string]domain.Task)
		taskToProjectMap := make(map[string]domain.ProjectID)

		// Get main project tasks
		taskList, err := h.queries.GetNextTasks(r.Context(), projectID, role, count*10, nil)
		if err != nil {
			if domain.IsDomainError(err) {
				h.controller.SendFail(w, r, nil, err)
			} else {
				h.controller.SendError(w, r, err)
			}
			return
		}

		for _, t := range taskList {
			allTasks[string(t.ID)] = t
			taskToProjectMap[string(t.ID)] = projectID
		}

		// Get sub-projects (fetch once)
		subProjects, err := h.queries.ListSubProjects(r.Context(), projectID)
		if err == nil {
			for _, subProj := range subProjects {
				subTaskList, err := h.queries.GetNextTasks(r.Context(), subProj.ID, role, count*10, nil)
				if err == nil {
					for _, t := range subTaskList {
						allTasks[string(t.ID)] = t
						taskToProjectMap[string(t.ID)] = subProj.ID
					}
				}
			}
		}

		// Convert map to slice and sort by priority score descending, then by created_at ascending
		taskSlice := make([]domain.Task, 0, len(allTasks))
		for _, t := range allTasks {
			taskSlice = append(taskSlice, t)
		}

		for i := 0; i < len(taskSlice); i++ {
			for j := i + 1; j < len(taskSlice); j++ {
				if taskSlice[j].PriorityScore > taskSlice[i].PriorityScore ||
					(taskSlice[j].PriorityScore == taskSlice[i].PriorityScore && taskSlice[j].CreatedAt.Before(taskSlice[i].CreatedAt)) {
					taskSlice[i], taskSlice[j] = taskSlice[j], taskSlice[i]
				}
			}
		}

		// Limit to requested count
		if len(taskSlice) > count {
			taskSlice = taskSlice[:count]
		}

		results = make([]nextTaskResult, len(taskSlice))
		for i, t := range taskSlice {
			projID := taskToProjectMap[string(t.ID)]
			if projID == "" {
				projID = projectID
			}
			results[i] = nextTaskResult{
				ID:        string(t.ID),
				Title:     t.Title,
				Role:      t.AssignedRole,
				ProjectID: string(projID),
				SessionID: t.SessionID,
			}
		}
	}

	h.controller.SendSuccess(w, r, results)
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

// GetWIPSlots gets the WIP slots information for a project
func (h *TaskQueriesHandler) GetWIPSlots(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	info, err := h.queries.GetWIPSlots(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, info)
}

// ListTasksByAgent returns all tasks assigned to a given agent within a project
func (h *TaskQueriesHandler) ListTasksByAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := domain.ProjectID(vars["projectId"])
	agentSlug := vars["slug"]

	if agentSlug == "" {
		http.Error(w, `{"status":"fail","data":{"error":"agent slug is required"}}`, http.StatusBadRequest)
		return
	}

	taskList, err := h.queries.GetProjectTasksByAgent(r.Context(), projectID, agentSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, pkgkanban.TasksByAgentResponse{
		AgentSlug: agentSlug,
		TaskCount: len(taskList),
		Tasks:     converters.ToPublicTasks(taskList),
	})
}
