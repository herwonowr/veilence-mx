package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

// Exchanger implements usecase.OAuthTokenExchanger for Google and GitHub.
type Exchanger struct {
	// callbackBaseURL is the base URL for OAuth callback endpoints.
	callbackBaseURL string
}

// NewExchanger creates a new OAuth exchanger.
func NewExchanger(callbackBaseURL string) *Exchanger {
	return &Exchanger{
		callbackBaseURL: callbackBaseURL,
	}
}

// ExchangeGoogle exchanges a Google OAuth authorization code for user info using PKCE.
func (e *Exchanger) ExchangeGoogle(ctx context.Context, config *OAuthConfig, code, codeVerifier string) (*OAuthUserInfo, error) {
	cfg := &oauth2.Config{
		ClientID:     config.OAuthClientID,
		ClientSecret: config.OAuthClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  e.callbackBaseURL + "/api/auth/oauth/callback",
		Scopes:       []string{"openid", "email", "profile"},
	}

	// Exchange code with PKCE code_verifier.
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}

	token, err := cfg.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("oauth.ExchangeGoogle: token exchange: %w", err)
	}

	// Fetch user info from Google.
	client := cfg.Client(ctx, token)
	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("oauth.ExchangeGoogle: fetching user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oauth.ExchangeGoogle: user info returned %d: %s", resp.StatusCode, string(body))
	}

	var gUser googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("oauth.ExchangeGoogle: decoding user info: %w", err)
	}

	return &OAuthUserInfo{
		ProviderUserID: gUser.Sub,
		Email:          gUser.Email,
		FirstName:      gUser.GivenName,
		LastName:       gUser.FamilyName,
		AvatarURL:      gUser.Picture,
		HostedDomain:   gUser.HD,
	}, nil
}

// googleUserInfo represents the Google userinfo response.
type googleUserInfo struct {
	Sub        string `json:"sub"`
	Email      string `json:"email"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture    string `json:"picture"`
	HD         string `json:"hd"` // hosted domain for Google Workspace
}

// Compile-time check that Exchanger implements TokenExchanger.
var _ TokenExchanger = (*Exchanger)(nil)
