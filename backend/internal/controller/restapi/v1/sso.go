package v1

import (
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/sso"
)

// SSOHandlers handles SSO flow endpoints.
type SSOHandlers struct {
	service     *sso.Service
	frontendURL string
}

// ssoFallbackError builds the fallback error redirect when no callback URL is available.
// Used only for pre-state errors (form parse failure, missing params, state not found).
func (h *SSOHandlers) ssoFallbackError(errorCode string) string {
	return h.frontendURL + "?error=" + errorCode
}

// HandleSSOProviders returns the list of enabled SSO providers for the login page.
// GET /api/auth/sso/providers
func (h *SSOHandlers) HandleSSOProviders(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.GetEnabledProviders(r.Context())
	if err != nil {
		slog.Error("HandleSSOProviders: fetching providers", "error", err)
		respondAppError(w, Internal("failed to fetch SSO providers"))
		return
	}

	authSettings, err := h.service.GetAuthSettings(r.Context())
	if err != nil {
		slog.Error("HandleSSOProviders: fetching auth settings", "error", err)
		respondAppError(w, Internal("failed to fetch auth settings"))
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

	callbackURL := r.URL.Query().Get("callback_url")
	if callbackURL != "" && !strings.HasPrefix(callbackURL, h.frontendURL) {
		respondAppError(w, BadRequest("invalid callback_url: must be a same-origin URL"))
		return
	}

	idpURL, err := h.service.InitiateSSOLogin(r.Context(), configID, callbackURL)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		if errors.Is(err, sso.ErrSSONotEnabled) {
			respondAppError(w, BadRequest("SSO provider is not enabled"))
			return
		}
		if errors.Is(err, sso.ErrUnsupportedProvider) {
			respondAppError(w, BadRequest("unsupported SSO provider"))
			return
		}
		slog.Error("HandleSSOLogin: initiating SSO", "error", err)
		respondAppError(w, Internal("failed to initiate SSO"))
		return
	}

	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, idpURL, http.StatusFound)
}

// HandleSAMLACS processes the SAML Assertion Consumer Service callback.
// POST /api/auth/saml/acs
func (h *SSOHandlers) HandleSAMLACS(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Error("HandleSAMLACS: parsing form", "error", err)
		http.Redirect(w, r, h.ssoFallbackError("sso_failed"), http.StatusFound)
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	relayState := r.FormValue("RelayState")

	if samlResponse == "" || relayState == "" {
		slog.Error("HandleSAMLACS: missing SAMLResponse or RelayState", "hasSAML", samlResponse != "", "hasRelay", relayState != "")
		http.Redirect(w, r, h.ssoFallbackError("sso_failed"), http.StatusFound)
		return
	}

	result, err := h.service.HandleSAMLCallback(r.Context(), samlResponse, relayState, r.RemoteAddr, r.UserAgent())
	if err != nil {
		slog.Error("HandleSAMLACS: processing callback", "error", err)
		http.Redirect(w, r, h.ssoFallbackError("sso_failed"), http.StatusFound)
		return
	}

	if result.IsError {
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, h.buildErrorRedirect(result), http.StatusFound)
	} else if result.TokenPair != nil {
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, h.buildSuccessRedirect(result), http.StatusFound)
	} else {
		// Link mode success - redirect back to frontend callback URL
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, h.resolveCallbackURL(result.CallbackURL), http.StatusFound)
	}
}

// HandleSAMLSLO processes the IdP-initiated SAML Single Logout.
// POST /api/auth/saml/slo
func (h *SSOHandlers) HandleSAMLSLO(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		slog.Error("HandleSAMLSLO: parsing form", "error", err)
		respondAppError(w, BadRequest("invalid request"))
		return
	}

	samlRequest := r.FormValue("SAMLRequest")
	if samlRequest == "" {
		// Try query params (HTTP-Redirect binding).
		samlRequest = r.URL.Query().Get("SAMLRequest")
	}
	if samlRequest == "" {
		respondAppError(w, BadRequest("missing SAMLRequest"))
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
		respondAppError(w, Internal("SLO processing failed"))
		return
	}

	logoutResp := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<samlp:LogoutResponse xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" Version="2.0">` +
		`<saml:Issuer>` + html.EscapeString(h.service.GetSAMLEntityID(r.Context())) + `</saml:Issuer>` +
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
		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			slog.Error("HandleOAuthCallback: OAuth error", "error", errParam, "description", r.URL.Query().Get("error_description"))
		}
		// Try to resolve callback URL from state for proper error redirect.
		if state != "" {
			if result, err := h.service.ResolveCallbackFromState(r.Context(), state); err == nil && result != "" {
				params := url.Values{}
				params.Set("error", "sso_failed")
				if errParam != "" {
					params.Set("message", r.URL.Query().Get("error_description"))
				}
				http.Redirect(w, r, appendQueryParams(result, params), http.StatusFound)
				return
			}
		}
		http.Redirect(w, r, h.ssoFallbackError("sso_failed"), http.StatusFound)
		return
	}

	result, err := h.service.HandleOAuthCallback(r.Context(), code, state, r.RemoteAddr, r.UserAgent())
	if err != nil {
		slog.Error("HandleOAuthCallback: processing callback", "error", err)
		http.Redirect(w, r, h.ssoFallbackError("sso_failed"), http.StatusFound)
		return
	}

	if result.IsError {
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, h.buildErrorRedirect(result), http.StatusFound)
	} else if result.TokenPair != nil {
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, h.buildSuccessRedirect(result), http.StatusFound)
	} else {
		// Link mode success - redirect back to frontend callback URL
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, h.resolveCallbackURL(result.CallbackURL), http.StatusFound)
	}
}

// HandleSAMLMetadata returns SAML SP metadata XML for a config.
// GET /api/auth/saml/{configId}/metadata
func (h *SSOHandlers) HandleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "configId")
	if configID == "" {
		respondAppError(w, Validation("invalid config ID"))
		return
	}

	metadata, err := h.service.GenerateSAMLMetadata(r.Context(), configID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		slog.Error("HandleSAMLMetadata: generating metadata", "error", err)
		respondAppError(w, Internal("failed to generate SAML metadata"))
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(metadata); err != nil {
		slog.Error("HandleSAMLMetadata: writing response", "error", err)
	}
}

// HandleSAMLSPCertificate returns the SP signing certificate in PEM format.
// GET /api/auth/saml/certificate
func (h *SSOHandlers) HandleSAMLSPCertificate(w http.ResponseWriter, r *http.Request) {
	certPEM, err := h.service.GetSPCertificate(r.Context())
	if err != nil {
		slog.Error("HandleSAMLSPCertificate: getting SP certificate", "error", err)
		respondAppError(w, Internal("failed to get SP certificate"))
		return
	}

	respondJSON(w, http.StatusOK, response.SPCertificateResponse{Certificate: certPEM}, nil)
}

// HandleGetIdentities returns all linked identities for the current user.
// GET /api/auth/identities
func (h *SSOHandlers) HandleGetIdentities(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	identities, err := h.service.ListIdentities(r.Context(), userID)
	if err != nil {
		slog.Error("HandleGetIdentities: listing identities", "error", err)
		respondAppError(w, Internal("failed to list identities"))
		return
	}

	respondJSON(w, http.StatusOK, response.UserIdentitiesFromEntities(identities), nil)
}

// HandleLinkIdentity initiates the identity linking flow.
// GET /api/auth/identities/link?configId=...&redirect=...
func (h *SSOHandlers) HandleLinkIdentity(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	configID := r.URL.Query().Get("configId")
	if configID == "" {
		respondAppError(w, Validation("configId query parameter is required"))
		return
	}

	callbackURL := r.URL.Query().Get("callback_url")
	if callbackURL != "" && !strings.HasPrefix(callbackURL, h.frontendURL) {
		respondAppError(w, BadRequest("invalid callback_url: must be a same-origin URL"))
		return
	}

	idpURL, err := h.service.InitiateLinkIdentity(r.Context(), userID, configID, callbackURL)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("SSO configuration"))
			return
		}
		if errors.Is(err, sso.ErrSSONotEnabled) {
			respondAppError(w, BadRequest("SSO provider is not enabled"))
			return
		}
		slog.Error("HandleLinkIdentity: initiating link", "error", err)
		respondAppError(w, Internal("failed to initiate identity linking"))
		return
	}

	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, idpURL, http.StatusFound)
}

// HandleUnlinkIdentity removes a linked identity.
// DELETE /api/auth/identities/{id}
func (h *SSOHandlers) HandleUnlinkIdentity(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
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
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("user"))
			return
		}
		slog.Error("HandleUnlinkIdentity: unlinking", "error", err)
		respondAppError(w, Internal("failed to unlink identity"))
		return
	}

	respondJSON(w, http.StatusOK, nil, nil)
}

// resolveCallbackURL returns the callback URL if it is valid and same-origin,
// falling back to frontendURL otherwise.
func (h *SSOHandlers) resolveCallbackURL(callbackURL string) string {
	if callbackURL != "" && strings.HasPrefix(callbackURL, h.frontendURL) {
		return callbackURL
	}
	return h.frontendURL
}

// buildSuccessRedirect constructs the callback URL with tokens in query params.
func (h *SSOHandlers) buildSuccessRedirect(result *sso.SSOCallbackResult) string {
	base := h.resolveCallbackURL(result.CallbackURL)
	params := url.Values{}
	params.Set("access_token", result.TokenPair.AccessToken)
	params.Set("refresh_token", result.TokenPair.RefreshToken)
	params.Set("expires_in", fmt.Sprintf("%d", result.TokenPair.ExpiresIn))
	return appendQueryParams(base, params)
}

// buildErrorRedirect constructs the callback URL with error params.
func (h *SSOHandlers) buildErrorRedirect(result *sso.SSOCallbackResult) string {
	base := h.resolveCallbackURL(result.CallbackURL)
	params := url.Values{}
	params.Set("error", result.ErrorCode)
	if result.ErrorMsg != "" {
		params.Set("message", result.ErrorMsg)
	}
	return appendQueryParams(base, params)
}

// appendQueryParams appends query params to a URL that may already have query params.
func appendQueryParams(base string, params url.Values) string {
	if strings.Contains(base, "?") {
		return base + "&" + params.Encode()
	}
	return base + "?" + params.Encode()
}
