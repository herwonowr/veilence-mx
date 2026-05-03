package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// Sentinel errors returned by SSO operations.
var (
	ErrSSONotConfigured    = errors.New("sso not configured")
	ErrSSONotEnabled       = errors.New("sso not enabled")
	ErrSSOStateNotFound    = errors.New("sso state not found or expired")
	ErrIdentityConflict    = errors.New("identity already linked to another account")
	ErrCannotUnlinkLast    = errors.New("cannot unlink last identity without password")
	ErrIdentityNotFound    = errors.New("identity not found")
	ErrUnsupportedProvider = errors.New("unsupported sso provider")
	ErrDomainNotAllowed    = errors.New("email domain not allowed")
	ErrAccountDeactivated  = errors.New("account deactivated")
	ErrIdentityNotLinked   = errors.New("identity not linked")
	ErrAccountNotFound     = errors.New("account not found")
)

const (
	// PKCEVerifierLength is the byte length of PKCE code_verifier (generates 43-char base64url).
	PKCEVerifierLength = 32
)

// SSOCallbackResult holds the outcome of an SSO callback.
type SSOCallbackResult struct {
	RedirectURL string     // where to redirect the browser
	TokenPair   *TokenPair // nil if error redirect
	IsError     bool       // true = redirect is an error page, no cookies
}

// Service provides SSO business logic operations.
type Service struct {
	configRepo          SSOConfigRepository
	identityRepo        UserIdentityRepository
	stateRepo           SSOStateRepository
	samlProvider        SAMLProvider
	oauthExchanger      OAuthTokenExchanger
	authService         AuthSessionCreator
	userFinder          usecase.UserRepository
	userCreator         usecase.UserAccountCreator
	sessionRepo         SessionRepository
	refreshTokenRepo    RefreshTokenRepository
	settingRepo         usecase.SettingRepository
	auditLogger         AuditLogger
	metadataFetcher     SAMLMetadataFetcher
	registrationEnabled bool
	stateTTL            time.Duration
	baseURL             string
}

// NewService creates a new SSO service.
func NewService(
	configRepo SSOConfigRepository,
	identityRepo UserIdentityRepository,
	stateRepo SSOStateRepository,
	samlProvider SAMLProvider,
	oauthExchanger OAuthTokenExchanger,
	authService AuthSessionCreator,
	userFinder usecase.UserRepository,
	userCreator usecase.UserAccountCreator,
	sessionRepo SessionRepository,
	refreshTokenRepo RefreshTokenRepository,
	settingRepo usecase.SettingRepository,
	stateTTL time.Duration,
	baseURL string,
	opts ...ServiceOption,
) *Service {
	s := &Service{
		configRepo:       configRepo,
		identityRepo:     identityRepo,
		stateRepo:        stateRepo,
		samlProvider:     samlProvider,
		oauthExchanger:   oauthExchanger,
		authService:      authService,
		userFinder:       userFinder,
		userCreator:      userCreator,
		sessionRepo:      sessionRepo,
		refreshTokenRepo: refreshTokenRepo,
		settingRepo:      settingRepo,
		stateTTL:         stateTTL,
		baseURL:          baseURL,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ServiceOption configures optional dependencies for the SSO service.
type ServiceOption func(*Service)

// WithAuditLogger sets the audit logger.
func WithAuditLogger(al AuditLogger) ServiceOption {
	return func(s *Service) { s.auditLogger = al }
}

// WithMetadataFetcher sets the SAML metadata fetcher.
func WithMetadataFetcher(mf SAMLMetadataFetcher) ServiceOption {
	return func(s *Service) { s.metadataFetcher = mf }
}

// WithRegistrationEnabled sets the registration enabled flag (from env config).
func WithRegistrationEnabled(enabled bool) ServiceOption {
	return func(s *Service) { s.registrationEnabled = enabled }
}

// --- Public SSO (unauthenticated) ---

// GetEnabledProviders returns enabled SSO providers for the login page.
func (s *Service) GetEnabledProviders(ctx context.Context) ([]entity.SSOConfig, error) {
	configs, err := s.configRepo.FindEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetEnabledProviders: %w", err)
	}
	return configs, nil
}

// InitiateSSOLogin starts SSO for an unauthenticated user on the login page.
// Returns the redirect URL to the IdP.
func (s *Service) InitiateSSOLogin(ctx context.Context, configID, redirectURL string) (string, error) {
	config, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return "", fmt.Errorf("InitiateSSOLogin: %w", err)
	}
	if !config.IsEnabled {
		return "", ErrSSONotEnabled
	}

	// Validate redirect URL - must be a relative path or empty.
	if redirectURL != "" && !isValidRedirectURL(redirectURL) {
		redirectURL = "/"
	}
	if redirectURL == "" {
		redirectURL = "/dashboard"
	}

	// Generate random state.
	stateToken, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("InitiateSSOLogin: generating state: %w", err)
	}

	ssoState := &entity.SSOState{
		ConfigID:    configID,
		State:       stateToken,
		Provider:    config.Provider,
		RedirectURL: redirectURL,
		Mode:        entity.SSOModeLogin,
		ExpiresAt:   time.Now().Add(s.stateTTL),
	}

	var idpURL string

	switch config.Provider {
	case entity.SSOProviderSAML:
		idpURL, err = s.samlProvider.GenerateAuthnRequest(config)
		if err != nil {
			return "", fmt.Errorf("InitiateSSOLogin: generating SAML request: %w", err)
		}
		// Append RelayState to the SAML redirect URL.
		if strings.Contains(idpURL, "?") {
			idpURL += "&RelayState=" + stateToken
		} else {
			idpURL += "?RelayState=" + stateToken
		}

	case entity.SSOProviderGoogle, entity.SSOProviderGitHub:
		// Generate PKCE code verifier.
		codeVerifier, codeChallenge, genErr := generatePKCE()
		if genErr != nil {
			return "", fmt.Errorf("InitiateSSOLogin: generating PKCE: %w", genErr)
		}
		ssoState.CodeVerifier = codeVerifier
		idpURL = buildOAuthURL(config, stateToken, codeChallenge, s.baseURL)

	default:
		return "", ErrUnsupportedProvider
	}

	if err := s.stateRepo.Create(ctx, ssoState); err != nil {
		return "", fmt.Errorf("InitiateSSOLogin: storing state: %w", err)
	}

	return idpURL, nil
}

// --- SSO Callbacks ---

// HandleSAMLCallback processes the SAML ACS callback.
func (s *Service) HandleSAMLCallback(ctx context.Context, samlResponse, relayState string) (*SSOCallbackResult, error) {
	// Look up state.
	ssoState, err := s.stateRepo.FindByState(ctx, relayState)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, ErrSSOStateNotFound
		}
		return nil, fmt.Errorf("HandleSAMLCallback: finding state: %w", err)
	}
	defer func() {
		_ = s.stateRepo.Delete(ctx, ssoState.ID)
	}()

	// Get SSO config.
	config, err := s.configRepo.FindByID(ctx, ssoState.ConfigID)
	if err != nil {
		return nil, fmt.Errorf("HandleSAMLCallback: finding config: %w", err)
	}

	// Validate SAML response.
	assertion, err := s.samlProvider.ValidateResponse(config, samlResponse)
	if err != nil {
		return nil, fmt.Errorf("HandleSAMLCallback: validating SAML response: %w", err)
	}

	if ssoState.Mode == entity.SSOModeLink {
		// Handle identity linking
		_, linkErr := s.LinkIdentity(ctx, *ssoState.UserID, entity.AuthProvider(config.Provider), assertion.NameID, assertion.Email)
		if linkErr != nil {
			return &SSOCallbackResult{
				RedirectURL: appendError(ssoState.RedirectURL, "internal_error", linkErr.Error()),
				IsError:     true,
			}, nil
		}
		return &SSOCallbackResult{RedirectURL: ssoState.RedirectURL}, nil
	}

	// Login mode - resolve user and issue JWT
	return s.resolveAndIssueJWT(ctx, assertion.Email, assertion.FirstName, assertion.LastName, entity.SSOProvider(config.Provider), assertion.NameID, config.ID, ssoState)
}

// HandleOAuthCallback processes the OAuth callback for Google or GitHub.
func (s *Service) HandleOAuthCallback(ctx context.Context, code, state string) (*SSOCallbackResult, error) {
	// Look up state.
	ssoState, err := s.stateRepo.FindByState(ctx, state)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, ErrSSOStateNotFound
		}
		return nil, fmt.Errorf("HandleOAuthCallback: finding state: %w", err)
	}
	defer func() {
		_ = s.stateRepo.Delete(ctx, ssoState.ID)
	}()

	// Get SSO config.
	config, err := s.configRepo.FindByID(ctx, ssoState.ConfigID)
	if err != nil {
		return nil, fmt.Errorf("HandleOAuthCallback: finding config: %w", err)
	}

	// Exchange code for user info using PKCE code_verifier.
	var userInfo *usecase.OAuthUserInfo
	switch ssoState.Provider {
	case entity.SSOProviderGoogle:
		userInfo, err = s.oauthExchanger.ExchangeGoogle(ctx, config, code, ssoState.CodeVerifier)
	case entity.SSOProviderGitHub:
		userInfo, err = s.oauthExchanger.ExchangeGitHub(ctx, config, code, ssoState.CodeVerifier)
	default:
		return nil, ErrUnsupportedProvider
	}
	if err != nil {
		return nil, fmt.Errorf("HandleOAuthCallback: exchanging code: %w", err)
	}

	// Validate GitHub org membership
	if len(config.GitHubOrgs) > 0 && config.Provider == entity.SSOProviderGitHub {
		if !isOrgMember(userInfo.Organizations, config.GitHubOrgs) {
			return &SSOCallbackResult{
				RedirectURL: appendError(ssoState.RedirectURL, "domain_not_allowed", "Not a member of any allowed GitHub organization"),
				IsError:     true,
			}, nil
		}
	}

	// Validate Google hosted domain
	if config.GoogleHostedDomain != "" && config.Provider == entity.SSOProviderGoogle {
		if !strings.EqualFold(userInfo.HostedDomain, config.GoogleHostedDomain) {
			return &SSOCallbackResult{
				RedirectURL: appendError(ssoState.RedirectURL, "domain_not_allowed", "Google account is not from the required hosted domain"),
				IsError:     true,
			}, nil
		}
	}

	if ssoState.Mode == entity.SSOModeLink {
		// Handle identity linking
		_, linkErr := s.LinkIdentity(ctx, *ssoState.UserID, entity.AuthProvider(ssoState.Provider), userInfo.ProviderUserID, userInfo.Email)
		if linkErr != nil {
			return &SSOCallbackResult{
				RedirectURL: appendError(ssoState.RedirectURL, "internal_error", linkErr.Error()),
				IsError:     true,
			}, nil
		}
		return &SSOCallbackResult{RedirectURL: ssoState.RedirectURL}, nil
	}

	// Login mode - resolve user and issue JWT
	return s.resolveAndIssueJWT(ctx, userInfo.Email, userInfo.FirstName, userInfo.LastName, entity.SSOProvider(config.Provider), userInfo.ProviderUserID, config.ID, ssoState)
}

// resolveAndIssueJWT implements the user resolution order from the spec.
func (s *Service) resolveAndIssueJWT(ctx context.Context, email, firstName, lastName string, provider entity.SSOProvider, providerUserID, configID string, state *entity.SSOState) (*SSOCallbackResult, error) {
	// 1. Load config
	config, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: %w", err)
	}

	// 2. Domain restriction check
	if len(config.AllowedDomains) > 0 && !emailInDomains(email, config.AllowedDomains) {
		return &SSOCallbackResult{RedirectURL: appendError(state.RedirectURL, "domain_not_allowed", ""), IsError: true}, nil
	}

	// 3a. Lookup by provider+providerUserID
	identity, _ := s.identityRepo.FindByProviderAndProviderUserID(ctx, entity.AuthProvider(provider), providerUserID)
	if identity != nil {
		user, findErr := s.userFinder.FindByID(ctx, identity.UserID)
		if findErr != nil {
			return nil, fmt.Errorf("resolveAndIssueJWT: finding user: %w", findErr)
		}
		if !user.IsActive {
			return &SSOCallbackResult{RedirectURL: appendError(state.RedirectURL, "account_deactivated", ""), IsError: true}, nil
		}
		tokenPair, tokenErr := s.authService.CreateSessionForUser(ctx, identity.UserID, string(provider))
		if tokenErr != nil {
			return nil, fmt.Errorf("resolveAndIssueJWT: creating session: %w", tokenErr)
		}
		return &SSOCallbackResult{RedirectURL: state.RedirectURL, TokenPair: tokenPair}, nil
	}

	// 3b. Lookup by email (case-insensitive) - REJECT if found
	user, _ := s.userFinder.FindByEmail(ctx, strings.ToLower(email))
	if user != nil {
		return &SSOCallbackResult{
			RedirectURL: appendError(state.RedirectURL, "identity_not_linked", "An account with this email exists. Link your SSO identity from account settings first."),
			IsError:     true,
		}, nil
	}

	// 3c. User not found - auto-create?
	if !config.AutoCreateUser {
		return &SSOCallbackResult{RedirectURL: appendError(state.RedirectURL, "account_not_found", ""), IsError: true}, nil
	}

	// Create new user (is_superadmin=false, is_active=true, email_verified=true)
	newUser, createErr := s.userCreator.CreateSSOUser(ctx, strings.ToLower(email), firstName, lastName, provider)
	if createErr != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: creating user: %w", createErr)
	}

	createIdentityErr := s.identityRepo.Create(ctx, &entity.UserIdentity{
		UserID:         newUser.ID,
		Provider:       entity.AuthProvider(provider),
		ProviderUserID: providerUserID,
		Email:          email,
	})
	if createIdentityErr != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: creating identity: %w", createIdentityErr)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "user.sso_created", "user", newUser.ID, fmt.Sprintf("provider=%s config_id=%s", provider, configID))
	}

	tokenPair, tokenErr := s.authService.CreateSessionForUser(ctx, newUser.ID, string(provider))
	if tokenErr != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: creating session: %w", tokenErr)
	}
	return &SSOCallbackResult{RedirectURL: state.RedirectURL, TokenPair: tokenPair}, nil
}

// --- SAML Single Logout ---

// HandleSAMLSLO receives an IdP-initiated LogoutRequest and invalidates JWT sessions.
// signature and sigAlg are extracted from the request (form values or query params).
func (s *Service) HandleSAMLSLO(ctx context.Context, samlRequest, signature, sigAlg, relayState string) error {
	slog.Info("HandleSAMLSLO: processing IdP-initiated logout")

	// Parse the SAML LogoutRequest to extract the NameID and Issuer.
	nameID, issuer, parseErr := s.samlProvider.ParseLogoutRequest(samlRequest)
	if parseErr != nil {
		return fmt.Errorf("HandleSAMLSLO: parsing logout request: %w", parseErr)
	}

	if nameID == "" {
		return fmt.Errorf("HandleSAMLSLO: no NameID in logout request")
	}

	slog.Info("HandleSAMLSLO: parsed logout request", "nameID", nameID, "issuer", issuer)

	// Look up the SSO config by the issuer (entity ID) to get the signing certificate.
	if issuer == "" {
		return fmt.Errorf("HandleSAMLSLO: no Issuer in logout request")
	}
	config, err := s.configRepo.FindBySAMLEntityID(ctx, issuer)
	if err != nil {
		return fmt.Errorf("HandleSAMLSLO: finding SSO config by issuer: %w", err)
	}

	// Verify the signature on the LogoutRequest before invalidating any sessions.
	if verifyErr := s.samlProvider.VerifyLogoutSignature(samlRequest, signature, sigAlg, config.SAMLCertificate); verifyErr != nil {
		slog.Warn("HandleSAMLSLO: signature verification failed", "issuer", issuer, "error", verifyErr)
		return fmt.Errorf("HandleSAMLSLO: signature verification failed: %w", verifyErr)
	}

	// Find the user identity by the external NameID.
	identity, err := s.identityRepo.FindByProviderAndProviderUserID(ctx, entity.AuthProviderSAML, nameID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			slog.Warn("HandleSAMLSLO: no identity found for NameID", "nameID", nameID)
			return nil
		}
		return fmt.Errorf("HandleSAMLSLO: finding identity: %w", err)
	}

	// Delete all JWT sessions and refresh tokens for this user.
	if err := s.sessionRepo.DeleteByUserID(ctx, identity.UserID); err != nil {
		return fmt.Errorf("HandleSAMLSLO: deleting sessions: %w", err)
	}
	if err := s.refreshTokenRepo.DeleteByUserID(ctx, identity.UserID); err != nil {
		return fmt.Errorf("HandleSAMLSLO: deleting refresh tokens: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "user.slo_initiated", "user", identity.UserID, fmt.Sprintf("nameID=%s", nameID))
	}

	slog.Info("HandleSAMLSLO: sessions invalidated", "userID", identity.UserID, "nameID", nameID)
	return nil
}

// --- Identity Linking ---

// LinkIdentity links an existing user to an SSO provider.
func (s *Service) LinkIdentity(ctx context.Context, userID string, provider entity.AuthProvider, providerUserID, email string) (*entity.UserIdentity, error) {
	// Check if this provider identity is already linked to another user.
	existing, err := s.identityRepo.FindByProviderAndProviderUserID(ctx, provider, providerUserID)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return nil, fmt.Errorf("LinkIdentity: checking existing: %w", err)
	}
	if existing != nil && existing.UserID != userID {
		return nil, ErrIdentityConflict
	}
	if existing != nil && existing.UserID == userID {
		// Already linked - update email if changed.
		if existing.Email != email {
			existing.Email = email
			existing.UpdatedAt = time.Now()
			if err := s.identityRepo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("LinkIdentity: updating: %w", err)
			}
		}
		return existing, nil
	}

	identity := &entity.UserIdentity{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
	}
	if err := s.identityRepo.Create(ctx, identity); err != nil {
		return nil, fmt.Errorf("LinkIdentity: creating: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.identity_linked", "user_identity", identity.ID, fmt.Sprintf("provider=%s", provider))
	}

	return identity, nil
}

// UnlinkIdentity removes an SSO identity link.
func (s *Service) UnlinkIdentity(ctx context.Context, userID, identityID string) error {
	identities, err := s.identityRepo.FindByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("UnlinkIdentity: listing identities: %w", err)
	}

	// Find the target identity and verify it belongs to this user.
	var target *entity.UserIdentity
	for i := range identities {
		if identities[i].ID == identityID {
			target = &identities[i]
			break
		}
	}
	if target == nil {
		return ErrIdentityNotFound
	}

	// Check that the user still has a password or at least one other identity.
	user, err := s.userFinder.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("UnlinkIdentity: finding user: %w", err)
	}
	hasPassword := user.PasswordHash != ""
	otherIdentities := len(identities) - 1

	if !hasPassword && otherIdentities == 0 {
		return ErrCannotUnlinkLast
	}

	if err := s.identityRepo.Delete(ctx, identityID); err != nil {
		return fmt.Errorf("UnlinkIdentity: deleting: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.identity_unlinked", "user_identity", identityID, fmt.Sprintf("provider=%s", target.Provider))
	}

	return nil
}

// ListIdentities returns all linked identities for a user.
func (s *Service) ListIdentities(ctx context.Context, userID string) ([]entity.UserIdentity, error) {
	identities, err := s.identityRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ListIdentities: %w", err)
	}
	return identities, nil
}

// InitiateLinkIdentity starts an identity linking flow for a user.
// Returns the redirect URL for the IdP.
func (s *Service) InitiateLinkIdentity(ctx context.Context, userID, configID, redirectURL string) (string, error) {
	config, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return "", fmt.Errorf("InitiateLinkIdentity: %w", err)
	}
	if !config.IsEnabled {
		return "", ErrSSONotEnabled
	}

	if redirectURL == "" {
		redirectURL = "/account"
	}
	if !isValidRedirectURL(redirectURL) {
		redirectURL = "/account"
	}

	// Generate random state.
	stateToken, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("InitiateLinkIdentity: generating state: %w", err)
	}

	ssoState := &entity.SSOState{
		ConfigID:    configID,
		State:       stateToken,
		UserID:      &userID,
		Provider:    config.Provider,
		RedirectURL: redirectURL,
		Mode:        entity.SSOModeLink,
		ExpiresAt:   time.Now().Add(s.stateTTL),
	}

	var idpURL string
	switch config.Provider {
	case entity.SSOProviderSAML:
		idpURL, err = s.samlProvider.GenerateAuthnRequest(config)
		if err != nil {
			return "", fmt.Errorf("InitiateLinkIdentity: generating SAML request: %w", err)
		}
		if strings.Contains(idpURL, "?") {
			idpURL += "&RelayState=" + stateToken
		} else {
			idpURL += "?RelayState=" + stateToken
		}

	case entity.SSOProviderGoogle, entity.SSOProviderGitHub:
		codeVerifier, codeChallenge, genErr := generatePKCE()
		if genErr != nil {
			return "", fmt.Errorf("InitiateLinkIdentity: generating PKCE: %w", genErr)
		}
		ssoState.CodeVerifier = codeVerifier
		idpURL = buildOAuthURL(config, stateToken, codeChallenge, s.baseURL)

	default:
		return "", ErrUnsupportedProvider
	}

	if err := s.stateRepo.Create(ctx, ssoState); err != nil {
		return "", fmt.Errorf("InitiateLinkIdentity: storing state: %w", err)
	}

	return idpURL, nil
}

// --- SSO Config Management (super-admin) ---

// GetSSOConfigs returns all SSO configs (platform-scoped).
func (s *Service) GetSSOConfigs(ctx context.Context) ([]entity.SSOConfig, error) {
	configs, err := s.configRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetSSOConfigs: %w", err)
	}
	return configs, nil
}

// GetSSOConfigByID returns an SSO config by ID.
func (s *Service) GetSSOConfigByID(ctx context.Context, id string) (*entity.SSOConfig, error) {
	config, err := s.configRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetSSOConfigByID: %w", err)
	}
	return config, nil
}

// CreateSSOConfig creates a platform-level SSO configuration.
func (s *Service) CreateSSOConfig(ctx context.Context, config *entity.SSOConfig) (*entity.SSOConfig, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("CreateSSOConfig: %w", err)
	}

	if err := s.configRepo.Create(ctx, config); err != nil {
		return nil, fmt.Errorf("CreateSSOConfig: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.config_created", "sso_config", config.ID, fmt.Sprintf("provider=%s", config.Provider))
	}

	return config, nil
}

// UpdateSSOConfig updates an SSO configuration.
func (s *Service) UpdateSSOConfig(ctx context.Context, id string, config *entity.SSOConfig) (*entity.SSOConfig, error) {
	config.ID = id
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("UpdateSSOConfig: %w", err)
	}

	if err := s.configRepo.Update(ctx, config); err != nil {
		return nil, fmt.Errorf("UpdateSSOConfig: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.config_updated", "sso_config", id, fmt.Sprintf("provider=%s", config.Provider))
	}

	return config, nil
}

// DeleteSSOConfig removes an SSO configuration.
func (s *Service) DeleteSSOConfig(ctx context.Context, id string) error {
	if err := s.configRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("DeleteSSOConfig: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.config_deleted", "sso_config", id, "")
	}

	return nil
}

// TestSSOConfig validates IdP connectivity for an SSO config.
// SAML: fetches metadata URL if available, otherwise validates certificate parsing.
// OAuth: validates config completeness (full connectivity test requires user
// interaction via redirect flow, so we only verify config fields are valid).
func (s *Service) TestSSOConfig(ctx context.Context, id string) (*usecase.SSOTestResult, error) {
	config, err := s.configRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TestSSOConfig: %w", err)
	}

	switch config.Provider {
	case entity.SSOProviderSAML:
		// Validate the IdP certificate can be parsed as a valid PEM/X.509 cert.
		if err := validateSAMLCertificate(config.SAMLCertificate); err != nil {
			return nil, fmt.Errorf("TestSSOConfig: invalid SAML certificate: %w", err)
		}
		return &usecase.SSOTestResult{
			IdpEntityID: config.SAMLEntityID,
			IdpSSOURL:   config.SAMLSsoURL,
		}, nil

	case entity.SSOProviderGoogle, entity.SSOProviderGitHub:
		// Full OAuth connectivity test would require user interaction (redirect
		// flow), so we validate config completeness via Validate().
		if err := config.Validate(); err != nil {
			return nil, err
		}
		return &usecase.SSOTestResult{}, nil
	}
	return nil, ErrUnsupportedProvider
}

// ImportSAMLMetadata fetches SAML metadata from the given URL and returns the extracted fields.
func (s *Service) ImportSAMLMetadata(ctx context.Context, metadataURL string) (*usecase.SAMLMetadataInfo, error) {
	if s.metadataFetcher == nil {
		return nil, fmt.Errorf("ImportSAMLMetadata: SAML metadata import is not available")
	}

	info, err := s.metadataFetcher.FetchAndParse(ctx, metadataURL)
	if err != nil {
		return nil, fmt.Errorf("ImportSAMLMetadata: %w", err)
	}

	return info, nil
}

// GenerateSAMLMetadata generates SAML SP metadata XML for a config.
func (s *Service) GenerateSAMLMetadata(ctx context.Context, configID string) ([]byte, error) {
	config, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("GenerateSAMLMetadata: %w", err)
	}
	metadata, err := s.samlProvider.GenerateMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("GenerateSAMLMetadata: %w", err)
	}
	return metadata, nil
}

// GetSAMLEntityID returns the base SP entity ID used in SAML responses.
func (s *Service) GetSAMLEntityID() string {
	return s.baseURL
}

// --- Platform Auth Settings ---

// GetAuthSettings returns platform auth configuration.
func (s *Service) GetAuthSettings(ctx context.Context) (*entity.PlatformAuthConfig, error) {
	keys := []string{"auth.password_login_enabled"}
	settings, err := s.settingRepo.FindPlatformSettings(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("GetAuthSettings: %w", err)
	}

	config := &entity.PlatformAuthConfig{
		PasswordLoginEnabled: true,
		RegistrationEnabled:  s.registrationEnabled,
	}
	for _, s := range settings {
		switch s.Key {
		case "auth.password_login_enabled":
			config.PasswordLoginEnabled = s.Value != "false"
		}
	}
	return config, nil
}

// UpdateAuthSettings updates platform auth configuration.
func (s *Service) UpdateAuthSettings(ctx context.Context, config *entity.PlatformAuthConfig) error {
	// Validation: cannot disable password login unless at least one SSO config is enabled
	if !config.PasswordLoginEnabled {
		enabled, err := s.configRepo.FindEnabled(ctx)
		if err != nil {
			return fmt.Errorf("UpdateAuthSettings: %w", err)
		}
		if len(enabled) == 0 {
			return fmt.Errorf("UpdateAuthSettings: cannot disable password login without at least one enabled SSO provider")
		}
	}

	passwordVal := "true"
	if !config.PasswordLoginEnabled {
		passwordVal = "false"
	}

	if err := s.settingRepo.UpsertPlatformSetting(ctx, "auth.password_login_enabled", passwordVal); err != nil {
		return fmt.Errorf("UpdateAuthSettings: %w", err)
	}

	return nil
}

// --- Cleanup ---

// CleanupExpired removes expired SSOState records.
func (s *Service) CleanupExpired(ctx context.Context) error {
	statesDeleted, err := s.stateRepo.DeleteExpired(ctx)
	if err != nil {
		return fmt.Errorf("CleanupExpired: deleting expired states: %w", err)
	}

	if statesDeleted > 0 {
		slog.Info("CleanupExpired: removed expired records", "states", statesDeleted)
	}

	return nil
}

// --- Certificate Validation ---

// validateSAMLCertificate checks that a PEM-encoded certificate can be parsed.
func validateSAMLCertificate(pemCert string) error {
	block, _ := pem.Decode([]byte(pemCert))
	if block == nil {
		return fmt.Errorf("no PEM block found in certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parsing X.509 certificate: %w", err)
	}
	if time.Now().After(cert.NotAfter) {
		return fmt.Errorf("certificate expired at %s", cert.NotAfter.Format(time.RFC3339))
	}
	return nil
}

// --- Internal Helpers ---

// generateRandomString generates a cryptographically random hex string.
func generateRandomString(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generatePKCE generates a PKCE code_verifier and code_challenge (S256).
func generatePKCE() (codeVerifier, codeChallenge string, err error) {
	b := make([]byte, PKCEVerifierLength)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	codeVerifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(codeVerifier))
	codeChallenge = base64.RawURLEncoding.EncodeToString(h[:])
	return codeVerifier, codeChallenge, nil
}

// buildOAuthURL constructs the OAuth authorization URL with PKCE parameters.
func buildOAuthURL(config *entity.SSOConfig, state, codeChallenge, baseURL string) string {
	switch config.Provider {
	case entity.SSOProviderGoogle:
		params := url.Values{}
		params.Set("client_id", config.OAuthClientID)
		params.Set("redirect_uri", baseURL+"/api/auth/oauth/callback")
		params.Set("response_type", "code")
		params.Set("scope", "openid email profile")
		params.Set("state", state)
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
		params.Set("access_type", "offline")
		if config.GoogleHostedDomain != "" {
			params.Set("hd", config.GoogleHostedDomain)
		}
		return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()

	case entity.SSOProviderGitHub:
		params := url.Values{}
		params.Set("client_id", config.OAuthClientID)
		params.Set("redirect_uri", baseURL+"/api/auth/oauth/callback")
		params.Set("state", state)
		params.Set("scope", "user:email read:org")
		return "https://github.com/login/oauth/authorize?" + params.Encode()

	default:
		return ""
	}
}

// isValidRedirectURL checks that a redirect URL is safe (relative path only).
func isValidRedirectURL(redirectURL string) bool {
	if !strings.HasPrefix(redirectURL, "/") || strings.HasPrefix(redirectURL, "//") {
		return false
	}
	parsed, err := url.Parse(redirectURL)
	if err != nil {
		return false
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return false
	}
	return true
}

// emailInDomains checks if an email address belongs to one of the allowed domains.
func emailInDomains(email string, domains []string) bool {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])
	for _, d := range domains {
		if domain == strings.ToLower(strings.TrimSpace(d)) {
			return true
		}
	}
	return false
}

// isOrgMember checks if any user org matches any allowed org.
func isOrgMember(userOrgs, allowedOrgs []string) bool {
	for _, allowed := range allowedOrgs {
		for _, userOrg := range userOrgs {
			if strings.EqualFold(userOrg, allowed) {
				return true
			}
		}
	}
	return false
}

// appendError appends ?error=code&message=msg to a URL (handles existing query params).
func appendError(baseURL, code, message string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + "?error=" + code
	}
	q := u.Query()
	q.Set("error", code)
	if message != "" {
		q.Set("message", message)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
