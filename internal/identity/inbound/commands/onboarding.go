package commands

import (
	"errors"
	"net/http"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
)

type OnboardingHandler struct {
	onboarding   service.OnboardingCommands
	authCommands service.AuthCommands
	authQueries  service.AuthQueries
	controller   *controller.Controller
	limiter      *authIPLimiter
}

func NewOnboardingHandler(
	onboarding service.OnboardingCommands,
	authCommands service.AuthCommands,
	authQueries service.AuthQueries,
	ctrl *controller.Controller,
) *OnboardingHandler {
	return &OnboardingHandler{
		onboarding:   onboarding,
		authCommands: authCommands,
		authQueries:  authQueries,
		controller:   ctrl,
		limiter:      newAuthIPLimiter(0, 0),
	}
}

func (h *OnboardingHandler) RegisterRoutes(router *mux.Router) {
	// Authenticated: generate code
	router.HandleFunc("/api/onboarding/codes", h.GenerateCode).Methods("POST")
	// Unauthenticated: daemon completes onboarding (rate-limited to prevent brute-force of 6-digit codes)
	router.Handle("/api/onboarding/complete", h.limiter.middleware(http.HandlerFunc(h.CompleteOnboarding))).Methods("POST")
	// Unauthenticated: daemon refreshes access token (rate-limited)
	router.Handle("/api/daemon/refresh", h.limiter.middleware(http.HandlerFunc(h.RefreshDaemonToken))).Methods("POST")
}

type generateCodeRequest struct {
	Mode     string `json:"mode" validate:"required,oneof=default shared"`
	NodeName string `json:"node_name"`
}

type generateCodeResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GenerateCode handles POST /api/onboarding/codes
func (h *OnboardingHandler) GenerateCode(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	actor, ok := ActorFromRequest(w, r, h.controller, h.authQueries)
	if !ok {
		return
	}

	var req generateCodeRequest
	if err := h.controller.DecodeAndValidate(r, &req, nil); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	code, err := h.onboarding.GenerateCode(r.Context(), actor, domain.NodeMode(req.Mode), req.NodeName)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, generateCodeResponse{
		Code:      code.Code,
		ExpiresAt: code.ExpiresAt,
	})
}

type completeOnboardingRequest struct {
	Code     string `json:"code" validate:"required,len=6,numeric"`
	NodeName string `json:"node_name"`
}

type completeOnboardingResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	Node         nodeResponse `json:"node"`
}

type nodeResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// CompleteOnboarding handles POST /api/onboarding/complete
func (h *OnboardingHandler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	var req completeOnboardingRequest
	if err := h.controller.DecodeAndValidate(r, &req, nil); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	accessToken, refreshToken, node, err := h.onboarding.CompleteOnboarding(r.Context(), req.Code, req.NodeName)
	if err != nil {
		h.handleOnboardingError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, completeOnboardingResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Node: nodeResponse{
			ID:        node.ID.String(),
			Name:      node.Name,
			Mode:      string(node.Mode),
			Status:    string(node.Status),
			CreatedAt: node.CreatedAt,
		},
	})
}

func (h *OnboardingHandler) handleOnboardingError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrOnboardingCodeNotFound):
		status := http.StatusNotFound
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "CODE_NOT_FOUND", Message: "onboarding code not found"})
	case errors.Is(err, domain.ErrOnboardingCodeExpired):
		status := http.StatusGone
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "CODE_EXPIRED", Message: "onboarding code has expired"})
	case errors.Is(err, domain.ErrOnboardingCodeUsed):
		status := http.StatusConflict
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "CODE_ALREADY_USED", Message: "onboarding code has already been used"})
	default:
		h.controller.SendError(w, r, err)
	}
}

type refreshDaemonTokenRequest struct {
	NodeID       string `json:"node_id" validate:"required,uuid"`
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshDaemonToken handles POST /api/daemon/refresh
func (h *OnboardingHandler) RefreshDaemonToken(w http.ResponseWriter, r *http.Request) {
	var req refreshDaemonTokenRequest
	if err := h.controller.DecodeAndValidate(r, &req, nil); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	nodeID, err := domain.ParseNodeID(req.NodeID)
	if err != nil {
		status := http.StatusBadRequest
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_NODE_ID", Message: "invalid node ID format"})
		return
	}

	newAccessToken, err := h.authCommands.RefreshDaemonToken(r.Context(), nodeID, req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			status := http.StatusUnauthorized
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "invalid refresh token"})
		case errors.Is(err, domain.ErrNodeRevoked):
			status := http.StatusGone
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "NODE_REVOKED", Message: "node has been revoked"})
		default:
			h.controller.SendError(w, r, err)
		}
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{
		"access_token": newAccessToken,
	})
}

