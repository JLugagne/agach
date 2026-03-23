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

// SkillCommandsHandler handles skill write operations
type SkillCommandsHandler struct {
	commands   service.Commands
	queries    service.Queries
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewSkillCommandsHandler creates a new skill commands handler
func NewSkillCommandsHandler(commands service.Commands, queries service.Queries, ctrl *controller.Controller, hub *websocket.Hub) *SkillCommandsHandler {
	return &SkillCommandsHandler{
		commands:   commands,
		queries:    queries,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers skill command routes
func (h *SkillCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/skills", h.CreateSkill).Methods("POST")
	router.HandleFunc("/api/skills/{slug}", h.UpdateSkill).Methods("PATCH")
	router.HandleFunc("/api/skills/{slug}", h.DeleteSkill).Methods("DELETE")
	router.HandleFunc("/api/agents/{slug}/skills", h.AddSkillToAgent).Methods("POST")
	router.HandleFunc("/api/agents/{slug}/skills/{skillSlug}", h.RemoveSkillFromAgent).Methods("DELETE")
}

// CreateSkill creates a new skill
func (h *SkillCommandsHandler) CreateSkill(w http.ResponseWriter, r *http.Request) {
	var req pkgserver.CreateSkillRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidSkillRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidSkillRequest, err))
		return
	}

	skill, err := h.commands.CreateSkill(
		r.Context(),
		req.Slug,
		req.Name,
		req.Description,
		req.Content,
		req.Icon,
		req.Color,
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

	h.hub.Broadcast(websocket.Event{
		Type: "skill_created",
		Data: converters.ToPublicSkill(skill),
	})

	h.controller.SendSuccess(w, r, converters.ToPublicSkill(skill))
}

// UpdateSkill updates an existing skill
func (h *SkillCommandsHandler) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrSkillNotFound)
		return
	}

	skill, err := h.queries.GetSkillBySlug(r.Context(), slug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	var req pkgserver.UpdateSkillRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidSkillRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidSkillRequest, err))
		return
	}

	name := skill.Name
	if req.Name != nil {
		name = *req.Name
	}
	description := skill.Description
	if req.Description != nil {
		description = *req.Description
	}
	content := skill.Content
	if req.Content != nil {
		content = *req.Content
	}
	icon := skill.Icon
	if req.Icon != nil {
		icon = *req.Icon
	}
	color := skill.Color
	if req.Color != nil {
		color = *req.Color
	}
	sortOrder := skill.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	if err := h.commands.UpdateSkill(r.Context(), skill.ID, name, description, content, icon, color, sortOrder); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "skill_updated",
		Data: map[string]string{"slug": slug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "skill updated"})
}

// DeleteSkill deletes a skill
func (h *SkillCommandsHandler) DeleteSkill(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrSkillNotFound)
		return
	}

	skill, err := h.queries.GetSkillBySlug(r.Context(), slug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	if err := h.commands.DeleteSkill(r.Context(), skill.ID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "skill_deleted",
		Data: map[string]string{"slug": slug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "skill deleted"})
}

// AddSkillToAgent assigns a skill to an agent
func (h *SkillCommandsHandler) AddSkillToAgent(w http.ResponseWriter, r *http.Request) {
	agentSlug := mux.Vars(r)["slug"]
	if agentSlug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrAgentSlugRequired)
		return
	}

	var req pkgserver.AddSkillToAgentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidSkillRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidSkillRequest, err))
		return
	}

	if err := h.commands.AddSkillToAgent(r.Context(), agentSlug, req.SkillSlug); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "agent_skill_added",
		Data: map[string]string{"agent_slug": agentSlug, "skill_slug": req.SkillSlug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "skill added to agent"})
}

// RemoveSkillFromAgent removes a skill assignment from an agent
func (h *SkillCommandsHandler) RemoveSkillFromAgent(w http.ResponseWriter, r *http.Request) {
	agentSlug := mux.Vars(r)["slug"]
	skillSlug := mux.Vars(r)["skillSlug"]

	if agentSlug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrAgentSlugRequired)
		return
	}
	if skillSlug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrSkillNotFound)
		return
	}

	if err := h.commands.RemoveSkillFromAgent(r.Context(), agentSlug, skillSlug); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "agent_skill_removed",
		Data: map[string]string{"agent_slug": agentSlug, "skill_slug": skillSlug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "skill removed from agent"})
}
