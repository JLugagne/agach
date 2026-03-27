package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/gorilla/mux"
)

// ChatsHandler handles chat session write operations
type ChatsHandler struct {
	chatService service.ChatService
	queries     service.Queries
	controller  *controller.Controller
	hub         *websocket.Hub
	dataDir     string
}

// NewChatsHandler creates a new chat commands handler
func NewChatsHandler(chatService service.ChatService, queries service.Queries, ctrl *controller.Controller, hub *websocket.Hub, dataDir string) *ChatsHandler {
	return &ChatsHandler{
		chatService: chatService,
		queries:     queries,
		controller:  ctrl,
		hub:         hub,
		dataDir:     dataDir,
	}
}

// RegisterRoutes registers chat command routes
func (h *ChatsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{projectId}/features/{featureId}/chats", h.StartSession).Methods("POST")
	router.HandleFunc("/api/projects/{projectId}/features/{featureId}/chats/{sessionId}/end", h.EndSession).Methods("POST")
	router.HandleFunc("/api/projects/{projectId}/features/{featureId}/chats/{sessionId}/stats", h.UpdateStats).Methods("PUT")
	router.HandleFunc("/api/projects/{projectId}/features/{featureId}/chats/{sessionId}/upload", h.UploadJSONL).Methods("POST")
}

// StartSession starts a new chat session for a feature
func (h *ChatsHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["projectId"])
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	var req pkgserver.StartChatSessionRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidChatSessionRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	// Handle optional resume_session_id if needed
	_ = req.ResumeSessionID

	session, err := h.chatService.CreateSession(r.Context(), featureID, projectID, req.NodeID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicChatSession(*session)

	h.hub.Broadcast(websocket.Event{
		Type: "chat_session_started",
		Data: resp,
	})

	// Build initial message from the project's default agent template + feature description
	initialMessage := h.buildInitialMessage(r.Context(), projectID, featureID)

	// Send chat.start to the targeted daemon so it spawns a Claude session
	log.Printf("[chats] StartSession: node_id=%q session_id=%s feature_id=%s project_id=%s", req.NodeID, session.ID, featureID, projectID)
	if req.NodeID != "" {
		startPayload, _ := json.Marshal(daemonws.ChatStartRequest{
			SessionID:       session.ID.String(),
			FeatureID:       featureID.String(),
			ProjectID:       projectID.String(),
			NodeID:          req.NodeID,
			ResumeSessionID: derefString(req.ResumeSessionID),
			InitialMessage:  initialMessage,
		})
		chatStartMsg, _ := json.Marshal(struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}{
			Type: daemonws.TypeChatStart,
			Data: startPayload,
		})
		sent := h.hub.SendToDaemon(req.NodeID, chatStartMsg)
		log.Printf("[chats] SendToDaemon result: sent=%v node_id=%q msg_len=%d", sent, req.NodeID, len(chatStartMsg))
	} else {
		log.Printf("[chats] StartSession: no node_id provided, skipping daemon dispatch")
	}

	h.controller.SendSuccess(w, r, resp)
}

// EndSession ends a chat session
func (h *ChatsHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := domain.ParseChatSessionID(mux.Vars(r)["sessionId"])
	if err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	if err := h.chatService.EndSession(r.Context(), sessionID, domain.ChatStateEnded); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "chat_session_ended",
		Data: map[string]string{"session_id": sessionID.String()},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "chat session ended"})
}

// UpdateStats updates token usage and model for a chat session
func (h *ChatsHandler) UpdateStats(w http.ResponseWriter, r *http.Request) {
	sessionID, err := domain.ParseChatSessionID(mux.Vars(r)["sessionId"])
	if err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	var req pkgserver.UpdateChatStatsRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidChatStatsRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	usage := domain.TokenUsage{
		InputTokens:      req.InputTokens,
		OutputTokens:     req.OutputTokens,
		CacheReadTokens:  req.CacheReadTokens,
		CacheWriteTokens: req.CacheWriteTokens,
		Model:            req.Model,
	}
	if err := h.chatService.UpdateTokenUsage(r.Context(), sessionID, usage); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{"message": "stats updated"})
}

// UploadJSONL uploads a JSONL file for a chat session
func (h *ChatsHandler) UploadJSONL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	featureID := domain.FeatureID(vars["featureId"])
	sessionID, err := domain.ParseChatSessionID(vars["sessionId"])
	if err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	// Limit upload to 100MB
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

	// Parse multipart form with 32MB buffer
	if err := r.ParseMultipartForm(32 * 1024 * 1024); err != nil {
		h.controller.SendFail(w, r, nil, &domain.Error{
			Code:    "INVALID_FORM",
			Message: "failed to parse multipart form",
		})
		return
	}

	// Get the "jsonl" form file
	file, header, err := r.FormFile("jsonl")
	if err != nil {
		h.controller.SendFail(w, r, nil, &domain.Error{
			Code:    "MISSING_JSONL_FILE",
			Message: "jsonl file not provided",
		})
		return
	}
	defer file.Close()

	// Validate .jsonl extension
	filename := header.Filename
	if !strings.HasSuffix(strings.ToLower(filename), ".jsonl") {
		h.controller.SendFail(w, r, nil, &domain.Error{
			Code:    "INVALID_FILE_EXTENSION",
			Message: "file must have .jsonl extension",
		})
		return
	}

	// Create directory data/chats/<feature_id>/
	chatDir := filepath.Join(h.dataDir, "chats", featureID.String())
	if err := os.MkdirAll(chatDir, 0755); err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to create chat directory: %w", err))
		return
	}

	// Save as <session_id>.jsonl with sanitization
	jsonlPath := filepath.Join(chatDir, sessionID.String()+".jsonl")

	// Ensure the path doesn't escape the chat directory
	absPath, err := filepath.Abs(jsonlPath)
	if err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to resolve path: %w", err))
		return
	}
	absChatDir, err := filepath.Abs(chatDir)
	if err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to resolve chat directory: %w", err))
		return
	}
	if !strings.HasPrefix(absPath, absChatDir) {
		h.controller.SendFail(w, r, nil, &domain.Error{
			Code:    "INVALID_PATH",
			Message: "invalid file path",
		})
		return
	}

	// Write file
	outFile, err := os.Create(jsonlPath)
	if err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to create file: %w", err))
		return
	}
	defer outFile.Close()

	// Copy file contents
	if _, err := outFile.ReadFrom(file); err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to write file: %w", err))
		return
	}

	// Calculate relative path for storage
	relativePath := filepath.Join("chats", featureID.String(), sessionID.String()+".jsonl")

	// Update session JSONL path via ChatService
	if err := h.chatService.UpdateJSONLPath(r.Context(), sessionID, relativePath); err != nil {
		h.controller.SendError(w, r, fmt.Errorf("failed to update session JSONL path: %w", err))
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{
		"message": "JSONL file uploaded successfully",
		"path":    relativePath,
	})
}

// buildInitialMessage renders the default agent's prompt_template with the
// feature description. If the template is empty, a default message is used.
// Returns "" for resumed sessions or if the feature cannot be fetched.
func (h *ChatsHandler) buildInitialMessage(ctx context.Context, projectID domain.ProjectID, featureID domain.FeatureID) string {
	if h.queries == nil {
		return ""
	}

	feature, err := h.queries.GetFeature(ctx, featureID)
	if err != nil || feature == nil {
		return ""
	}

	project, err := h.queries.GetProject(ctx, projectID)
	if err != nil || project == nil || project.DefaultRole == "" {
		return ""
	}

	agent, err := h.queries.GetProjectAgentBySlug(ctx, projectID, project.DefaultRole)
	if err != nil || agent == nil {
		// Fall back to global agent
		agent, err = h.queries.GetAgentBySlug(ctx, project.DefaultRole)
		if err != nil || agent == nil {
			return renderDefaultFeatureMessage(feature.Description)
		}
	}

	if agent.PromptTemplate == "" {
		return renderDefaultFeatureMessage(feature.Description)
	}

	return renderFeatureTemplate(agent.PromptTemplate, feature)
}

// renderDefaultFeatureMessage returns the fallback initial message.
func renderDefaultFeatureMessage(description string) string {
	return "Here is a feature I want to implement, help me refining it:\n\n" + description
}

// renderFeatureTemplate renders an agent's prompt_template with feature data.
// Template slots use {{key.field}} notation (same as prompt.go).
func renderFeatureTemplate(tmpl string, feature *domain.Feature) string {
	// Simple string replacement for feature-scoped templates.
	r := strings.NewReplacer(
		"{{task.description}}", feature.Description,
		"{{task.title}}", feature.Name,
		"{{feature.description}}", feature.Description,
		"{{feature.name}}", feature.Name,
	)
	return r.Replace(tmpl)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
