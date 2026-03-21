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
	"github.com/JLugagne/agach-mcp/pkg/apierror"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/api/auth/refresh"
	refreshCookieTTL  = 7 * 24 * time.Hour
)

// AuthCommandsHandler handles authentication HTTP endpoints.
type AuthCommandsHandler struct {
	commands    service.AuthCommands
	queries     service.AuthQueries
	controller  *controller.Controller
	authLimiter *authIPLimiter
}

// NewAuthCommandsHandler creates a new auth commands handler.
func NewAuthCommandsHandler(cmds service.AuthCommands, qrs service.AuthQueries, ctrl *controller.Controller) *AuthCommandsHandler {
	return &AuthCommandsHandler{
		commands:    cmds,
		queries:     qrs,
		controller:  ctrl,
		authLimiter: newAuthIPLimiter(),
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
	router.HandleFunc("/api/auth/apikeys", h.ListAPIKeys).Methods("GET")
	router.Handle("/api/auth/apikeys", h.authLimiter.middleware(bodySizeLimit(http.HandlerFunc(h.CreateAPIKey)))).Methods("POST")
	router.HandleFunc("/api/auth/apikeys/{id}", h.RevokeAPIKey).Methods("DELETE")
}

type registerRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type createAPIKeyRequest struct {
	Name      string     `json:"name" validate:"required"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
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

	accessToken, refreshToken, err := h.commands.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	h.setRefreshCookie(w, r, refreshToken)
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

	accessToken, refreshToken, err := h.commands.Login(r.Context(), req.Email, req.Password)
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

	h.setRefreshCookie(w, r, refreshToken)
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

// ListAPIKeys handles GET /api/auth/apikeys.
func (h *AuthCommandsHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	keys, err := h.queries.ListAPIKeys(r.Context(), actor)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, toPublicAPIKeys(keys))
}

// CreateAPIKey handles POST /api/auth/apikeys.
func (h *AuthCommandsHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	var req createAPIKeyRequest
	if err := h.controller.DecodeAndValidate(r, &req, &apierror.Error{Code: "INVALID_REQUEST", Message: "invalid request body"}); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	key, rawKey, err := h.commands.CreateAPIKey(r.Context(), actor, req.Name, req.Scopes, req.ExpiresAt)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	pub := toPublicAPIKey(key)
	h.controller.SendSuccess(w, r, map[string]interface{}{
		"api_key":    rawKey,
		"id":         pub.ID,
		"name":       pub.Name,
		"scopes":     pub.Scopes,
		"expires_at": pub.ExpiresAt,
		"created_at": pub.CreatedAt,
	})
}

// RevokeAPIKey handles DELETE /api/auth/apikeys/{id}.
func (h *AuthCommandsHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actorFromRequest(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	keyID, err := domain.ParseAPIKeyID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_KEY_ID", Message: "invalid api key id"})
		return
	}

	if err := h.commands.RevokeAPIKey(r.Context(), actor, keyID); err != nil {
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			status := http.StatusNotFound
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "API_KEY_NOT_FOUND", Message: "api key not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			status := http.StatusForbidden
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "FORBIDDEN", Message: "access denied"})
			return
		}
		h.controller.SendError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ActorFromRequest extracts the actor from the Authorization or X-Api-Key header.
// Returns (actor, true) on success; writes an error response and returns (zero, false) on failure.
func (h *AuthCommandsHandler) ActorFromRequest(w http.ResponseWriter, r *http.Request) (domain.Actor, bool) {
	return h.actorFromRequest(w, r)
}

func (h *AuthCommandsHandler) actorFromRequest(w http.ResponseWriter, r *http.Request) (domain.Actor, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	apiKeyHeader := strings.TrimSpace(r.Header.Get("X-Api-Key"))

	status := http.StatusUnauthorized

	if authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		actor, err := h.queries.ValidateJWT(r.Context(), token)
		if err != nil {
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "invalid or expired token"})
			return domain.Actor{}, false
		}
		return actor, true
	}

	if apiKeyHeader != "" {
		actor, err := h.queries.ValidateAPIKey(r.Context(), apiKeyHeader)
		if err != nil {
			h.controller.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "invalid api key"})
			return domain.Actor{}, false
		}
		return actor, true
	}

	h.controller.SendFail(w, r, &status, &apierror.Error{Code: "UNAUTHORIZED", Message: "authentication required"})
	return domain.Actor{}, false
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
	default:
		h.controller.SendError(w, r, err)
	}
}

func (h *AuthCommandsHandler) setRefreshCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   int(refreshCookieTTL.Seconds()),
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

type publicAPIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func toPublicAPIKey(k domain.APIKey) publicAPIKey {
	return publicAPIKey{
		ID:         k.ID.String(),
		Name:       k.Name,
		Scopes:     k.Scopes,
		ExpiresAt:  k.ExpiresAt,
		LastUsedAt: k.LastUsedAt,
		RevokedAt:  k.RevokedAt,
		CreatedAt:  k.CreatedAt,
	}
}

func toPublicAPIKeys(keys []domain.APIKey) []publicAPIKey {
	out := make([]publicAPIKey, 0, len(keys))
	for _, k := range keys {
		if k.RevokedAt != nil {
			continue
		}
		out = append(out, toPublicAPIKey(k))
	}
	return out
}

// authIPLimiter is a per-IP rate limiter for auth endpoints.
// 5 requests per 15 minutes per IP.
type authIPLimiter struct {
	mu       sync.Mutex
	limiters map[string]*authLimiterEntry
}

type authLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newAuthIPLimiter() *authIPLimiter {
	return &authIPLimiter{
		limiters: make(map[string]*authLimiterEntry),
	}
}

func (l *authIPLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.limiters[ip]
	if !ok {
		lim := rate.NewLimiter(rate.Every(15*time.Minute/5), 5)
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
