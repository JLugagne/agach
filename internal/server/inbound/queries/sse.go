package queries

import (
	"fmt"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/sse"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// SSEHandler serves Server-Sent Events for project task notifications
type SSEHandler struct {
	sseHub     *sse.Hub
	queries    service.Queries
	controller *controller.Controller
}

func NewSSEHandler(sseHub *sse.Hub, ctrl ...*controller.Controller) *SSEHandler {
	h := &SSEHandler{sseHub: sseHub}
	if len(ctrl) > 0 && ctrl[0] != nil {
		h.controller = ctrl[0]
	} else {
		h.controller = controller.NewController(logrus.New())
	}
	return h
}

func NewSSEHandlerWithQueries(sseHub *sse.Hub, queries service.Queries, ctrl *controller.Controller) *SSEHandler {
	return &SSEHandler{sseHub: sseHub, queries: queries, controller: ctrl}
}

// checkSSEAccess verifies the caller has access to the given project before allowing SSE subscription.
func (h *SSEHandler) checkSSEAccess(r *http.Request, projectID domain.ProjectID) bool {
	if h.queries == nil {
		return true
	}
	ok, err := h.queries.HasProjectAccess(r.Context(), projectID, "", nil)
	if err != nil || !ok {
		return false
	}
	return true
}

func (h *SSEHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/sse", h.ServeSSE).Methods("GET")
}

func (h *SSEHandler) ServeSSE(w http.ResponseWriter, r *http.Request) {
	rawID := mux.Vars(r)["id"]
	projectID, err := domain.ParseProjectID(rawID)
	if err != nil {
		h.controller.SendFail(w, r, nil, &apierror.Error{Code: "INVALID_PROJECT_ID", Message: "invalid project ID"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.controller.SendError(w, r, fmt.Errorf("response writer does not support flushing"))
		return
	}

	ch, unsubscribe := h.sseHub.Subscribe(string(projectID))
	defer unsubscribe()

	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
