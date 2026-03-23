package commands

import (
	"errors"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/gorilla/mux"
)

// DockerfileCommandsHandler handles dockerfile write operations
type DockerfileCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
}

// NewDockerfileCommandsHandler creates a new dockerfile commands handler
func NewDockerfileCommandsHandler(commands service.Commands, ctrl *controller.Controller) *DockerfileCommandsHandler {
	return &DockerfileCommandsHandler{
		commands:   commands,
		controller: ctrl,
	}
}

// RegisterRoutes registers dockerfile command routes
func (h *DockerfileCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/dockerfiles", h.CreateDockerfile).Methods("POST")
	router.HandleFunc("/api/dockerfiles/{id}", h.UpdateDockerfile).Methods("PATCH")
	router.HandleFunc("/api/dockerfiles/{id}", h.DeleteDockerfile).Methods("DELETE")
	router.HandleFunc("/api/projects/{id}/dockerfile", h.SetProjectDockerfile).Methods("PUT")
	router.HandleFunc("/api/projects/{id}/dockerfile", h.ClearProjectDockerfile).Methods("DELETE")
}

// CreateDockerfile creates a new dockerfile
func (h *DockerfileCommandsHandler) CreateDockerfile(w http.ResponseWriter, r *http.Request) {
	var req pkgserver.CreateDockerfileRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidDockerfileRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidDockerfileRequest, err))
		return
	}

	dockerfile, err := h.commands.CreateDockerfile(
		r.Context(),
		req.Slug,
		req.Name,
		req.Description,
		req.Version,
		req.Content,
		req.IsLatest,
		req.SortOrder,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, converters.ToPublicDockerfile(dockerfile))
}

// UpdateDockerfile updates an existing dockerfile
func (h *DockerfileCommandsHandler) UpdateDockerfile(w http.ResponseWriter, r *http.Request) {
	id := domain.DockerfileID(mux.Vars(r)["id"])

	var req pkgserver.UpdateDockerfileRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidDockerfileRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidDockerfileRequest, err))
		return
	}

	if err := h.commands.UpdateDockerfile(r.Context(), id, req.Name, req.Description, req.Content, req.IsLatest, req.SortOrder); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, nil)
}

// DeleteDockerfile deletes a dockerfile
func (h *DockerfileCommandsHandler) DeleteDockerfile(w http.ResponseWriter, r *http.Request) {
	id := domain.DockerfileID(mux.Vars(r)["id"])

	if err := h.commands.DeleteDockerfile(r.Context(), id); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, nil)
}

// SetProjectDockerfile assigns a dockerfile to a project
func (h *DockerfileCommandsHandler) SetProjectDockerfile(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	var req pkgserver.SetProjectDockerfileRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidDockerfileRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidDockerfileRequest, err))
		return
	}

	if err := h.commands.SetProjectDockerfile(r.Context(), projectID, domain.DockerfileID(req.DockerfileID)); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, nil)
}

// ClearProjectDockerfile removes the dockerfile assignment from a project
func (h *DockerfileCommandsHandler) ClearProjectDockerfile(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	if err := h.commands.ClearProjectDockerfile(r.Context(), projectID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, nil)
}
