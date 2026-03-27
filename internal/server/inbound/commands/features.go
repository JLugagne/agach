package commands

import (
	"context"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
	"github.com/gorilla/mux"
)

// FeatureCommandsHandler handles feature write operations
type FeatureCommandsHandler struct {
	commands     service.Commands
	queries      service.Queries
	controller   *controller.Controller
	hub          *websocket.Hub
	teamResolver TeamIDResolver
}

// NewFeatureCommandsHandler creates a new feature commands handler.
// An optional queries parameter enables project-ownership verification on mutations.
func NewFeatureCommandsHandler(commands service.Commands, ctrl *controller.Controller, hub *websocket.Hub, queries ...service.Queries) *FeatureCommandsHandler {
	h := &FeatureCommandsHandler{
		commands:   commands,
		controller: ctrl,
		hub:        hub,
	}
	if len(queries) > 0 {
		h.queries = queries[0]
	}
	return h
}

// SetTeamResolver injects a TeamIDResolver for project access checks.
func (h *FeatureCommandsHandler) SetTeamResolver(tr TeamIDResolver) { h.teamResolver = tr }

// CheckAccess verifies the caller has access to the given project.
func (h *FeatureCommandsHandler) CheckAccess(r *http.Request, projectID domain.ProjectID) bool {
	return checkProjectAccess(r, projectID, h.queries, h.teamResolver)
}

func (h *FeatureCommandsHandler) verifyFeatureOwnership(w http.ResponseWriter, r *http.Request, projectID domain.ProjectID, featureID domain.FeatureID) bool {
	if h.queries == nil {
		return true
	}
	feature, err := h.queries.GetFeature(r.Context(), featureID)
	if err != nil {
		h.controller.SendFail(w, r, nil, domain.ErrFeatureNotFound)
		return false
	}
	if feature.ProjectID != projectID {
		h.controller.SendFail(w, r, nil, domain.ErrFeatureNotInProject)
		return false
	}
	return true
}

// RegisterRoutes registers feature command routes
func (h *FeatureCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{id}/features", h.CreateFeature).Methods("POST")
	router.HandleFunc("/api/projects/{id}/features/{featureId}", h.UpdateFeature).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/features/{featureId}/status", h.UpdateFeatureStatus).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/features/{featureId}/changelogs", h.UpdateFeatureChangelogs).Methods("PATCH")
	router.HandleFunc("/api/projects/{id}/features/{featureId}", h.DeleteFeature).Methods("DELETE")
}

// CreateFeature creates a new feature
func (h *FeatureCommandsHandler) CreateFeature(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])

	if !h.CheckAccess(r, projectID) {
		h.controller.SendFail(w, r, nil, domain.ErrProjectNotFound)
		return
	}

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
		Type:      "feature_created",
		ProjectID: projectID.String(),
		Data:      resp,
	})

	h.controller.SendSuccess(w, r, resp)
}

// UpdateFeature updates a feature
func (h *FeatureCommandsHandler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	if !h.verifyFeatureOwnership(w, r, projectID, featureID) {
		return
	}

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
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	if !h.verifyFeatureOwnership(w, r, projectID, featureID) {
		return
	}

	var req pkgserver.UpdateFeatureStatusRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidFeatureRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	status := domain.FeatureStatus(req.Status)

	if err := h.commands.UpdateFeatureStatus(r.Context(), featureID, status, req.NodeID); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{Type: "feature_status_updated", Data: map[string]string{"feature_id": string(featureID), "status": req.Status}})

	// Send notification for notable feature status changes
	h.notifyFeatureStatusChange(r.Context(), projectID, featureID, status)

	h.controller.SendSuccess(w, r, map[string]string{"message": "feature status updated"})
}

// UpdateFeatureChangelogs updates feature changelogs
func (h *FeatureCommandsHandler) UpdateFeatureChangelogs(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	if !h.verifyFeatureOwnership(w, r, projectID, featureID) {
		return
	}

	var req pkgserver.UpdateFeatureChangelogsRequest
	if err := h.controller.DecodeAndValidate(r, &req, pkgserver.ErrInvalidFeatureRequest); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	if err := h.commands.UpdateFeatureChangelogs(r.Context(), featureID, req.UserChangelog, req.TechChangelog); err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.hub.Broadcast(websocket.Event{
		Type: "feature_changelogs_updated",
		Data: map[string]string{"feature_id": string(featureID)},
	})

	h.controller.SendSuccess(w, r, map[string]string{"message": "feature changelogs updated"})
}

// DeleteFeature deletes a feature
func (h *FeatureCommandsHandler) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	projectID := domain.ProjectID(mux.Vars(r)["id"])
	featureID := domain.FeatureID(mux.Vars(r)["featureId"])

	if !h.verifyFeatureOwnership(w, r, projectID, featureID) {
		return
	}

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

// notifyFeatureStatusChange creates and broadcasts a notification for notable feature transitions.
func (h *FeatureCommandsHandler) notifyFeatureStatusChange(ctx context.Context, projectID domain.ProjectID, featureID domain.FeatureID, status domain.FeatureStatus) {
	var severity domain.NotificationSeverity
	var title string

	switch status {
	case domain.FeatureStatusDone:
		severity = domain.SeveritySuccess
		title = "Feature completed"
	case domain.FeatureStatusBlocked:
		severity = domain.SeverityWarning
		title = "Feature blocked"
	case domain.FeatureStatusInProgress:
		severity = domain.SeverityInfo
		title = "Feature started"
	default:
		return
	}

	featureName := string(featureID)
	if h.queries != nil {
		if f, err := h.queries.GetFeature(ctx, featureID); err == nil && f != nil {
			featureName = f.Name
		}
	}

	pid := projectID
	notification, err := h.commands.CreateNotification(ctx, &pid, domain.NotificationScopeProject, "", severity, title, featureName, "", "", "")
	if err != nil {
		return
	}
	h.hub.Broadcast(websocket.Event{
		Type:      "notification",
		ProjectID: string(projectID),
		Data:      converters.ToPublicNotification(notification),
	})
}
