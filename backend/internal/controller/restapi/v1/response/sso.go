package response

import (
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// SSOConfigResponse is the JSON representation of an SSO configuration.
// NOTE: oauthClientSecret is NEVER included - it is write-only.
type SSOConfigResponse struct {
	ID             string   `json:"id"`
	Provider       string   `json:"provider"`
	DisplayName    string   `json:"displayName"`
	IsEnabled      bool     `json:"isEnabled"`
	AutoCreateUser bool     `json:"autoCreateUser"`
	AllowedDomains []string `json:"allowedDomains"`
	// SAML
	SAMLEntityID      *string `json:"samlEntityId"`
	SAMLSsoURL        *string `json:"samlSsoUrl"`
	SAMLCertificate   *string `json:"samlCertificate"`
	SAMLAttrEmail     string  `json:"samlAttrEmail"`
	SAMLAttrFirstName string  `json:"samlAttrFirstName"`
	SAMLAttrLastName  string  `json:"samlAttrLastName"`
	// OAuth (oauthClientSecret is NEVER returned - write-only field)
	OAuthClientID      *string  `json:"oauthClientId"`
	GoogleHostedDomain *string  `json:"googleHostedDomain"`
	GitHubOrgs         []string `json:"githubOrgs"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
}

// SSOConfigFromEntity maps a domain SSOConfig to a response DTO.
func SSOConfigFromEntity(e entity.SSOConfig) SSOConfigResponse {
	resp := SSOConfigResponse{
		ID:                e.ID,
		Provider:          string(e.Provider),
		DisplayName:       e.DisplayName,
		IsEnabled:         e.IsEnabled,
		AutoCreateUser:    e.AutoCreateUser,
		AllowedDomains:    e.AllowedDomains,
		SAMLAttrEmail:     e.SAMLAttrEmail,
		SAMLAttrFirstName: e.SAMLAttrFirstName,
		SAMLAttrLastName:  e.SAMLAttrLastName,
		GitHubOrgs:        e.GitHubOrgs,
		CreatedAt:         e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         e.UpdatedAt.Format(time.RFC3339),
	}
	if e.SAMLEntityID != "" {
		resp.SAMLEntityID = &e.SAMLEntityID
	}
	if e.SAMLSsoURL != "" {
		resp.SAMLSsoURL = &e.SAMLSsoURL
	}
	if e.SAMLCertificate != "" {
		resp.SAMLCertificate = &e.SAMLCertificate
	}
	if e.OAuthClientID != "" {
		resp.OAuthClientID = &e.OAuthClientID
	}
	if e.GoogleHostedDomain != "" {
		resp.GoogleHostedDomain = &e.GoogleHostedDomain
	}
	// NOTE: OAuthClientSecret is intentionally omitted - never returned in API responses
	if resp.AllowedDomains == nil {
		resp.AllowedDomains = []string{}
	}
	if resp.GitHubOrgs == nil {
		resp.GitHubOrgs = []string{}
	}
	return resp
}

// SSOConfigsFromEntities maps a slice of domain SSOConfigs to response DTOs.
func SSOConfigsFromEntities(configs []entity.SSOConfig) []SSOConfigResponse {
	result := make([]SSOConfigResponse, len(configs))
	for i := range configs {
		result[i] = SSOConfigFromEntity(configs[i])
	}
	return result
}

// UserIdentityResponse is the JSON representation of a linked external identity.
type UserIdentityResponse struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"`
	ProviderUserID string `json:"providerUserId"`
	ProviderEmail  string `json:"providerEmail"`
	LinkedAt       string `json:"linkedAt"`
}

// UserIdentityFromEntity maps a domain UserIdentity to a response DTO.
func UserIdentityFromEntity(e entity.UserIdentity) UserIdentityResponse {
	return UserIdentityResponse{
		ID:             e.ID,
		Provider:       string(e.Provider),
		ProviderUserID: e.ProviderUserID,
		ProviderEmail:  e.Email,
		LinkedAt:       e.CreatedAt.Format(time.RFC3339),
	}
}

// UserIdentitiesFromEntities maps a slice of domain UserIdentities to response DTOs.
func UserIdentitiesFromEntities(identities []entity.UserIdentity) []UserIdentityResponse {
	result := make([]UserIdentityResponse, len(identities))
	for i := range identities {
		result[i] = UserIdentityFromEntity(identities[i])
	}
	return result
}

// SSOProviderInfo represents a single enabled SSO provider for the login page.
type SSOProviderInfo struct {
	ConfigID    string `json:"id"`
	Provider    string `json:"provider"`
	DisplayName string `json:"displayName"`
}

// SSOProvidersResponse is the response for GET /api/auth/sso/providers.
type SSOProvidersResponse struct {
	Providers            []SSOProviderInfo `json:"providers"`
	PasswordLoginEnabled bool              `json:"passwordLoginEnabled"`
	RegistrationEnabled  bool              `json:"registrationEnabled"`
}

// TestSSOConfigResponse is the response for SSO config connectivity test.
type TestSSOConfigResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	IdpEntityID string `json:"idpEntityId,omitempty"`
	IdpSSOURL   string `json:"idpSsoUrl,omitempty"`
}

// SAMLMetadataImportResponse is the response for SAML metadata URL import.
type SAMLMetadataImportResponse struct {
	EntityID    string `json:"entityId"`
	SSOURL      string `json:"ssoUrl"`
	SloURL      string `json:"sloUrl,omitempty"`
	Certificate string `json:"certificate"`
}

// PlatformAuthSettingsResponse is the response for GET /api/admin/auth-settings.
type PlatformAuthSettingsResponse struct {
	PasswordLoginEnabled bool `json:"passwordLoginEnabled"`
	RegistrationEnabled  bool `json:"registrationEnabled"`
}

// SPCertificateResponse is the response for GET /api/auth/saml/certificate.
type SPCertificateResponse struct {
	Certificate string `json:"certificate"`
}
