package queries

import (
	"fmt"
	"net/http"

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
	projectID := mux.Vars(r)["id"]

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ch, unsubscribe := h.sseHub.Subscribe(projectID)
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
