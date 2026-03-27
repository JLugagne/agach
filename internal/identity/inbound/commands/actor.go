package commands

import (
	"net/http"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
)

// ActorFromRequest extracts and validates the Bearer token from the Authorization header.
// Returns (actor, true) on success; writes an error response and returns (zero, false) on failure.
func ActorFromRequest(w http.ResponseWriter, r *http.Request, ctrl *controller.Controller, authQueries service.AuthQueries) (domain.Actor, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	status := http.StatusUnauthorized

	if authHeader == "" {
		ctrl.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "authentication required"})
		return domain.Actor{}, false
	}

	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
		ctrl.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "authorization header must use Bearer scheme"})
		return domain.Actor{}, false
	}

	token := authHeader[len(prefix):]
	actor, err := authQueries.ValidateJWT(r.Context(), token)
	if err != nil {
		ctrl.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "invalid or expired token"})
		return domain.Actor{}, false
	}

	return actor, true
}
