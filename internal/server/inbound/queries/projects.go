package queries

import (
	"net/http"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/gorilla/mux"
)

// ProjectQueriesHandler handles project read operations
type ProjectQueriesHandler struct {
	queries      service.Queries
	controller   *controller.Controller
	teamResolver TeamIDResolver
}

// NewProjectQueriesHandler creates a new project queries handler
func NewProjectQueriesHandler(queries service.Queries, ctrl *controller.Controller, teamResolver TeamIDResolver) *ProjectQueriesHandler {
	return &ProjectQueriesHandler{
		queries:      queries,
		controller:   ctrl,
		teamResolver: teamResolver,
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

// ListProjects lists root projects with summaries, filtered by the caller's access.
// Admins see all projects. Non-admins see only projects they have direct or team-based access to.
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

	// Filter by access if the caller is a non-admin user.
	actor, ok := r.Context().Value(middleware.ActorContextKey).(identitydomain.Actor)
	if ok && !actor.IsAdmin() && h.teamResolver != nil {
		teamIDs, _ := h.teamResolver.GetUserTeamIDs(r.Context(), actor.UserID)
		teamIDStrings := make([]string, len(teamIDs))
		for i, tid := range teamIDs {
			teamIDStrings[i] = tid.String()
		}

		accessibleIDs, err := h.queries.ListAccessibleProjectIDs(r.Context(), actor.UserID.String(), teamIDStrings)
		if err == nil {
			allowed := make(map[domain.ProjectID]bool, len(accessibleIDs))
			for _, id := range accessibleIDs {
				allowed[id] = true
			}
			filtered := make([]domain.ProjectWithSummary, 0, len(projects))
			for _, p := range projects {
				if allowed[p.ID] {
					filtered = append(filtered, p)
				}
			}
			projects = filtered
		}
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
