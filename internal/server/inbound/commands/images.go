package commands

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/gorilla/mux"
)

// ImageCommandsHandler handles image upload and serving
type ImageCommandsHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

// NewImageCommandsHandler creates a new image commands handler
func NewImageCommandsHandler(queries service.Queries, ctrl *controller.Controller) *ImageCommandsHandler {
	return &ImageCommandsHandler{
		queries:    queries,
		controller: ctrl,
	}
}

// RegisterRoutes registers image routes
func (h *ImageCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/images", h.UploadImage).Methods("POST")
	router.HandleFunc("/api/projects/{id}/images/{filename}", h.ServeImage).Methods("GET")
}

// UploadImage handles multipart image upload.
// NOTE: Image upload is currently unavailable as the work_dir feature has been removed.
// This feature will need to be reworked to use a different storage mechanism.
func (h *ImageCommandsHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidImageRequest, fmt.Errorf("image upload is not available: project work_dir has been removed")))
}

// ServeImage serves an image file.
// NOTE: Image serving is currently unavailable as the work_dir feature has been removed.
func (h *ImageCommandsHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidImageRequest, fmt.Errorf("image serving is not available: project work_dir has been removed")))
}
