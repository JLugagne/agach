package commands

import (
	"errors"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
)

// AgentCommandsHandler handles role write operations
type AgentCommandsHandler struct {
	commands   service.Commands
	queries    service.Queries
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewAgentCommandsHandler creates a new role commands handler
func NewAgentCommandsHandler(commands service.Commands, queries service.Queries, ctrl *controller.Controller, hub *websocket.Hub) *AgentCommandsHandler {
	return &AgentCommandsHandler{
		commands:   commands,
		queries:    queries,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers role command routes
func (h *AgentCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/agents", h.CreateAgent).Methods("POST")
	router.HandleFunc("/api/agents/{slug}", h.UpdateAgent).Methods("PATCH")
	router.HandleFunc("/api/agents/{slug}", h.DeleteAgent).Methods("DELETE")
	router.HandleFunc("/api/agents/{slug}/clone", h.CloneAgent).Methods("POST")
}

// CreateAgent creates a new role
func (h *AgentCommandsHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var req pkgserver.CreateAgentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidAgentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidAgentRequest, err))
		return
	}

	role, err := h.commands.CreateAgent(
		r.Context(),
		req.Slug,
		req.Name,
		req.Icon,
		req.Color,
		req.Description,
		req.PromptHint,
		req.PromptTemplate,
		req.TechStack,
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

	// Broadcast agent_created event
	h.hub.Broadcast(websocket.Event{
		Type: "agent_created",
		Data: converters.ToPublicAgent(role),
	})

	h.controller.SendSuccess(w, r, converters.ToPublicAgent(role))
}

// UpdateAgent updates an existing role
func (h *AgentCommandsHandler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrAgentSlugRequired)
		return
	}

	var req pkgserver.UpdateAgentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidAgentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidAgentRequest, err))
		return
	}

	// Get role ID from slug first (we need the ID)
	// This is a temporary workaround - ideally we'd add GetRoleBySlug to Commands or modify UpdateAgent signature
	// For now, we'll handle this through the service layer
	// TODO: Enhance this flow

	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	icon := ""
	if req.Icon != nil {
		icon = *req.Icon
	}
	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	promptHint := ""
	if req.PromptHint != nil {
		promptHint = *req.PromptHint
	}
	promptTemplate := ""
	if req.PromptTemplate != nil {
		promptTemplate = *req.PromptTemplate
	}
	techStack := []string{}
	if req.TechStack != nil {
		techStack = *req.TechStack
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	// Resolve slug to actual role ID
	agent, err := h.queries.GetAgentBySlug(r.Context(), slug)
	if err != nil || agent == nil {
		h.controller.SendFail(w, r, nil, domain.ErrRoleNotFound)
		return
	}
	roleID := agent.ID

	err = h.commands.UpdateAgent(
		r.Context(),
		roleID,
		name,
		icon,
		color,
		description,
		promptHint,
		promptTemplate,
		techStack,
		sortOrder,
	)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast agent_updated event
	h.hub.Broadcast(websocket.Event{
		Type: "agent_updated",
		Data: map[string]string{"slug": slug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "role updated"})
}

// DeleteAgent deletes a role
func (h *AgentCommandsHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrAgentSlugRequired)
		return
	}

	// Resolve slug to actual role ID
	agent, err := h.queries.GetAgentBySlug(r.Context(), slug)
	if err != nil || agent == nil {
		h.controller.SendFail(w, r, nil, domain.ErrRoleNotFound)
		return
	}
	roleID := agent.ID

	err = h.commands.DeleteAgent(r.Context(), roleID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Broadcast agent_deleted event
	h.hub.Broadcast(websocket.Event{
		Type: "agent_deleted",
		Data: map[string]string{"slug": slug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "role deleted"})
}

// CloneAgent clones an existing role into a new role with a different slug
func (h *AgentCommandsHandler) CloneAgent(w http.ResponseWriter, r *http.Request) {
	sourceSlug := mux.Vars(r)["slug"]
	if sourceSlug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrAgentSlugRequired)
		return
	}

	var req pkgserver.CloneAgentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidAgentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidAgentRequest, err))
		return
	}

	cloned, err := h.commands.CloneAgent(r.Context(), sourceSlug, req.NewSlug, req.NewName)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "agent_cloned",
		Data: map[string]string{
			"source_slug": sourceSlug,
			"new_slug":    req.NewSlug,
		},
	})

	h.controller.SendSuccess(w, r, converters.ToPublicAgent(cloned))
}
