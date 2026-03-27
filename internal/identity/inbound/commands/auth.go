package commands

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

const (
	refreshCookieName   = "refresh_token"
	refreshCookiePath   = "/api/auth/refresh"
	refreshCookieTTL    = domain.DefaultRefreshTokenTTL
	rememberMeCookieTTL = domain.DefaultRememberMeTokenTTL
)

// AuthCommandsHandler handles authentication HTTP endpoints.
type AuthCommandsHandler struct {
	commands    service.AuthCommands
	queries     service.AuthQueries
	controller  *controller.Controller
	authLimiter *authIPLimiter
}

// NewAuthCommandsHandler creates a new auth commands handler.
// ratePerSecond and burst configure the auth rate limiter; zero values use defaults.
func NewAuthCommandsHandler(cmds service.AuthCommands, qrs service.AuthQueries, ctrl *controller.Controller, ratePerSecond float64, burst int) *AuthCommandsHandler {
	return &AuthCommandsHandler{
		commands:    cmds,
		queries:     qrs,
		controller:  ctrl,
		authLimiter: newAuthIPLimiter(ratePerSecond, burst),
	}
}

const maxBodyBytes = 64 * 1024 // 64 KB

// bodySizeLimit wraps a handler to reject requests with bodies larger than maxBodyBytes.
func bodySizeLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// RegisterRoutes registers auth routes on the router.
func (h *AuthCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.Handle("/api/auth/register", bodySizeLimit(h.authLimiter.middleware(http.HandlerFunc(h.Register)))).Methods("POST")
	router.Handle("/api/auth/login", bodySizeLimit(h.authLimiter.middleware(http.HandlerFunc(h.Login)))).Methods("POST")
	router.HandleFunc("/api/auth/refresh", h.Refresh).Methods("POST")
	router.HandleFunc("/api/auth/logout", h.Logout).Methods("POST")
	router.HandleFunc("/api/auth/me", h.GetMe).Methods("GET")
	router.HandleFunc("/api/auth/me", h.UpdateProfile).Methods("PATCH")
	router.HandleFunc("/api/auth/me/password", h.ChangePassword).Methods("POST")
	router.Handle("/api/auth/complete-invite", bodySizeLimit(h.authLimiter.middleware(http.HandlerFunc(h.CompleteInvite)))).Methods("POST")
}

type registerRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required"`
	RememberMe bool   `json:"remember_me"`
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name" validate:"required"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

type completeInviteRequest struct {
	Token       string `json:"token" validate:"required"`
	DisplayName string `json:"display_name" validate:"required"`
	Password    string `json:"password" validate:"required,min=8"`
}

// Register handles POST /api/auth/register.
func (h *AuthCommandsHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := h.controller.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	user, err := h.commands.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	accessToken, refreshToken, err := h.commands.Login(r.Context(), req.Email, req.Password, false)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	h.setRefreshCookie(w, r, refreshToken, false)
	h.controller.SendSuccess(w, r, map[string]interface{}{
		"user":         toPublicUser(user),
		"access_token": accessToken,
	})
}

// Login handles POST /api/auth/login.
func (h *AuthCommandsHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := h.controller.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			status := http.StatusRequestEntityTooLarge
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "PAYLOAD_TOO_LARGE", Message: "request body too large"})
			return
		}
		h.controller.SendFail(w, r, nil, err)
		return
	}

	accessToken, refreshToken, err := h.commands.Login(r.Context(), req.Email, req.Password, req.RememberMe)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	actor, err := h.queries.ValidateJWT(r.Context(), accessToken)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	user, err := h.queries.GetCurrentUser(r.Context(), actor)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	h.setRefreshCookie(w, r, refreshToken, req.RememberMe)
	h.controller.SendSuccess(w, r, map[string]interface{}{
		"user":         toPublicUser(user),
		"access_token": accessToken,
	})
}

// Refresh handles POST /api/auth/refresh.
func (h *AuthCommandsHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		status := http.StatusUnauthorized
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "refresh token missing"})
		return
	}

	newAccessToken, err := h.commands.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, map[string]string{
		"access_token": newAccessToken,
	})
}

// Logout handles POST /api/auth/logout.
func (h *AuthCommandsHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
	h.controller.SendSuccess(w, r, nil)
}

// GetMe handles GET /api/auth/me.
func (h *AuthCommandsHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.queries)
	if !ok {
		return
	}

	user, err := h.queries.GetCurrentUser(r.Context(), actor)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, toPublicUser(user))
}

// UpdateProfile handles PATCH /api/auth/me.
func (h *AuthCommandsHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.queries)
	if !ok {
		return
	}

	var req updateProfileRequest
	if err := h.controller.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	user, err := h.commands.UpdateProfile(r.Context(), actor, req.DisplayName)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, toPublicUser(user))
}

// ChangePassword handles POST /api/auth/me/password.
func (h *AuthCommandsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.queries)
	if !ok {
		return
	}

	var req changePasswordRequest
	if err := h.controller.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	rawToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if err := h.commands.ChangePassword(r.Context(), actor, req.CurrentPassword, req.NewPassword, rawToken); err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			status := http.StatusUnauthorized
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_CREDENTIALS", Message: err.Error()})
			return
		}
		h.controller.SendError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CompleteInvite handles POST /api/auth/complete-invite.
func (h *AuthCommandsHandler) CompleteInvite(w http.ResponseWriter, r *http.Request) {
	var req completeInviteRequest
	if err := h.controller.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	user, err := h.commands.CompleteInvite(r.Context(), req.Token, req.DisplayName, req.Password)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	// Auto-login after completing invite
	accessToken, refreshToken, err := h.commands.Login(r.Context(), user.Email, req.Password, false)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	h.setRefreshCookie(w, r, refreshToken, false)
	h.controller.SendSuccess(w, r, map[string]interface{}{
		"user":         toPublicUser(user),
		"access_token": accessToken,
	})
}

func (h *AuthCommandsHandler) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusUnauthorized
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_CREDENTIALS", Message: err.Error()})
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		s := http.StatusConflict
		h.controller.SendFail(w, r, &s, &apierror.Error{Code: "EMAIL_ALREADY_EXISTS", Message: err.Error()})
	case errors.Is(err, domain.ErrSSOUserNoPassword):
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "SSO_USER_NO_PASSWORD", Message: err.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: err.Error()})
	case domain.IsDomainError(err):
		s := http.StatusBadRequest
		var de *domain.Error
		if errors.As(err, &de) {
			h.controller.SendFail(w, r, &s, &apierror.Error{Code: de.Code, Message: de.Message})
		} else {
			h.controller.SendError(w, r, err)
		}
	default:
		h.controller.SendError(w, r, err)
	}
}

func (h *AuthCommandsHandler) setRefreshCookie(w http.ResponseWriter, r *http.Request, token string, rememberMe bool) {
	ttl := refreshCookieTTL
	if rememberMe {
		ttl = rememberMeCookieTTL
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func isSecure(r *http.Request) bool {
	return r.TLS != nil
}

type publicUser struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

func toPublicUser(u domain.User) publicUser {
	return publicUser{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        string(u.Role),
		CreatedAt:   u.CreatedAt,
	}
}

// authIPLimiter is a per-IP rate limiter for auth endpoints.
// 5 requests per 15 minutes per IP.
type authIPLimiter struct {
	mu            sync.Mutex
	limiters      map[string]*authLimiterEntry
	ratePerSecond float64
	burst         int
}

type authLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newAuthIPLimiter(ratePerSecond float64, burst int) *authIPLimiter {
	// Defaults: 5 requests per second, burst of 10
	if ratePerSecond <= 0 {
		ratePerSecond = 5
	}
	if burst <= 0 {
		burst = 10
	}
	return &authIPLimiter{
		limiters:      make(map[string]*authLimiterEntry),
		ratePerSecond: ratePerSecond,
		burst:         burst,
	}
}

func (l *authIPLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.limiters[ip]
	if !ok {
		lim := rate.NewLimiter(rate.Limit(l.ratePerSecond), l.burst)
		l.limiters[ip] = &authLimiterEntry{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (l *authIPLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIPFromRequest(r)
		if !l.getLimiter(ip).Allow() {
			http.Error(w, `{"status":"fail","error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
