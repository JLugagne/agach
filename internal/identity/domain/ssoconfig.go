package domain

type SsoProvider struct {
	Name string      `yaml:"name"`
	Icon string      `yaml:"icon"`
	SAML *SAMLConfig `yaml:"saml,omitempty"`
	OIDC *OIDCConfig `yaml:"oidc,omitempty"`
}

type SAMLConfig struct {
	MetadataURL string `yaml:"metadata_url"`
	EntityID    string `yaml:"entity_id"`
	ACSURL      string `yaml:"acs_url"`
	Certificate string `yaml:"certificate,omitempty"`
}

type OIDCConfig struct {
	IssuerURL    string   `yaml:"issuer_url"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url"`
	Scopes       []string `yaml:"scopes,omitempty"`
}

type SsoConfig struct {
	Providers []SsoProvider `yaml:"providers"`
}
