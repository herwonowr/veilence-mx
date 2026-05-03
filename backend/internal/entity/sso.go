package entity

import (
	"fmt"
	"strings"
	"time"
)

// SSOProvider represents the type of SSO provider.
type SSOProvider string

const (
	SSOProviderSAML   SSOProvider = "saml"
	SSOProviderGoogle SSOProvider = "google"
	SSOProviderGitHub SSOProvider = "github"
)

// AuthProvider tracks how a user account was created/linked.
type AuthProvider string

const (
	AuthProviderLocal  AuthProvider = "local"
	AuthProviderSAML   AuthProvider = "saml"
	AuthProviderGoogle AuthProvider = "google"
	AuthProviderGitHub AuthProvider = "github"
)

// SSOState mode constants.
const (
	SSOModeLogin = "login" // SSO login (unauthenticated)
	SSOModeLink  = "link"  // identity linking from account settings
)

// SSOConfig holds a platform-level SSO configuration.
type SSOConfig struct {
	ID             string
	Provider       SSOProvider
	DisplayName    string // e.g. "Company Google", "Corporate SAML"
	IsEnabled      bool
	AllowedDomains []string // restrict SSO to these email domains
	AutoCreateUser bool     // create user account on first SSO login (default false)

	// SAML-specific fields
	SAMLEntityID      string
	SAMLSsoURL        string
	SAMLCertificate   string // PEM-encoded X.509 certificate
	SAMLAttrEmail     string
	SAMLAttrFirstName string
	SAMLAttrLastName  string

	// OAuth-specific fields (Google/GitHub)
	OAuthClientID      string
	OAuthClientSecret  string // plaintext in entity; repo encrypts/decrypts transparently
	GoogleHostedDomain string
	GitHubOrgs         []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate performs domain-level validation on SSOConfig.
func (c *SSOConfig) Validate() error {
	switch c.Provider {
	case SSOProviderSAML, SSOProviderGoogle, SSOProviderGitHub:
		// valid
	default:
		return &ValidationError{Message: fmt.Sprintf("invalid SSO provider %q", c.Provider)}
	}

	// If AutoCreateUser is true, AllowedDomains must be non-empty
	if c.AutoCreateUser && len(c.AllowedDomains) == 0 {
		return &ValidationError{Message: "allowed domains are required when auto-create user is enabled"}
	}

	// Provider-specific required fields
	switch c.Provider {
	case SSOProviderSAML:
		if c.SAMLEntityID == "" {
			return &ValidationError{Message: "SAML entity ID is required"}
		}
		if c.SAMLSsoURL == "" {
			return &ValidationError{Message: "SAML SSO URL is required"}
		}
		if c.SAMLCertificate == "" {
			return &ValidationError{Message: "SAML certificate is required"}
		}
	case SSOProviderGoogle, SSOProviderGitHub:
		if c.OAuthClientID == "" {
			return &ValidationError{Message: "OAuth client ID is required"}
		}
		if c.OAuthClientSecret == "" {
			return &ValidationError{Message: "OAuth client secret is required"}
		}
	}

	// Validate allowed domains format (no protocol, no path)
	for _, d := range c.AllowedDomains {
		if d == "" {
			return &ValidationError{Message: "allowed domain must not be empty"}
		}
		if strings.Contains(d, "://") {
			return &ValidationError{Message: fmt.Sprintf("allowed domain %q must not contain protocol", d)}
		}
		if strings.Contains(d, "/") {
			return &ValidationError{Message: fmt.Sprintf("allowed domain %q must not contain path", d)}
		}
	}

	// Set SAML attribute defaults if empty
	if c.Provider == SSOProviderSAML {
		if c.SAMLAttrEmail == "" {
			c.SAMLAttrEmail = "email"
		}
		if c.SAMLAttrFirstName == "" {
			c.SAMLAttrFirstName = "firstName"
		}
		if c.SAMLAttrLastName == "" {
			c.SAMLAttrLastName = "lastName"
		}
	}

	return nil
}

// PlatformAuthConfig holds platform-level authentication settings.
type PlatformAuthConfig struct {
	PasswordLoginEnabled bool // default true; set false to force SSO-only
	RegistrationEnabled  bool // default true
}

// UserIdentity represents a linked external identity for a user.
type UserIdentity struct {
	ID             string
	UserID         string
	Provider       AuthProvider
	ProviderUserID string // external user ID (SAML NameID, Google sub, GitHub user ID)
	Email          string // email from provider at time of linking
	Metadata       string // JSON blob: provider-specific data (org memberships, etc.)
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Validate performs domain-level validation on UserIdentity.
func (i UserIdentity) Validate() error {
	if i.UserID == "" {
		return &ValidationError{Message: "user ID is required"}
	}
	if i.Provider == "" {
		return &ValidationError{Message: "provider is required"}
	}
	if i.ProviderUserID == "" {
		return &ValidationError{Message: "provider user ID is required"}
	}
	if i.Email == "" {
		return &ValidationError{Message: "email is required"}
	}
	return nil
}

// SSOState represents a pending SSO authentication attempt (CSRF protection).
// The State field is sent as SAML RelayState or OAuth state parameter.
type SSOState struct {
	ID           string
	ConfigID     string // which SSO config initiated this flow (required)
	State        string // random state parameter, sent as SAML RelayState / OAuth state
	Provider     SSOProvider
	RedirectURL  string  // where to send user after SSO completes
	Mode         string  // SSOModeLogin or SSOModeLink
	UserID       *string // nil for login mode, set for link mode
	CodeVerifier string  // PKCE code_verifier for OAuth flows (S256 challenge method)
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// Validate performs domain-level validation on SSOState.
func (s SSOState) Validate() error {
	if s.State == "" {
		return &ValidationError{Message: "state is required"}
	}
	if s.ConfigID == "" {
		return &ValidationError{Message: "config ID is required"}
	}
	if s.Mode != SSOModeLogin && s.Mode != SSOModeLink {
		return &ValidationError{Message: fmt.Sprintf("invalid SSO mode %q", s.Mode)}
	}
	if s.Mode == SSOModeLink && (s.UserID == nil || *s.UserID == "") {
		return &ValidationError{Message: "user ID is required for link mode"}
	}
	if s.ExpiresAt.IsZero() {
		return &ValidationError{Message: "expires at is required"}
	}

	// RedirectURL validation (defense-in-depth, also validated in service layer).
	if s.RedirectURL == "" {
		return &ValidationError{Message: "redirect URL is required"}
	}
	if !strings.HasPrefix(s.RedirectURL, "/") {
		return &ValidationError{Message: "redirect URL must be a relative path starting with /"}
	}
	if strings.Contains(s.RedirectURL, "//") {
		return &ValidationError{Message: "redirect URL must not contain //"}
	}
	if strings.Contains(s.RedirectURL, "://") {
		return &ValidationError{Message: "redirect URL must not contain ://"}
	}

	return nil
}
