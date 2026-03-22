package queries

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// DockerfileQueriesHandler handles dockerfile read operations
type DockerfileQueriesHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewDockerfileQueriesHandler creates a new dockerfile queries handler
func NewDockerfileQueriesHandler(queries service.Queries, ctrl *controller.Controller) *DockerfileQueriesHandler {
	return &DockerfileQueriesHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers dockerfile query routes
func (h *DockerfileQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/dockerfiles", h.ListDockerfiles).Methods("GET")
	router.HandleFunc("/api/dockerfiles/{id}", h.GetDockerfile).Methods("GET")
	router.HandleFunc("/api/projects/{id}/dockerfile", h.GetProjectDockerfile).Methods("GET")
}

// ListDockerfiles lists all dockerfiles
func (h *DockerfileQueriesHandler) ListDockerfiles(w http.ResponseWriter, r *http.Request) {
	list, err := h.queries.ListDockerfiles(r.Context())
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}
	h.controller.SendSuccess(w, r, converters.ToPublicDockerfiles(list))
}

// GetDockerfile gets a single dockerfile by ID
func (h *DockerfileQueriesHandler) GetDockerfile(w http.ResponseWriter, r *http.Request) {
	id := domain.DockerfileID(mux.Vars(r)["id"])

	dockerfile, err := h.queries.GetDockerfile(r.Context(), id)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicDockerfile(*dockerfile))
}

// GetProjectDockerfile gets the dockerfile assigned to a project
func (h *DockerfileQueriesHandler) GetProjectDockerfile(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	dockerfile, err := h.queries.GetProjectDockerfile(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	if dockerfile == nil {
		h.controller.SendSuccess(w, r, nil)
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicDockerfile(*dockerfile))
}
