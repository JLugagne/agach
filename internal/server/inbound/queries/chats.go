package queries

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/gorilla/mux"
)

// ChatQueriesHandler handles chat session read operations
type ChatQueriesHandler struct {
	chatService service.ChatService
	controller  *controller.Controller
	dataDir     string
}

// NewChatQueriesHandler creates a new chat queries handler
func NewChatQueriesHandler(chatService service.ChatService, ctrl *controller.Controller, dataDir string) *ChatQueriesHandler {
	return &ChatQueriesHandler{
		chatService: chatService,
		controller:  ctrl,
		dataDir:     dataDir,
	}
}

// RegisterRoutes registers chat query routes
func (h *ChatQueriesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{projectId}/features/{featureId}/chats", h.ListSessions).Methods("GET")
	router.HandleFunc("/api/projects/{projectId}/features/{featureId}/chats/{sessionId}", h.GetSession).Methods("GET")
	router.HandleFunc("/api/projects/{projectId}/features/{featureId}/chats/{sessionId}/download", h.DownloadJSONL).Methods("GET")
}

// ListSessions lists all chat sessions for a feature
func (h *ChatQueriesHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	sessions, err := h.chatService.ListSessions(r.Context(), featureID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicChatSessions(sessions)
	h.controller.SendSuccess(w, r, resp)
}

// GetSession retrieves a specific chat session
func (h *ChatQueriesHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := domain.ParseChatSessionID(mux.Vars(r)["sessionId"])
	if err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	session, err := h.chatService.GetSession(r.Context(), sessionID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicChatSession(*session)
	h.controller.SendSuccess(w, r, resp)
}

// DownloadJSONL serves the JSONL file for a chat session
func (h *ChatQueriesHandler) DownloadJSONL(w http.ResponseWriter, r *http.Request) {
	sessionID, err := domain.ParseChatSessionID(mux.Vars(r)["sessionId"])
	if err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	session, err := h.chatService.GetSession(r.Context(), sessionID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	if session.JSONLPath == "" {
		h.controller.SendFail(w, r, nil, &domain.Error{
			Code:    "JSONL_NOT_AVAILABLE",
			Message: "no JSONL file available for this session",
		})
		return
	}

	// Resolve the JSONL path relative to the data directory
	jsonlPath := session.JSONLPath
	if !filepath.IsAbs(jsonlPath) {
		jsonlPath = filepath.Join(h.dataDir, jsonlPath)
	}

	// Check if the file exists
	if _, err := os.Stat(jsonlPath); err != nil {
		if os.IsNotExist(err) {
			h.controller.SendFail(w, r, nil, &domain.Error{
				Code:    "FILE_NOT_FOUND",
				Message: "JSONL file not found",
			})
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Set appropriate headers for download
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", "attachment; filename=\"session-"+sessionID.String()+".jsonl\"")

	// Serve the file
	http.ServeFile(w, r, jsonlPath)
}
