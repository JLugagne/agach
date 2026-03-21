package identitysvrconfig

// SsoProvider holds the configuration for a single SSO provider.
// Exactly one of SAML or OIDC must be set.
type SsoProvider struct {
	Name string      `yaml:"name"`
	Icon string      `yaml:"icon"`
	SAML *SAMLConfig `yaml:"saml,omitempty"`
	OIDC *OIDCConfig `yaml:"oidc,omitempty"`
}

// SAMLConfig holds SAML 2.0 provider settings.
type SAMLConfig struct {
	MetadataURL  string `yaml:"metadata_url"`
	EntityID     string `yaml:"entity_id"`
	ACSURL       string `yaml:"acs_url"`
	Certificate  string `yaml:"certificate,omitempty"`
}

// OIDCConfig holds OpenID Connect provider settings.
type OIDCConfig struct {
	IssuerURL    string   `yaml:"issuer_url"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url"`
	Scopes       []string `yaml:"scopes,omitempty"`
}

// SsoConfig holds the list of configured SSO providers.
type SsoConfig struct {
	Providers []SsoProvider `yaml:"providers"`
}
