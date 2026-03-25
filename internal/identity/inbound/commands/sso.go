package commands

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/pkg/apierror"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
)

// SSOCommandsHandler handles SSO authentication HTTP endpoints.
type SSOCommandsHandler struct {
	commands service.AuthCommands
	queries  service.AuthQueries
	ctrl     *controller.Controller
	cfg      domain.SsoConfig
	secret   []byte
}

// NewSSOCommandsHandler creates a new SSO handler.
func NewSSOCommandsHandler(
	cmds service.AuthCommands,
	qrs service.AuthQueries,
	ctrl *controller.Controller,
	cfg domain.SsoConfig,
	secret []byte,
) *SSOCommandsHandler {
	return &SSOCommandsHandler{
		commands: cmds,
		queries:  qrs,
		ctrl:     ctrl,
		cfg:      cfg,
		secret:   secret,
	}
}

// RegisterRoutes registers SSO routes on the router.
func (h *SSOCommandsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/auth/sso/providers", h.ListProviders).Methods("GET")
	router.HandleFunc("/api/auth/sso/{provider}/authorize", h.Authorize).Methods("GET")
	router.HandleFunc("/api/auth/sso/{provider}/callback", h.Callback).Methods("GET")
}

type publicSSOProvider struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
	Type string `json:"type"`
}

// ListProviders handles GET /api/auth/sso/providers.
// Returns the list of configured SSO providers (name, icon, type) — no secrets exposed.
func (h *SSOCommandsHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	out := make([]publicSSOProvider, 0, len(h.cfg.Providers))
	for _, p := range h.cfg.Providers {
		t := "saml"
		if p.OIDC != nil {
			t = "oidc"
		}
		out = append(out, publicSSOProvider{Name: p.Name, Icon: p.Icon, Type: t})
	}
	h.ctrl.SendSuccess(w, r, out)
}

// Authorize handles GET /api/auth/sso/{provider}/authorize.
// Redirects to the OIDC authorization endpoint with a signed state cookie.
func (h *SSOCommandsHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	providerName := mux.Vars(r)["provider"]
	prov := h.findProvider(providerName)
	if prov == nil {
		status := http.StatusNotFound
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "SSO_PROVIDER_NOT_FOUND", Message: "SSO provider not configured"})
		return
	}
	if prov.OIDC == nil {
		status := http.StatusNotImplemented
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "SSO_NOT_SUPPORTED", Message: "SAML not yet supported"})
		return
	}

	// Generate random state (32 bytes, base64url)
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		h.ctrl.SendError(w, r, err)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	sig := h.signState(state)

	// Set signed state cookie (SameSite=Lax to survive redirect)
	http.SetCookie(w, &http.Cookie{
		Name:     "sso_state_" + providerName,
		Value:    state + "|" + sig,
		Path:     "/api/auth/sso",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})

	// Perform OIDC discovery to get authorization_endpoint
	disc, err := discoverOIDC(prov.OIDC.IssuerURL)
	if err != nil {
		h.ctrl.SendError(w, r, err)
		return
	}

	// Build authorization URL
	scopes := prov.OIDC.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	authURL := disc.AuthorizationEndpoint +
		"?response_type=code" +
		"&client_id=" + prov.OIDC.ClientID +
		"&redirect_uri=" + prov.OIDC.RedirectURL +
		"&scope=" + strings.Join(scopes, "+") +
		"&state=" + state

	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles GET /api/auth/sso/{provider}/callback.
// Validates state, exchanges code, sets cookies, redirects to frontend.
func (h *SSOCommandsHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := mux.Vars(r)["provider"]
	prov := h.findProvider(providerName)
	if prov == nil {
		status := http.StatusNotFound
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "SSO_PROVIDER_NOT_FOUND", Message: "SSO provider not configured"})
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Validate state cookie
	cookie, err := r.Cookie("sso_state_" + providerName)
	if err != nil || !h.verifyState(cookie.Value, state) {
		status := http.StatusUnauthorized
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_STATE", Message: "invalid or missing SSO state"})
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "sso_state_" + providerName,
		Value:  "",
		Path:   "/api/auth/sso",
		MaxAge: -1,
	})

	redirectURI := ""
	if prov.OIDC != nil {
		redirectURI = prov.OIDC.RedirectURL
	}

	accessToken, refreshToken, err := h.commands.LoginSSO(r.Context(), providerName, code, redirectURI)
	if err != nil {
		h.handleSSOError(w, r, err)
		return
	}

	// Set refresh cookie (reuse constants from auth.go — same package)
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     refreshCookiePath,
		MaxAge:   int(refreshCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to frontend with access token in fragment
	http.Redirect(w, r, "/#sso_token="+accessToken, http.StatusFound)
}

func (h *SSOCommandsHandler) findProvider(name string) *domain.SsoProvider {
	for i := range h.cfg.Providers {
		if h.cfg.Providers[i].Name == name {
			return &h.cfg.Providers[i]
		}
	}
	return nil
}

func (h *SSOCommandsHandler) signState(state string) string {
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(state))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// verifyState checks that cookieValue == "state|sig" and that sig is valid for state.
func (h *SSOCommandsHandler) verifyState(cookieValue, state string) bool {
	parts := strings.SplitN(cookieValue, "|", 2)
	if len(parts) != 2 {
		return false
	}
	cookieState, cookieSig := parts[0], parts[1]
	if cookieState != state {
		return false
	}
	expected := h.signState(state)
	eSig, _ := base64.RawURLEncoding.DecodeString(cookieSig)
	eExp, _ := base64.RawURLEncoding.DecodeString(expected)
	return hmac.Equal(eSig, eExp)
}

func (h *SSOCommandsHandler) handleSSOError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, domain.ErrSSOProviderNotFound) {
		status := http.StatusNotFound
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "SSO_PROVIDER_NOT_FOUND", Message: err.Error()})
		return
	}
	if errors.Is(err, domain.ErrSSONotSupported) {
		status := http.StatusNotImplemented
		h.ctrl.SendFail(w, r, &status, &apierror.Error{Code: "SSO_NOT_SUPPORTED", Message: err.Error()})
		return
	}
	h.ctrl.SendError(w, r, err)
}

// oidcMeta holds the subset of OIDC discovery fields needed for Authorize.
type oidcMeta struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
}

func discoverOIDC(issuerURL string) (*oidcMeta, error) {
	u := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	resp, err := http.Get(u)
	if err != nil {
		return nil, &domain.Error{Code: "OIDC_DISCOVERY_FAILED", Message: err.Error()}
	}
	defer resp.Body.Close()
	var m oidcMeta
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, &domain.Error{Code: "OIDC_DISCOVERY_FAILED", Message: "cannot decode discovery document"}
	}
	return &m, nil
}
