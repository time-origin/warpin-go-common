package oauth

import "github.com/coreos/go-oidc/v3/oidc"

var (
	GoogleDefaultScopes = []string{oidc.ScopeOpenID, "email", "profile"}
	GitHubDefaultScopes = []string{"user:email"}
)

// ProviderConfig contains the standard OAuth authorization-code settings.
type ProviderConfig struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
}

// Config groups the providers currently implemented by this module.
type Config struct {
	Google ProviderConfig `mapstructure:"google"`
	GitHub ProviderConfig `mapstructure:"github"`
}
