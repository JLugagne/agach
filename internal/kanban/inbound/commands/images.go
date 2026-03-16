package commands

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	maxImageSize = 10 << 20 // 10 MB
)

var allowedMIMETypes = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/gif":  "gif",
	"image/webp": "webp",
}

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

// UploadImage handles multipart image upload and saves it to the project's work_dir
func (h *ImageCommandsHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
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

	if project.WorkDir == "" {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidImageRequest, fmt.Errorf("project has no work_dir configured")))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidImageRequest, fmt.Errorf("file too large or invalid multipart form: %w", err)))
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidImageRequest, fmt.Errorf("missing image field: %w", err)))
		return
	}
	defer file.Close()

	// Detect MIME type from first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		h.controller.SendError(w, r, fmt.Errorf("failed to read file: %w", err))
		return
	}
	detectedMIME := http.DetectContentType(buf[:n])

	// Fall back to Content-Type header if detection is ambiguous
	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedMIMETypes[detectedMIME]
	if !ok {
		// Try the declared content type
		ext, ok = allowedMIMETypes[contentType]
		if !ok {
			h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidImageRequest, fmt.Errorf("unsupported MIME type: %s", detectedMIME)))
			return
		}
	}

	// Build destination directory: <work_dir>/.agach-mcp
	destDir := filepath.Join(project.WorkDir, ".agach-mcp")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to create image directory: %w", err))
		return
	}

	filename := uuid.New().String() + "." + ext
	destPath := filepath.Join(destDir, filename)

	out, err := os.Create(destPath)
	if err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to create image file: %w", err))
		return
	}
	defer out.Close()

	// Write the already-read bytes first, then copy the remainder
	if _, err := out.Write(buf[:n]); err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to write image: %w", err))
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to write image: %w", err))
		return
	}

	url := fmt.Sprintf("/api/projects/%s/images/%s", string(projectID), filename)
	h.controller.SendSuccess(w, r, map[string]string{"url": url})
}

// ServeImage serves an image file from the project's work_dir
func (h *ImageCommandsHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	filename := mux.Vars(r)["filename"]

	// Reject any path traversal attempts
	if strings.ContainsAny(filename, "/\\") {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidImageRequest, fmt.Errorf("invalid filename")))
		return
	}

	project, err := h.queries.GetProject(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	if project.WorkDir == "" {
		h.controller.SendFail(w, r, nil, errors.Join(pkgkanban.ErrInvalidImageRequest, fmt.Errorf("project has no work_dir configured")))
		return
	}

	filePath := filepath.Join(project.WorkDir, ".agach-mcp", filename)
	http.ServeFile(w, r, filePath)
}
