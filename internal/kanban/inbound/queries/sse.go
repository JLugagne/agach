package queries

import (
	"fmt"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/pkg/sse"
	"github.com/gorilla/mux"
)

// SSEHandler serves Server-Sent Events for project task notifications
type SSEHandler struct {
	sseHub *sse.Hub
}

func NewSSEHandler(sseHub *sse.Hub) *SSEHandler {
	return &SSEHandler{sseHub: sseHub}
}

func (h *SSEHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/sse", h.ServeSSE).Methods("GET")
}

func (h *SSEHandler) ServeSSE(w http.ResponseWriter, r *http.Request) {
	rawID := mux.Vars(r)["id"]
	projectID, err := domain.ParseProjectID(rawID)
	if err != nil {
		http.Error(w, `{"status":"fail","data":{"error":"invalid project ID"}}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
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
