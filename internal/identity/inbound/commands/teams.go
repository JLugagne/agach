package commands

import (
	"errors"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
)

// TeamsHandler handles team and membership HTTP endpoints.
type TeamsHandler struct {
	commands   service.TeamCommands
	queries    service.TeamQueries
	authQueries service.AuthQueries
	ctrl       *controller.Controller
}

// NewTeamsHandler creates a teams handler.
func NewTeamsHandler(cmds service.TeamCommands, qrs service.TeamQueries, authQrs service.AuthQueries, ctrl *controller.Controller) *TeamsHandler {
	return &TeamsHandler{commands: cmds, queries: qrs, authQueries: authQrs, ctrl: ctrl}
}

// RegisterRoutes registers team routes on the router.
func (h *TeamsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/identity/teams", h.ListTeams).Methods("GET")
	router.HandleFunc("/api/identity/teams", h.CreateTeam).Methods("POST")
	router.HandleFunc("/api/identity/teams/{id}", h.DeleteTeam).Methods("DELETE")
	router.HandleFunc("/api/identity/users", h.ListUsers).Methods("GET")
	router.HandleFunc("/api/identity/users/{id}/team", h.SetUserTeam).Methods("PUT")
	router.HandleFunc("/api/identity/users/{id}/team", h.RemoveUserFromTeam).Methods("DELETE")
	router.HandleFunc("/api/identity/users/{id}/role", h.SetUserRole).Methods("PUT")
}

type createTeamRequest struct {
	Name        string `json:"name" validate:"required"`
	Slug        string `json:"slug" validate:"required,slug"`
	Description string `json:"description"`
}

type setUserTeamRequest struct {
	TeamID string `json:"team_id" validate:"required"`
}

type setUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

// ListTeams handles GET /api/identity/teams.
func (h *TeamsHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actorFromRequest(w, r); !ok {
		return
	}
	teams, err := h.queries.ListTeams(r.Context())
	if err != nil {
		h.ctrl.SendError(w, r, err)
		return
	}
	out := make([]map[string]interface{}, 0, len(teams))
	for _, t := range teams {
		out = append(out, teamToMap(t))
	}
	h.ctrl.SendSuccess(w, r, out)
}

// CreateTeam handles POST /api/identity/teams.
func (h *TeamsHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	var req createTeamRequest
	if err := h.ctrl.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.ctrl.SendFail(w, r, nil, err)
		return
	}

	team, err := h.commands.CreateTeam(r.Context(), actor, req.Name, req.Slug, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			status := http.StatusForbidden
			h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "FORBIDDEN", Message: "access denied"})
			return
		}
		if errors.Is(err, domain.ErrTeamSlugConflict) {
			status := http.StatusConflict
			h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "TEAM_SLUG_CONFLICT", Message: err.Error()})
			return
		}
		h.ctrl.SendError(w, r, err)
		return
	}

	h.ctrl.SendSuccess(w, r, teamToMap(team))
}

// DeleteTeam handles DELETE /api/identity/teams/{id}.
func (h *TeamsHandler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	teamID, err := domain.ParseTeamID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_TEAM_ID", Message: "invalid team id"})
		return
	}

	if err := h.commands.DeleteTeam(r.Context(), actor, teamID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			status := http.StatusForbidden
			h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "FORBIDDEN", Message: "access denied"})
			return
		}
		if errors.Is(err, domain.ErrTeamNotFound) {
			status := http.StatusNotFound
			h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "TEAM_NOT_FOUND", Message: "team not found"})
			return
		}
		h.ctrl.SendError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListUsers handles GET /api/identity/users.
func (h *TeamsHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		h.ctrl.SendError(w, r, err)
		return
	}
	out := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		out = append(out, userToPublicMap(u, actor.Role))
	}
	h.ctrl.SendSuccess(w, r, out)
}

// SetUserTeam handles PUT /api/identity/users/{id}/team.
func (h *TeamsHandler) SetUserTeam(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	userID, err := domain.ParseUserID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_USER_ID", Message: "invalid user id"})
		return
	}

	var req setUserTeamRequest
	if err := h.ctrl.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.ctrl.SendFail(w, r, nil, err)
		return
	}

	teamID, err := domain.ParseTeamID(req.TeamID)
	if err != nil {
		status := http.StatusBadRequest
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_TEAM_ID", Message: "invalid team id"})
		return
	}

	if err := h.commands.AddUserToTeam(r.Context(), actor, userID, teamID); err != nil {
		h.handleTeamError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveUserFromTeam handles DELETE /api/identity/users/{id}/team.
func (h *TeamsHandler) RemoveUserFromTeam(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	userID, err := domain.ParseUserID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_USER_ID", Message: "invalid user id"})
		return
	}

	if err := h.commands.RemoveUserFromTeam(r.Context(), actor, userID); err != nil {
		h.handleTeamError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetUserRole handles PUT /api/identity/users/{id}/role.
func (h *TeamsHandler) SetUserRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	userID, err := domain.ParseUserID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_USER_ID", Message: "invalid user id"})
		return
	}

	var req setUserRoleRequest
	if err := h.ctrl.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.ctrl.SendFail(w, r, nil, err)
		return
	}

	if err := h.commands.SetUserRole(r.Context(), actor, userID, domain.MemberRole(req.Role)); err != nil {
		h.handleTeamError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TeamsHandler) actorFromRequest(w http.ResponseWriter, r *http.Request) (domain.Actor, bool) {
	authHeader := r.Header.Get("Authorization")
	status := http.StatusUnauthorized

	if authHeader != "" {
		const prefix = "Bearer "
		if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
			h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "authorization header must use Bearer scheme"})
			return domain.Actor{}, false
		}
		token := authHeader[len(prefix):]
		actor, err := h.authQueries.ValidateJWT(r.Context(), token)
		if err != nil {
			h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "invalid or expired token"})
			return domain.Actor{}, false
		}
		return actor, true
	}

	h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "authentication required"})
	return domain.Actor{}, false
}

func (h *TeamsHandler) handleTeamError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		status := http.StatusForbidden
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "FORBIDDEN", Message: "access denied"})
	case errors.Is(err, domain.ErrUserNotFound):
		status := http.StatusNotFound
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "USER_NOT_FOUND", Message: "user not found"})
	case errors.Is(err, domain.ErrTeamNotFound):
		status := http.StatusNotFound
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "TEAM_NOT_FOUND", Message: "team not found"})
	default:
		h.ctrl.SendError(w, r, err)
	}
}

func teamToMap(t domain.Team) map[string]interface{} {
	return map[string]interface{}{
		"id":          t.ID.String(),
		"name":        t.Name,
		"slug":        t.Slug,
		"description": t.Description,
		"created_at":  t.CreatedAt,
		"updated_at":  t.UpdatedAt,
	}
}

func userToPublicMap(u domain.User, callerRole domain.MemberRole) map[string]interface{} {
	var teamID *string
	if u.TeamID != nil {
		s := u.TeamID.String()
		teamID = &s
	}
	m := map[string]interface{}{
		"id":           u.ID.String(),
		"display_name": u.DisplayName,
		"role":         string(u.Role),
		"team_id":      teamID,
		"created_at":   u.CreatedAt,
		"updated_at":   u.UpdatedAt,
	}
	if callerRole == domain.RoleAdmin {
		m["email"] = u.Email
	}
	return m
}
