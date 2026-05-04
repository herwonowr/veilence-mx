// Package oauth provides OAuth 2.0 token exchange implementations for SSO providers.
// Types are defined locally to avoid importing internal/ packages (clean architecture).
package oauth

import "context"

// OAuthConfig holds the OAuth-related fields needed for token exchange.
// This mirrors the relevant subset of entity.SSOConfig without importing internal/.
type OAuthConfig struct {
	OAuthClientID      string
	OAuthClientSecret  string
	GoogleHostedDomain string
}

// OAuthUserInfo represents normalized user info from an OAuth provider.
// This mirrors usecase.OAuthUserInfo without importing internal/.
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	FirstName      string
	LastName       string
	AvatarURL      string
	Organizations  []string // GitHub orgs
	HostedDomain   string   // Google Workspace domain
}

// TokenExchanger exchanges authorization codes for user info.
type TokenExchanger interface {
	ExchangeGoogle(ctx context.Context, cfg *OAuthConfig, code, codeVerifier string) (*OAuthUserInfo, error)
	ExchangeGitHub(ctx context.Context, cfg *OAuthConfig, code, codeVerifier string) (*OAuthUserInfo, error)
}
