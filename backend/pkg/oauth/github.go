package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// ExchangeGitHub exchanges a GitHub OAuth authorization code for user info using PKCE.
func (e *Exchanger) ExchangeGitHub(ctx context.Context, config *OAuthConfig, code, codeVerifier string) (*OAuthUserInfo, error) {
	cfg := &oauth2.Config{
		ClientID:     config.OAuthClientID,
		ClientSecret: config.OAuthClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  e.githubTokenURL,
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		RedirectURL:  e.callbackBaseURL + "/api/auth/oauth/callback",
		Scopes:       []string{"user:email", "read:org"},
	}

	// Exchange code with PKCE code_verifier.
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}

	token, err := cfg.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("oauth.ExchangeGitHub: token exchange: %w", err)
	}

	// Build an HTTP client that attaches the Bearer token directly instead of
	// relying on oauth2.Transport, which can have subtle issues with Keycloak
	// when AuthStyleInHeader is set on the endpoint.
	client := &http.Client{
		Transport: &bearerTransport{token: token.AccessToken},
	}

	// Fetch user profile.
	user, err := fetchGitHubUser(client, e.githubUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("oauth.ExchangeGitHub: %w", err)
	}

	// Fetch primary email if not in profile.
	email := user.Email
	if email == "" {
		email, err = fetchGitHubPrimaryEmail(client, e.githubEmailsURL)
		if err != nil {
			return nil, fmt.Errorf("oauth.ExchangeGitHub: %w", err)
		}
	}

	// Fetch organization memberships.
	orgs, err := fetchGitHubOrgs(client, e.githubOrgsURL)
	if err != nil {
		// Non-fatal - org fetch may fail due to permissions.
		orgs = nil
	}

	// Split name into first/last (best effort).
	firstName, lastName := splitName(user.Name)

	return &OAuthUserInfo{
		ProviderUserID: fmt.Sprintf("%d", user.ID),
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
		AvatarURL:      user.AvatarURL,
		Organizations:  orgs,
	}, nil
}

// githubUser represents the GitHub /user response.
type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// githubEmail represents a GitHub user email entry.
type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// githubOrg represents a GitHub organization.
type githubOrg struct {
	Login string `json:"login"`
}

func fetchGitHubUser(client *http.Client, userURL string) (*githubUser, error) {
	resp, err := client.Get(userURL)
	if err != nil {
		return nil, fmt.Errorf("fetching GitHub user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("GitHub user API returned %d: %s", resp.StatusCode, string(body))
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decoding GitHub user: %w", err)
	}
	return &user, nil
}

func fetchGitHubPrimaryEmail(client *http.Client, emailsURL string) (string, error) {
	resp, err := client.Get(emailsURL)
	if err != nil {
		return "", fmt.Errorf("fetching GitHub emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("GitHub emails API returned %d: %s", resp.StatusCode, string(body))
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decoding GitHub emails: %w", err)
	}

	// Find primary verified email.
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	// Fallback to first verified email.
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found on GitHub account")
}

func fetchGitHubOrgs(client *http.Client, orgsURL string) ([]string, error) {
	resp, err := client.Get(orgsURL)
	if err != nil {
		return nil, fmt.Errorf("fetching GitHub orgs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub orgs API returned %d", resp.StatusCode)
	}

	var orgs []githubOrg
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("decoding GitHub orgs: %w", err)
	}

	result := make([]string, len(orgs))
	for i, o := range orgs {
		result[i] = o.Login
	}
	return result, nil
}

// splitName splits a full name into first and last name (best effort).
func splitName(name string) (string, string) {
	if name == "" {
		return "", ""
	}
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == ' ' {
			return name[:i], name[i+1:]
		}
	}
	return name, ""
}
