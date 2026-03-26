package commands

import (
	"errors"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/gorilla/mux"
)

type SpecializedAgentCommandsHandler struct {
	commands   service.Commands
	queries    service.Queries
	controller *controller.Controller
	hub        *websocket.Hub
}

func NewSpecializedAgentCommandsHandler(commands service.Commands, queries service.Queries, ctrl *controller.Controller, hub *websocket.Hub) *SpecializedAgentCommandsHandler {
	return &SpecializedAgentCommandsHandler{
		commands:   commands,
		queries:    queries,
		controller: ctrl,
		hub:        hub,
	}
}

func (h *SpecializedAgentCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/agents/{slug}/specialized", h.HandleCreateSpecializedAgent).Methods("POST")
	router.HandleFunc("/api/agents/{slug}/specialized/{specSlug}", h.HandleUpdateSpecializedAgent).Methods("PATCH")
	router.HandleFunc("/api/agents/{slug}/specialized/{specSlug}", h.HandleDeleteSpecializedAgent).Methods("DELETE")
}

func (h *SpecializedAgentCommandsHandler) HandleCreateSpecializedAgent(w http.ResponseWriter, r *http.Request) {
	parentSlug := mux.Vars(r)["slug"]
	if parentSlug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrAgentSlugRequired)
		return
	}

	var req pkgserver.CreateSpecializedAgentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidSpecializedAgentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidSpecializedAgentRequest, err))
		return
	}

	agent, err := h.commands.CreateSpecializedAgent(r.Context(), parentSlug, req.Slug, req.Name, req.SkillSlugs, req.SortOrder)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "specialized_agent_created",
		Data: converters.ToPublicSpecializedAgent(agent, parentSlug, len(req.SkillSlugs)),
	})

	h.controller.SendSuccess(w, r, converters.ToPublicSpecializedAgent(agent, parentSlug, len(req.SkillSlugs)))
}

func (h *SpecializedAgentCommandsHandler) HandleUpdateSpecializedAgent(w http.ResponseWriter, r *http.Request) {
	specSlug := mux.Vars(r)["specSlug"]
	if specSlug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrSpecializedAgentNotFound)
		return
	}

	existing, err := h.queries.GetSpecializedAgent(r.Context(), specSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	var req pkgserver.UpdateSpecializedAgentRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidSpecializedAgentRequest); err != nil {
		h.controller.SendFail(w, r, nil, errors.Join(pkgserver.ErrInvalidSpecializedAgentRequest, err))
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	sortOrder := existing.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	var skillSlugs []string
	if req.SkillSlugs != nil {
		skillSlugs = *req.SkillSlugs
	}

	if err := h.commands.UpdateSpecializedAgent(r.Context(), existing.ID, name, skillSlugs, sortOrder); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "specialized_agent_updated",
		Data: map[string]string{"slug": specSlug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "specialized agent updated"})
}

func (h *SpecializedAgentCommandsHandler) HandleDeleteSpecializedAgent(w http.ResponseWriter, r *http.Request) {
	specSlug := mux.Vars(r)["specSlug"]
	if specSlug == "" {
		h.controller.SendFail(w, r, nil, domain.ErrSpecializedAgentNotFound)
		return
	}

	existing, err := h.queries.GetSpecializedAgent(r.Context(), specSlug)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	if err := h.commands.DeleteSpecializedAgent(r.Context(), existing.ID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "specialized_agent_deleted",
		Data: map[string]string{"slug": specSlug},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "specialized agent deleted"})
}
