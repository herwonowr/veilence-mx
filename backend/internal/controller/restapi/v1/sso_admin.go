package v1

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/sso"
)

// createSSOConfigRequest is the request body for POST /api/admin/sso.
type createSSOConfigRequest struct {
	Provider       string   `json:"provider"`
	DisplayName    string   `json:"displayName"`
	IsEnabled      bool     `json:"isEnabled"`
	AutoCreateUser bool     `json:"autoCreateUser"`
	AllowedDomains []string `json:"allowedDomains"`
	// SAML
	SAMLEntityID      *string `json:"samlEntityId"`
	SAMLSsoURL        *string `json:"samlSsoUrl"`
	SAMLCertificate   *string `json:"samlCertificate"`
	SAMLAttrEmail     *string `json:"samlAttrEmail"`
	SAMLAttrFirstName *string `json:"samlAttrFirstName"`
	SAMLAttrLastName  *string `json:"samlAttrLastName"`
	// OAuth
	OAuthClientID      *string  `json:"oauthClientId"`
	OAuthClientSecret  *string  `json:"oauthClientSecret"`
	GoogleHostedDomain *string  `json:"googleHostedDomain"`
	GitHubOrgs         []string `json:"githubOrgs"`
}

// updateSSOConfigRequest is the request body for PUT /api/admin/sso/{id}.
type updateSSOConfigRequest struct {
	DisplayName    *string   `json:"displayName,omitempty"`
	IsEnabled      *bool     `json:"isEnabled,omitempty"`
	AutoCreateUser *bool     `json:"autoCreateUser,omitempty"`
	AllowedDomains *[]string `json:"allowedDomains,omitempty"`
	// SAML
	SAMLEntityID      *string `json:"samlEntityId,omitempty"`
	SAMLSsoURL        *string `json:"samlSsoUrl,omitempty"`
	SAMLCertificate   *string `json:"samlCertificate,omitempty"`
	SAMLAttrEmail     *string `json:"samlAttrEmail,omitempty"`
	SAMLAttrFirstName *string `json:"samlAttrFirstName,omitempty"`
	SAMLAttrLastName  *string `json:"samlAttrLastName,omitempty"`
	// OAuth
	OAuthClientID      *string   `json:"oauthClientId,omitempty"`
	OAuthClientSecret  *string   `json:"oauthClientSecret,omitempty"`
	GoogleHostedDomain *string   `json:"googleHostedDomain,omitempty"`
	GitHubOrgs         *[]string `json:"githubOrgs,omitempty"`
}

// updatePlatformAuthSettingsRequest is the request body for PUT /api/admin/auth-settings.
type updatePlatformAuthSettingsRequest struct {
	PasswordLoginEnabled *bool `json:"passwordLoginEnabled"`
}

// HandleListSSOConfigs returns all platform SSO configs.
// GET /api/admin/sso
func (h *SSOHandlers) HandleListSSOConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.GetSSOConfigs(r.Context())
	if err != nil {
		slog.Error("HandleListSSOConfigs: listing configs", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list SSO configs")
		return
	}

	respondJSON(w, http.StatusOK, response.SSOConfigsFromEntities(configs), nil)
}

// HandleCreateSSOConfig creates a platform SSO configuration.
// POST /api/admin/sso
func (h *SSOHandlers) HandleCreateSSOConfig(w http.ResponseWriter, r *http.Request) {
	var req createSSOConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Provider == "" {
		respondAppError(w, Validation("provider is required"))
		return
	}

	config := &entity.SSOConfig{
		Provider:       entity.SSOProvider(req.Provider),
		DisplayName:    req.DisplayName,
		IsEnabled:      req.IsEnabled,
		AutoCreateUser: req.AutoCreateUser,
		AllowedDomains: req.AllowedDomains,
		GitHubOrgs:     req.GitHubOrgs,
	}

	if req.SAMLEntityID != nil {
		config.SAMLEntityID = *req.SAMLEntityID
	}
	if req.SAMLSsoURL != nil {
		config.SAMLSsoURL = *req.SAMLSsoURL
	}
	if req.SAMLCertificate != nil {
		config.SAMLCertificate = *req.SAMLCertificate
	}
	if req.SAMLAttrEmail != nil {
		config.SAMLAttrEmail = *req.SAMLAttrEmail
	}
	if req.SAMLAttrFirstName != nil {
		config.SAMLAttrFirstName = *req.SAMLAttrFirstName
	}
	if req.SAMLAttrLastName != nil {
		config.SAMLAttrLastName = *req.SAMLAttrLastName
	}
	if req.OAuthClientID != nil {
		config.OAuthClientID = *req.OAuthClientID
	}
	if req.OAuthClientSecret != nil {
		config.OAuthClientSecret = *req.OAuthClientSecret
	}
	if req.GoogleHostedDomain != nil {
		config.GoogleHostedDomain = *req.GoogleHostedDomain
	}

	created, err := h.service.CreateSSOConfig(r.Context(), config)
	if err != nil {
		if errors.Is(err, entity.ErrValidation) {
			respondAppError(w, Validation(err.Error()))
			return
		}
		slog.Error("HandleCreateSSOConfig: creating config", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create SSO config")
		return
	}

	respondJSON(w, http.StatusCreated, response.SSOConfigFromEntity(*created), nil)
}

// HandleGetSSOConfig returns a specific SSO config.
// GET /api/admin/sso/{id}
func (h *SSOHandlers) HandleGetSSOConfig(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "id")
	if configID == "" {
		respondAppError(w, Validation("config ID is required"))
		return
	}

	config, err := h.service.GetSSOConfigByID(r.Context(), configID)
	if err != nil {
		if errors.Is(err, sso.ErrSSONotConfigured) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		slog.Error("HandleGetSSOConfig: fetching config", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch SSO config")
		return
	}

	respondJSON(w, http.StatusOK, response.SSOConfigFromEntity(*config), nil)
}

// HandleUpdateSSOConfig updates an SSO configuration.
// PUT /api/admin/sso/{id}
func (h *SSOHandlers) HandleUpdateSSOConfig(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "id")
	if configID == "" {
		respondAppError(w, Validation("config ID is required"))
		return
	}

	var req updateSSOConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Fetch existing config to merge partial updates.
	existing, err := h.service.GetSSOConfigByID(r.Context(), configID)
	if err != nil {
		if errors.Is(err, sso.ErrSSONotConfigured) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		slog.Error("HandleUpdateSSOConfig: fetching existing config", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch SSO config")
		return
	}

	// Apply partial updates.
	if req.DisplayName != nil {
		existing.DisplayName = *req.DisplayName
	}
	if req.IsEnabled != nil {
		existing.IsEnabled = *req.IsEnabled
	}
	if req.AutoCreateUser != nil {
		existing.AutoCreateUser = *req.AutoCreateUser
	}
	if req.AllowedDomains != nil {
		existing.AllowedDomains = *req.AllowedDomains
	}
	if req.SAMLEntityID != nil {
		existing.SAMLEntityID = *req.SAMLEntityID
	}
	if req.SAMLSsoURL != nil {
		existing.SAMLSsoURL = *req.SAMLSsoURL
	}
	if req.SAMLCertificate != nil {
		existing.SAMLCertificate = *req.SAMLCertificate
	}
	if req.SAMLAttrEmail != nil {
		existing.SAMLAttrEmail = *req.SAMLAttrEmail
	}
	if req.SAMLAttrFirstName != nil {
		existing.SAMLAttrFirstName = *req.SAMLAttrFirstName
	}
	if req.SAMLAttrLastName != nil {
		existing.SAMLAttrLastName = *req.SAMLAttrLastName
	}
	if req.OAuthClientID != nil {
		existing.OAuthClientID = *req.OAuthClientID
	}
	if req.OAuthClientSecret != nil {
		existing.OAuthClientSecret = *req.OAuthClientSecret
	}
	if req.GoogleHostedDomain != nil {
		existing.GoogleHostedDomain = *req.GoogleHostedDomain
	}
	if req.GitHubOrgs != nil {
		existing.GitHubOrgs = *req.GitHubOrgs
	}

	updated, err := h.service.UpdateSSOConfig(r.Context(), configID, existing)
	if err != nil {
		if errors.Is(err, entity.ErrValidation) {
			respondAppError(w, Validation(err.Error()))
			return
		}
		slog.Error("HandleUpdateSSOConfig: updating config", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update SSO config")
		return
	}

	respondJSON(w, http.StatusOK, response.SSOConfigFromEntity(*updated), nil)
}

// HandleDeleteSSOConfig removes an SSO configuration.
// DELETE /api/admin/sso/{id}
func (h *SSOHandlers) HandleDeleteSSOConfig(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "id")
	if configID == "" {
		respondAppError(w, Validation("config ID is required"))
		return
	}

	if err := h.service.DeleteSSOConfig(r.Context(), configID); err != nil {
		if errors.Is(err, sso.ErrSSONotConfigured) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		slog.Error("HandleDeleteSSOConfig: deleting config", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete SSO config")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleTestSSOConfig tests IdP connectivity for an SSO config.
// POST /api/admin/sso/{id}/test
func (h *SSOHandlers) HandleTestSSOConfig(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "id")
	if configID == "" {
		respondAppError(w, Validation("config ID is required"))
		return
	}

	result, err := h.service.TestSSOConfig(r.Context(), configID)
	if err != nil {
		if errors.Is(err, sso.ErrSSONotConfigured) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		respondJSON(w, http.StatusOK, response.TestSSOConfigResponse{
			Success: false,
			Message: err.Error(),
		}, nil)
		return
	}

	respondJSON(w, http.StatusOK, response.TestSSOConfigResponse{
		Success:     true,
		Message:     "SSO configuration test passed",
		IdpEntityID: result.IdpEntityID,
		IdpSSOURL:   result.IdpSSOURL,
	}, nil)
}

// importSAMLMetadataRequest is the request body for POST /api/admin/sso/saml/import-metadata.
type importSAMLMetadataRequest struct {
	MetadataURL string `json:"metadataUrl"`
}

// HandleImportSAMLMetadata fetches SAML metadata from a URL and returns extracted fields.
// POST /api/admin/sso/saml/import-metadata
func (h *SSOHandlers) HandleImportSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	var req importSAMLMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.MetadataURL == "" {
		respondAppError(w, Validation("metadataUrl is required"))
		return
	}

	parsed, err := url.Parse(req.MetadataURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		respondAppError(w, Validation("metadataUrl must be a valid HTTP or HTTPS URL"))
		return
	}

	info, err := h.service.ImportSAMLMetadata(r.Context(), req.MetadataURL)
	if err != nil {
		slog.Error("HandleImportSAMLMetadata: importing metadata", "error", err)
		respondError(w, http.StatusBadRequest, "Failed to import SAML metadata: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, response.SAMLMetadataImportResponse{
		EntityID:    info.EntityID,
		SSOURL:      info.SSOURL,
		SloURL:      info.SloURL,
		Certificate: info.Certificate,
	}, nil)
}

// HandleGetAuthSettings returns platform auth settings.
// GET /api/admin/auth-settings
func (h *SSOHandlers) HandleGetAuthSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetAuthSettings(r.Context())
	if err != nil {
		slog.Error("HandleGetAuthSettings: fetching settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch auth settings")
		return
	}

	respondJSON(w, http.StatusOK, response.PlatformAuthSettingsResponse{
		PasswordLoginEnabled: settings.PasswordLoginEnabled,
		RegistrationEnabled:  settings.RegistrationEnabled,
	}, nil)
}

// HandleUpdateAuthSettings updates platform auth settings.
// PUT /api/admin/auth-settings
func (h *SSOHandlers) HandleUpdateAuthSettings(w http.ResponseWriter, r *http.Request) {
	var req updatePlatformAuthSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	existing, err := h.service.GetAuthSettings(r.Context())
	if err != nil {
		slog.Error("HandleUpdateAuthSettings: fetching existing settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch auth settings")
		return
	}

	if req.PasswordLoginEnabled != nil {
		existing.PasswordLoginEnabled = *req.PasswordLoginEnabled
	}

	if err := h.service.UpdateAuthSettings(r.Context(), existing); err != nil {
		if errors.Is(err, entity.ErrValidation) {
			respondAppError(w, Validation(err.Error()))
			return
		}
		slog.Error("HandleUpdateAuthSettings: updating settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update auth settings")
		return
	}

	respondJSON(w, http.StatusOK, response.PlatformAuthSettingsResponse{
		PasswordLoginEnabled: existing.PasswordLoginEnabled,
		RegistrationEnabled:  existing.RegistrationEnabled,
	}, nil)
}
