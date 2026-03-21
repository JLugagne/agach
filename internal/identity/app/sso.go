package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users"
	identitysvrconfig "github.com/JLugagne/agach-mcp/internal/identity/svrconfig"
	"github.com/golang-jwt/jwt/v5"
)

type SSOService struct {
	cfg    identitysvrconfig.SsoConfig
	users  users.UserRepository
	secret []byte
	http   *http.Client
}

// NewSSOService creates an SSO service backed by the provided config and user repository.
func NewSSOService(cfg identitysvrconfig.SsoConfig, u users.UserRepository, secret []byte) *SSOService {
	return &SSOService{
		cfg:    cfg,
		users:  u,
		secret: secret,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SSOService) LoginSSO(ctx context.Context, providerName, code, redirectURI string) (string, string, error) {
	prov := s.findProvider(providerName)
	if prov == nil {
		return "", "", domain.ErrSSOProviderNotFound
	}
	if prov.OIDC == nil {
		return "", "", domain.ErrSSONotSupported
	}
	return s.loginOIDC(ctx, providerName, prov.OIDC, code, redirectURI)
}

func (s *SSOService) findProvider(name string) *identitysvrconfig.SsoProvider {
	for i := range s.cfg.Providers {
		if s.cfg.Providers[i].Name == name {
			return &s.cfg.Providers[i]
		}
	}
	return nil
}

type oidcDiscovery struct {
	TokenEndpoint string `json:"token_endpoint"`
	JWKSURI       string `json:"jwks_uri"`
	Issuer        string `json:"issuer"`
}

func (s *SSOService) discover(issuerURL string) (*oidcDiscovery, error) {
	u := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	resp, err := s.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned %d", resp.StatusCode)
	}
	var d oidcDiscovery
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&d); err != nil {
		return nil, fmt.Errorf("OIDC discovery decode: %w", err)
	}
	return &d, nil
}

type tokenResponse struct {
	IDToken string `json:"id_token"`
	Error   string `json:"error"`
}

func (s *SSOService) exchangeCode(tokenEndpoint, code, redirectURI string, cfg *identitysvrconfig.OIDCConfig) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}
	resp, err := s.http.PostForm(tokenEndpoint, form)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()
	var tr tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&tr); err != nil {
		return "", fmt.Errorf("token response decode: %w", err)
	}
	if tr.Error != "" {
		return "", &domain.Error{Code: "OIDC_TOKEN_ERROR", Message: tr.Error}
	}
	if tr.IDToken == "" {
		return "", &domain.Error{Code: "OIDC_NO_ID_TOKEN", Message: "no id_token in token response"}
	}
	return tr.IDToken, nil
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Crv string `json:"crv"`
}

func (s *SSOService) fetchJWKS(jwksURI string) (*jwks, error) {
	resp, err := s.http.Get(jwksURI)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	var ks jwks
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256*1024)).Decode(&ks); err != nil {
		return nil, fmt.Errorf("JWKS decode: %w", err)
	}
	return &ks, nil
}

func (s *SSOService) validateIDToken(idToken string, disc *oidcDiscovery, cfg *identitysvrconfig.OIDCConfig) (jwt.MapClaims, error) {
	unverified, _, err := jwt.NewParser().ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, &domain.Error{Code: "OIDC_INVALID_TOKEN", Message: "invalid id_token"}
	}
	kid, _ := unverified.Header["kid"].(string)

	ks, err := s.fetchJWKS(disc.JWKSURI)
	if err != nil {
		return nil, err
	}

	parsed, err := jwt.ParseWithClaims(idToken, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); ok {
			for _, k := range ks.Keys {
				if k.Kid == kid || kid == "" {
					pub, err := parseRSAPublicKey(k.N, k.E)
					if err != nil {
						continue
					}
					return pub, nil
				}
			}
			return nil, fmt.Errorf("no matching JWK found for kid %q", kid)
		}
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); ok {
			for _, k := range ks.Keys {
				if k.Kid == kid || kid == "" {
					pub, err := parseECPublicKey(k)
					if err != nil {
						continue
					}
					return pub, nil
				}
			}
			return nil, fmt.Errorf("no matching EC JWK found for kid %q", kid)
		}
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); ok {
			return s.secret, nil
		}
		return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
	})
	if err != nil || !parsed.Valid {
		return nil, &domain.Error{Code: "OIDC_INVALID_TOKEN", Message: "id_token validation failed"}
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, &domain.Error{Code: "OIDC_INVALID_TOKEN", Message: "cannot read id_token claims"}
	}

	iss, _ := claims["iss"].(string)
	if iss != disc.Issuer && iss != strings.TrimRight(cfg.IssuerURL, "/") {
		return nil, &domain.Error{Code: "OIDC_ISS_MISMATCH", Message: "id_token issuer mismatch"}
	}

	switch aud := claims["aud"].(type) {
	case string:
		if aud != cfg.ClientID {
			return nil, &domain.Error{Code: "OIDC_AUD_MISMATCH", Message: "id_token audience mismatch"}
		}
	case []interface{}:
		found := false
		for _, a := range aud {
			if a.(string) == cfg.ClientID {
				found = true
				break
			}
		}
		if !found {
			return nil, &domain.Error{Code: "OIDC_AUD_MISMATCH", Message: "id_token audience mismatch"}
		}
	default:
		return nil, &domain.Error{Code: "OIDC_AUD_MISMATCH", Message: "id_token audience missing"}
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, &domain.Error{Code: "OIDC_NO_SUB", Message: "id_token missing sub claim"}
	}

	return claims, nil
}

func (s *SSOService) loginOIDC(ctx context.Context, providerName string, cfg *identitysvrconfig.OIDCConfig, code, redirectURI string) (string, string, error) {
	disc, err := s.discover(cfg.IssuerURL)
	if err != nil {
		return "", "", err
	}

	idToken, err := s.exchangeCode(disc.TokenEndpoint, code, redirectURI, cfg)
	if err != nil {
		return "", "", err
	}

	claims, err := s.validateIDToken(idToken, disc, cfg)
	if err != nil {
		return "", "", err
	}

	sub := claims["sub"].(string)
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	if email == "" {
		email = sub + "@sso.local"
	}
	if name == "" {
		name = providerName + "-user"
	}

	user, err := s.users.FindBySSO(ctx, providerName, sub)
	if err != nil {
		if !isNotFound(err) {
			return "", "", err
		}
		now := time.Now()
		user = domain.User{
			ID:          domain.NewUserID(),
			Email:       email,
			DisplayName: name,
			SSOProvider: providerName,
			SSOSubject:  sub,
			Role:        domain.RoleMember,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.users.Create(ctx, user); err != nil {
			return "", "", err
		}
	}

	accessToken, err := issueToken(user, tokenTypeAccess, accessTokenTTL, s.secret)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := issueToken(user, tokenTypeRefresh, refreshTokenTTL, s.secret)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func isNotFound(err error) bool {
	return errors.Is(err, domain.ErrUserNotFound)
}

func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	return &rsa.PublicKey{N: n, E: e}, nil
}

func parseECPublicKey(k jwk) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, err
	}
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve: %s", k.Crv)
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}
