package commands

import (
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
)

// FeatureCommandsHandler handles feature write operations
type FeatureCommandsHandler struct {
	commands   service.Commands
	controller *controller.Controller
	hub        *websocket.Hub
}

// NewFeatureCommandsHandler creates a new feature commands handler
func NewFeatureCommandsHandler(commands service.Commands, ctrl *controller.Controller, hub *websocket.Hub) *FeatureCommandsHandler {
	return &FeatureCommandsHandler{
		commands:   commands,
		controller: ctrl,
		hub:        hub,
	}
}

// RegisterRoutes registers feature command routes
func (h *FeatureCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/features", h.CreateFeature).Methods("POST")
	router.HandleFunc("/api/projects/{id}/features/{featureId}", h.UpdateFeature).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/features/{featureId}/status", h.UpdateFeatureStatus).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/features/{featureId}", h.DeleteFeature).Methods("DELETE")
}

// CreateFeature creates a new feature
func (h *FeatureCommandsHandler) CreateFeature(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	var req pkgserver.CreateFeatureRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidFeatureRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	feature, err := h.commands.CreateFeature(r.Context(), projectID, req.Name, req.Description, req.CreatedByRole, req.CreatedByAgent)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	resp := converters.ToPublicFeature(feature)

	h.hub.Broadcast(websocket.Event{
		Type: "feature_created",
		Data: resp,
	})

	h.controller.SendSuccess(w, r, resp)
}

// UpdateFeature updates a feature
func (h *FeatureCommandsHandler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	var req pkgserver.UpdateFeatureRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidFeatureRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}

	if err := h.commands.UpdateFeature(r.Context(), featureID, name, desc); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{Type: "feature_updated", Data: map[string]string{"feature_id": string(featureID)}})
	h.controller.SendSuccess(w, r, map[string]string{"message": "feature updated"})
}

// UpdateFeatureStatus updates a feature's status
func (h *FeatureCommandsHandler) UpdateFeatureStatus(w http.ResponseWriter, r *http.Request) {
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	var req pkgserver.UpdateFeatureStatusRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidFeatureRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	status := domain.FeatureStatus(req.Status)

	if err := h.commands.UpdateFeatureStatus(r.Context(), featureID, status); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{Type: "feature_status_updated", Data: map[string]string{"feature_id": string(featureID), "status": req.Status}})
	h.controller.SendSuccess(w, r, map[string]string{"message": "feature status updated"})
}

// DeleteFeature deletes a feature
func (h *FeatureCommandsHandler) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	if err := h.commands.DeleteFeature(r.Context(), featureID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "feature_deleted",
		Data: map[string]string{"feature_id": string(featureID)},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "feature deleted"})
}
