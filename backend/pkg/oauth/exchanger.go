package oauth

import "net/http"

// Exchanger implements TokenExchanger for Google and GitHub OAuth providers.
type Exchanger struct {
	// callbackBaseURL is the base URL for OAuth callback endpoints.
	callbackBaseURL string

	// Google endpoint URLs
	googleTokenURL    string
	googleUserInfoURL string

	// GitHub endpoint URLs
	githubTokenURL    string
	githubUserInfoURL string
	githubEmailsURL   string
	githubOrgsURL     string
}

// ExchangerConfig holds configurable OAuth endpoint URLs.
type ExchangerConfig struct {
	CallbackBaseURL   string
	GoogleTokenURL    string
	GoogleUserInfoURL string
	GitHubTokenURL    string
	GitHubUserInfoURL string
	GitHubEmailsURL   string
	GitHubOrgsURL     string
}

// NewExchanger creates a new OAuth exchanger.
func NewExchanger(cfg ExchangerConfig) *Exchanger {
	return &Exchanger{
		callbackBaseURL:   cfg.CallbackBaseURL,
		googleTokenURL:    cfg.GoogleTokenURL,
		googleUserInfoURL: cfg.GoogleUserInfoURL,
		githubTokenURL:    cfg.GitHubTokenURL,
		githubUserInfoURL: cfg.GitHubUserInfoURL,
		githubEmailsURL:   cfg.GitHubEmailsURL,
		githubOrgsURL:     cfg.GitHubOrgsURL,
	}
}

// bearerTransport is an http.RoundTripper that attaches a Bearer token
// to every request. Used instead of oauth2.Transport to avoid subtle issues
// with Keycloak when AuthStyleInHeader is set on the endpoint config.
type bearerTransport struct {
	token string
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

// Compile-time check that Exchanger implements TokenExchanger.
var _ TokenExchanger = (*Exchanger)(nil)
