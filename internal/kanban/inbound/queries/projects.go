package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// ProjectQueriesHandler handles project read operations
type ProjectQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewProjectQueriesHandler creates a new project queries handler
func NewProjectQueriesHandler(queries service.Queries, ctrl *controller.Controller) *ProjectQueriesHandler {
	return &ProjectQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers project query routes
func (h *ProjectQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects", h.ListProjects).Methods("GET")
	router.HandleFunc("/api/projects/{id}", h.GetProject).Methods("GET")
	router.HandleFunc("/api/projects/{id}/info", h.GetProjectInfo).Methods("GET")
	router.HandleFunc("/api/projects/{id}/summary", h.GetProjectSummary).Methods("GET")
	router.HandleFunc("/api/projects/{id}/children", h.ListSubProjects).Methods("GET")
}

// ListProjects lists all root projects with summaries.
func (h *ProjectQueriesHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.queries.ListProjectsWithSummary(r.Context())
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, projects)
}

// GetProject gets a single project
func (h *ProjectQueriesHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	project, err := h.queries.GetProject(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicProject(*project))
}

// GetProjectSummary gets project task summary
func (h *ProjectQueriesHandler) GetProjectSummary(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	summary, err := h.queries.GetProjectSummary(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicProjectSummary(*summary))
}

// GetProjectInfo gets complete project information
func (h *ProjectQueriesHandler) GetProjectInfo(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	info, err := h.queries.GetProjectInfo(r.Context(), projectID)
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

// ListSubProjects lists sub-projects of a parent with summaries
func (h *ProjectQueriesHandler) ListSubProjects(w http.ResponseWriter, r *http.Request) {
	parentID := domain.ProjectID(mux.Vars(r)["id"])

	projects, err := h.queries.ListSubProjectsWithSummary(r.Context(), parentID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, projects)
}
