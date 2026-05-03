package v1

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/sso"
)

// SSOHandlers handles SSO flow endpoints.
type SSOHandlers struct {
	service     *sso.Service
	frontendURL string
}

// HandleSSOProviders returns the list of enabled SSO providers for the login page.
// GET /api/auth/sso/providers
func (h *SSOHandlers) HandleSSOProviders(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.GetEnabledProviders(r.Context())
	if err != nil {
		slog.Error("HandleSSOProviders: fetching providers", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch SSO providers")
		return
	}

	authSettings, err := h.service.GetAuthSettings(r.Context())
	if err != nil {
		slog.Error("HandleSSOProviders: fetching auth settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch auth settings")
		return
	}

	providers := make([]response.SSOProviderInfo, len(configs))
	for i, c := range configs {
		providers[i] = response.SSOProviderInfo{
			ConfigID:    c.ID,
			Provider:    string(c.Provider),
			DisplayName: c.DisplayName,
		}
	}

	respondJSON(w, http.StatusOK, response.SSOProvidersResponse{
		Providers:            providers,
		PasswordLoginEnabled: authSettings.PasswordLoginEnabled,
		RegistrationEnabled:  authSettings.RegistrationEnabled,
	}, nil)
}

// HandleSSOLogin initiates SSO login for a specific config.
// GET /api/auth/sso/{configId}/login
func (h *SSOHandlers) HandleSSOLogin(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "configId")
	if configID == "" {
		respondAppError(w, Validation("config ID is required"))
		return
	}

	redirectURL := r.URL.Query().Get("redirect")

	idpURL, err := h.service.InitiateSSOLogin(r.Context(), configID, redirectURL)
	if err != nil {
		if errors.Is(err, sso.ErrSSONotConfigured) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		if errors.Is(err, sso.ErrSSONotEnabled) {
			respondAppError(w, BadRequest("SSO provider is not enabled"))
			return
		}
		slog.Error("HandleSSOLogin: initiating SSO", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to initiate SSO")
		return
	}

	http.Redirect(w, r, idpURL, http.StatusFound)
}

// HandleSAMLACS processes the SAML Assertion Consumer Service callback.
// POST /api/auth/saml/acs
func (h *SSOHandlers) HandleSAMLACS(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Error("HandleSAMLACS: parsing form", "error", err)
		http.Redirect(w, r, h.frontendURL+"?error=sso_failed", http.StatusFound)
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	relayState := r.FormValue("RelayState")

	if samlResponse == "" || relayState == "" {
		http.Redirect(w, r, h.frontendURL+"?error=sso_failed", http.StatusFound)
		return
	}

	result, err := h.service.HandleSAMLCallback(r.Context(), samlResponse, relayState)
	if err != nil {
		slog.Error("HandleSAMLACS: processing callback", "error", err)
		http.Redirect(w, r, h.frontendURL+"?error=sso_failed", http.StatusFound)
		return
	}

	if !result.IsError && result.TokenPair != nil {
		setAuthCookies(w, result.TokenPair)
	}

	http.Redirect(w, r, result.RedirectURL, http.StatusFound)
}

// HandleSAMLSLO processes the IdP-initiated SAML Single Logout.
// POST /api/auth/saml/slo
func (h *SSOHandlers) HandleSAMLSLO(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Error("HandleSAMLSLO: parsing form", "error", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	samlRequest := r.FormValue("SAMLRequest")
	if samlRequest == "" {
		// Try query params (HTTP-Redirect binding).
		samlRequest = r.URL.Query().Get("SAMLRequest")
	}
	if samlRequest == "" {
		http.Error(w, "missing SAMLRequest", http.StatusBadRequest)
		return
	}

	// Extract signature parameters (form values for HTTP-POST, query params for HTTP-Redirect).
	signature := r.FormValue("Signature")
	sigAlg := r.FormValue("SigAlg")
	relayState := r.FormValue("RelayState")
	if signature == "" {
		signature = r.URL.Query().Get("Signature")
	}
	if sigAlg == "" {
		sigAlg = r.URL.Query().Get("SigAlg")
	}
	if relayState == "" {
		relayState = r.URL.Query().Get("RelayState")
	}

	if err := h.service.HandleSAMLSLO(r.Context(), samlRequest, signature, sigAlg, relayState); err != nil {
		slog.Error("HandleSAMLSLO: processing SLO", "error", err)
		http.Error(w, "SLO processing failed", http.StatusInternalServerError)
		return
	}

	logoutResp := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<samlp:LogoutResponse xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" Version="2.0">` +
		`<saml:Issuer>` + h.service.GetSAMLEntityID() + `</saml:Issuer>` +
		`<samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status>` +
		`</samlp:LogoutResponse>`
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(logoutResp)); err != nil {
		slog.Error("HandleSAMLSLO: writing response", "error", err)
	}
}

// HandleOAuthCallback processes the unified OAuth callback.
// GET /api/auth/oauth/callback
func (h *SSOHandlers) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		// Check for OAuth error response.
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			slog.Error("HandleOAuthCallback: OAuth error", "error", errParam, "description", r.URL.Query().Get("error_description"))
		}
		http.Redirect(w, r, h.frontendURL+"?error=sso_failed", http.StatusFound)
		return
	}

	result, err := h.service.HandleOAuthCallback(r.Context(), code, state)
	if err != nil {
		slog.Error("HandleOAuthCallback: processing callback", "error", err)
		http.Redirect(w, r, h.frontendURL+"?error=sso_failed", http.StatusFound)
		return
	}

	if !result.IsError && result.TokenPair != nil {
		setAuthCookies(w, result.TokenPair)
	}

	http.Redirect(w, r, result.RedirectURL, http.StatusFound)
}

// HandleSAMLMetadata returns SAML SP metadata XML for a config.
// GET /api/auth/saml/{configId}/metadata
func (h *SSOHandlers) HandleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "configId")
	if configID == "" {
		respondError(w, http.StatusBadRequest, "invalid config ID")
		return
	}

	metadata, err := h.service.GenerateSAMLMetadata(r.Context(), configID)
	if err != nil {
		slog.Error("HandleSAMLMetadata: generating metadata", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to generate SAML metadata")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(metadata); err != nil {
		slog.Error("HandleSAMLMetadata: writing response", "error", err)
	}
}

// HandleGetIdentities returns all linked identities for the current user.
// GET /api/auth/identities
func (h *SSOHandlers) HandleGetIdentities(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	identities, err := h.service.ListIdentities(r.Context(), userID)
	if err != nil {
		slog.Error("HandleGetIdentities: listing identities", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list identities")
		return
	}

	respondJSON(w, http.StatusOK, response.UserIdentitiesFromEntities(identities), nil)
}

// HandleLinkIdentity initiates the identity linking flow.
// GET /api/auth/identities/link?configId=...&redirect=...
func (h *SSOHandlers) HandleLinkIdentity(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	configID := r.URL.Query().Get("configId")
	if configID == "" {
		respondAppError(w, Validation("configId query parameter is required"))
		return
	}

	redirectURL := r.URL.Query().Get("redirect")

	idpURL, err := h.service.InitiateLinkIdentity(r.Context(), userID, configID, redirectURL)
	if err != nil {
		if errors.Is(err, sso.ErrSSONotConfigured) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		slog.Error("HandleLinkIdentity: initiating link", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to initiate identity linking")
		return
	}

	http.Redirect(w, r, idpURL, http.StatusFound)
}

// HandleUnlinkIdentity removes a linked identity.
// DELETE /api/auth/identities/{id}
func (h *SSOHandlers) HandleUnlinkIdentity(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	identityID := chi.URLParam(r, "id")
	if identityID == "" {
		respondAppError(w, Validation("identity ID is required"))
		return
	}

	if err := h.service.UnlinkIdentity(r.Context(), userID, identityID); err != nil {
		if errors.Is(err, sso.ErrIdentityNotFound) {
			respondAppError(w, NotFound("identity"))
			return
		}
		if errors.Is(err, sso.ErrCannotUnlinkLast) {
			respondAppError(w, BadRequest("cannot unlink last identity without a password"))
			return
		}
		slog.Error("HandleUnlinkIdentity: unlinking", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to unlink identity")
		return
	}

	respondJSON(w, http.StatusOK, nil, nil)
}

// setAuthCookies sets the access and refresh token cookies on the response.
func setAuthCookies(w http.ResponseWriter, tp *sso.TokenPair) {
	if tp == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tp.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(tp.ExpiresIn),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tp.RefreshToken,
		Path:     "/api/auth/refresh",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 30, // 30 days
	})
}
