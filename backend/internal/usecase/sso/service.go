package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// Sentinel errors returned by SSO operations.
var (
	ErrSSONotEnabled       = errors.New("sso not enabled")
	ErrSSOStateNotFound    = errors.New("sso state not found or expired")
	ErrCannotUnlinkLast    = errors.New("cannot unlink last identity without password")
	ErrIdentityNotFound    = errors.New("identity not found")
	ErrUnsupportedProvider = errors.New("unsupported sso provider")
	ErrIdentityConflict    = errors.New("identity already linked to another account")
)

const (
	// pkceVerifierLength is the byte length of PKCE code_verifier (generates 43-char base64url).
	pkceVerifierLength = 32
)

// SSOCallbackResult holds the outcome of an SSO callback.
type SSOCallbackResult struct {
	CallbackURL string             // full frontend callback URL to redirect to
	TokenPair   *usecase.TokenPair // nil if error redirect
	IsError     bool               // true = redirect is an error page, no cookies
	ErrorCode   string             // error code for the frontend (e.g. "account_not_found")
	ErrorMsg    string             // optional human-readable error message
}

// Service provides SSO business logic operations.
type Service struct {
	configRepo          usecase.SSOConfigRepository
	identityRepo        usecase.UserIdentityRepository
	stateRepo           usecase.SSOStateRepository
	samlProvider        usecase.SAMLProvider
	oauthExchanger      usecase.OAuthTokenExchanger
	authService         usecase.AuthSessionCreator
	userFinder          usecase.UserRepository
	userCreator         usecase.UserAccountCreator
	sessionRepo         usecase.SessionRepository
	refreshTokenRepo    usecase.RefreshTokenRepository
	settingRepo         usecase.SettingRepository
	auditLogger         usecase.AuditLogger
	metadataFetcher     usecase.SAMLMetadataFetcher
	registrationEnabled bool
	stateTTL            time.Duration
	baseURL             string
	// spKey is the cached SP signing private key (loaded/generated on first use).
	spKey *rsa.PrivateKey
	// spCertPEM is the cached SP signing certificate in PEM format.
	spCertPEM string
	// spKeyOnce ensures EnsureSPSigningKey only generates/loads once.
	spKeyOnce sync.Once
	// spKeyErr stores the error from the first EnsureSPSigningKey call.
	spKeyErr error
}

// NewService creates a new SSO service.
func NewService(
	configRepo usecase.SSOConfigRepository,
	identityRepo usecase.UserIdentityRepository,
	stateRepo usecase.SSOStateRepository,
	samlProvider usecase.SAMLProvider,
	oauthExchanger usecase.OAuthTokenExchanger,
	authService usecase.AuthSessionCreator,
	userFinder usecase.UserRepository,
	userCreator usecase.UserAccountCreator,
	sessionRepo usecase.SessionRepository,
	refreshTokenRepo usecase.RefreshTokenRepository,
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
func WithAuditLogger(al usecase.AuditLogger) ServiceOption {
	return func(s *Service) { s.auditLogger = al }
}

// WithMetadataFetcher sets the SAML metadata fetcher.
func WithMetadataFetcher(mf usecase.SAMLMetadataFetcher) ServiceOption {
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
func (s *Service) InitiateSSOLogin(ctx context.Context, configID, callbackURL string) (string, error) {
	config, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return "", fmt.Errorf("InitiateSSOLogin: %w", err)
	}
	if !config.IsEnabled {
		return "", ErrSSONotEnabled
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
		CallbackURL: callbackURL,
		Mode:        entity.SSOModeLogin,
		ExpiresAt:   time.Now().Add(s.stateTTL),
	}

	var idpURL string

	switch config.Provider {
	case entity.SSOProviderSAML:
		var requestID string
		idpURL, requestID, err = s.samlProvider.GenerateAuthnRequest(config)
		if err != nil {
			return "", fmt.Errorf("InitiateSSOLogin: generating SAML request: %w", err)
		}
		ssoState.SAMLRequestID = requestID
		// Append RelayState to the SAML redirect URL.
		if strings.Contains(idpURL, "?") {
			idpURL += "&RelayState=" + stateToken
		} else {
			idpURL += "?RelayState=" + stateToken
		}

	case entity.SSOProviderGoogle, entity.SSOProviderGitHub:
		// Generate PKCE code verifier (only effective for Google; GitHub ignores PKCE).
		var codeChallenge string
		if config.Provider == entity.SSOProviderGoogle {
			codeVerifier, cc, genErr := generatePKCE()
			if genErr != nil {
				return "", fmt.Errorf("InitiateSSOLogin: generating PKCE: %w", genErr)
			}
			ssoState.CodeVerifier = codeVerifier
			codeChallenge = cc
		}
		idpURL = buildOAuthURL(config, stateToken, codeChallenge, s.baseURL)

	default:
		return "", ErrUnsupportedProvider
	}

	if err := ssoState.Validate(); err != nil {
		return "", fmt.Errorf("InitiateSSOLogin: validating state: %w", err)
	}

	if err := s.stateRepo.Create(ctx, ssoState); err != nil {
		return "", fmt.Errorf("InitiateSSOLogin: storing state: %w", err)
	}

	return idpURL, nil
}

// --- SSO Callbacks ---

// HandleSAMLCallback processes the SAML ACS callback.
func (s *Service) HandleSAMLCallback(ctx context.Context, samlResponse, relayState, ipAddress, userAgent string) (*SSOCallbackResult, error) {
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
	assertion, err := s.samlProvider.ValidateResponse(config, samlResponse, ssoState.SAMLRequestID)
	if err != nil {
		return nil, fmt.Errorf("HandleSAMLCallback: validating SAML response: %w", err)
	}

	if ssoState.Mode == entity.SSOModeLink {
		// Handle identity linking
		_, linkErr := s.LinkIdentity(ctx, *ssoState.UserID, entity.AuthProvider(config.Provider), assertion.NameID, assertion.Email)
		if linkErr != nil {
			errorCode := "internal_error"
			errorMsg := "Failed to link identity. Please try again."
			if errors.Is(linkErr, ErrIdentityConflict) {
				errorCode = "identity_conflict"
				errorMsg = "This identity is already linked to another account."
			}
			if s.auditLogger != nil {
				s.auditLogger.LogActionWithUser(ctx, *ssoState.UserID, assertion.Email, "sso.identity_link_failed", "user_identity", *ssoState.UserID, fmt.Sprintf("email=%s provider=%s reason=%s", assertion.Email, config.Provider, errorCode))
			}
			return &SSOCallbackResult{
				CallbackURL: ssoState.CallbackURL,
				IsError:     true,
				ErrorCode:   errorCode,
				ErrorMsg:    errorMsg,
			}, nil
		}
		return &SSOCallbackResult{CallbackURL: ssoState.CallbackURL}, nil
	}

	// Login mode - resolve user and issue JWT
	return s.resolveAndIssueJWT(ctx, assertion.Email, assertion.FirstName, assertion.LastName, entity.SSOProvider(config.Provider), assertion.NameID, config, ssoState, ipAddress, userAgent)
}

// HandleOAuthCallback processes the OAuth callback for Google or GitHub.
func (s *Service) HandleOAuthCallback(ctx context.Context, code, state, ipAddress, userAgent string) (*SSOCallbackResult, error) {
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
				CallbackURL: ssoState.CallbackURL,
				IsError:     true,
				ErrorCode:   "domain_not_allowed",
				ErrorMsg:    "Not a member of any allowed GitHub organization",
			}, nil
		}
	}

	// Validate Google hosted domain
	if config.GoogleHostedDomain != "" && config.Provider == entity.SSOProviderGoogle {
		if !strings.EqualFold(userInfo.HostedDomain, config.GoogleHostedDomain) {
			return &SSOCallbackResult{
				CallbackURL: ssoState.CallbackURL,
				IsError:     true,
				ErrorCode:   "domain_not_allowed",
				ErrorMsg:    "Google account is not from the required hosted domain",
			}, nil
		}
	}

	if ssoState.Mode == entity.SSOModeLink {
		// Handle identity linking
		_, linkErr := s.LinkIdentity(ctx, *ssoState.UserID, entity.AuthProvider(ssoState.Provider), userInfo.ProviderUserID, userInfo.Email)
		if linkErr != nil {
			errorCode := "internal_error"
			errorMsg := "Failed to link identity. Please try again."
			if errors.Is(linkErr, ErrIdentityConflict) {
				errorCode = "identity_conflict"
				errorMsg = "This identity is already linked to another account."
			}
			if s.auditLogger != nil {
				s.auditLogger.LogActionWithUser(ctx, *ssoState.UserID, userInfo.Email, "sso.identity_link_failed", "user_identity", *ssoState.UserID, fmt.Sprintf("email=%s provider=%s reason=%s", userInfo.Email, ssoState.Provider, errorCode))
			}
			return &SSOCallbackResult{
				CallbackURL: ssoState.CallbackURL,
				IsError:     true,
				ErrorCode:   errorCode,
				ErrorMsg:    errorMsg,
			}, nil
		}
		return &SSOCallbackResult{CallbackURL: ssoState.CallbackURL}, nil
	}

	// Login mode - resolve user and issue JWT
	return s.resolveAndIssueJWT(ctx, userInfo.Email, userInfo.FirstName, userInfo.LastName, entity.SSOProvider(config.Provider), userInfo.ProviderUserID, config, ssoState, ipAddress, userAgent)
}

// resolveAndIssueJWT implements the user resolution order from the spec.
func (s *Service) resolveAndIssueJWT(ctx context.Context, email, firstName, lastName string, provider entity.SSOProvider, providerUserID string, config *entity.SSOConfig, state *entity.SSOState, ipAddress, userAgent string) (*SSOCallbackResult, error) {
	// Domain restriction check
	if len(config.AllowedDomains) > 0 && !emailInDomains(email, config.AllowedDomains) {
		if s.auditLogger != nil {
			s.auditLogger.LogActionWithUser(ctx, "", email, "sso.login_failed", "auth", "", fmt.Sprintf("email=%s provider=%s reason=domain not allowed", email, provider))
		}
		return &SSOCallbackResult{CallbackURL: state.CallbackURL, IsError: true, ErrorCode: "domain_not_allowed"}, nil
	}

	// 3a. Lookup by provider+providerUserID
	identity, findIdentErr := s.identityRepo.FindByProviderAndProviderUserID(ctx, entity.AuthProvider(provider), providerUserID)
	if findIdentErr != nil && !errors.Is(findIdentErr, entity.ErrNotFound) {
		return nil, fmt.Errorf("resolveAndIssueJWT: finding identity: %w", findIdentErr)
	}
	if identity != nil {
		user, findErr := s.userFinder.FindByID(ctx, identity.UserID)
		if findErr != nil {
			return nil, fmt.Errorf("resolveAndIssueJWT: finding user: %w", findErr)
		}
		if !user.IsActive {
			if s.auditLogger != nil {
				s.auditLogger.LogActionWithUser(ctx, user.ID, email, "sso.login_failed", "auth", user.ID, fmt.Sprintf("email=%s provider=%s reason=account deactivated", email, provider))
			}
			return &SSOCallbackResult{CallbackURL: state.CallbackURL, IsError: true, ErrorCode: "account_deactivated"}, nil
		}
		tokenPair, tokenErr := s.authService.CreateSessionForUser(ctx, identity.UserID, string(provider), ipAddress, userAgent)
		if tokenErr != nil {
			return nil, fmt.Errorf("resolveAndIssueJWT: creating session: %w", tokenErr)
		}
		if s.auditLogger != nil {
			s.auditLogger.LogActionWithUser(ctx, user.ID, email, "sso.login_success", "auth", user.ID, fmt.Sprintf("email=%s provider=%s user_type=existing", email, provider))
		}
		return &SSOCallbackResult{CallbackURL: state.CallbackURL, TokenPair: tokenPair}, nil
	}

	// 3b. Lookup by email (case-insensitive) - REJECT if found
	user, findUserErr := s.userFinder.FindByEmail(ctx, strings.ToLower(email))
	if findUserErr != nil && !errors.Is(findUserErr, entity.ErrNotFound) {
		return nil, fmt.Errorf("resolveAndIssueJWT: finding user by email: %w", findUserErr)
	}
	if user != nil {
		return &SSOCallbackResult{
			CallbackURL: state.CallbackURL,
			IsError:     true,
			ErrorCode:   "identity_not_linked",
			ErrorMsg:    "An account with this email exists. Link your SSO identity from account settings first.",
		}, nil
	}

	// 3c. User not found - auto-create?
	if !config.AutoCreateUser {
		return &SSOCallbackResult{CallbackURL: state.CallbackURL, IsError: true, ErrorCode: "account_not_found"}, nil
	}

	// Create new user (is_superadmin=false, is_active=true, email_verified=true)
	newUser, createErr := s.userCreator.CreateSSOUser(ctx, strings.ToLower(email), firstName, lastName, provider)
	if createErr != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: creating user: %w", createErr)
	}

	newIdentity := &entity.UserIdentity{
		UserID:         newUser.ID,
		Provider:       entity.AuthProvider(provider),
		ProviderUserID: providerUserID,
		Email:          email,
	}
	if err := newIdentity.Validate(); err != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: validating identity: %w", err)
	}
	createIdentityErr := s.identityRepo.Create(ctx, newIdentity)
	if createIdentityErr != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: creating identity: %w", createIdentityErr)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogActionWithUser(ctx, newUser.ID, email, "user.sso_create", "user", newUser.ID, fmt.Sprintf("email=%s provider=%s config_id=%s", email, provider, config.ID))
	}

	tokenPair, tokenErr := s.authService.CreateSessionForUser(ctx, newUser.ID, string(provider), ipAddress, userAgent)
	if tokenErr != nil {
		return nil, fmt.Errorf("resolveAndIssueJWT: creating session: %w", tokenErr)
	}
	if s.auditLogger != nil {
		s.auditLogger.LogActionWithUser(ctx, newUser.ID, email, "sso.login_success", "auth", newUser.ID, fmt.Sprintf("email=%s provider=%s user_type=auto_created", email, provider))
	}
	return &SSOCallbackResult{CallbackURL: state.CallbackURL, TokenPair: tokenPair}, nil
}

// --- SAML Single Logout ---

// HandleSAMLSLO receives an IdP-initiated LogoutRequest and invalidates JWT sessions.
// signature and sigAlg are extracted from the request (form values or query params).
func (s *Service) HandleSAMLSLO(ctx context.Context, samlRequest, signature, sigAlg, relayState string) error {
	// Parse the SAML LogoutRequest to extract the NameID and Issuer.
	nameID, issuer, parseErr := s.samlProvider.ParseLogoutRequest(samlRequest)
	if parseErr != nil {
		return fmt.Errorf("HandleSAMLSLO: parsing logout request: %w", parseErr)
	}

	if nameID == "" {
		return fmt.Errorf("HandleSAMLSLO: no NameID in logout request")
	}

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
		return fmt.Errorf("HandleSAMLSLO: signature verification failed: %w", verifyErr)
	}

	// Find the user identity by the external NameID.
	identity, err := s.identityRepo.FindByProviderAndProviderUserID(ctx, entity.AuthProviderSAML, nameID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
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
		s.auditLogger.LogAction(ctx, "user.slo_initiate", "user", identity.UserID, fmt.Sprintf("nameID=%s", nameID))
	}

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
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("LinkIdentity: validating: %w", err)
	}
	if err := s.identityRepo.Create(ctx, identity); err != nil {
		return nil, fmt.Errorf("LinkIdentity: creating: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.identity_link", "user_identity", identity.ID, fmt.Sprintf("provider=%s", provider))
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
		s.auditLogger.LogAction(ctx, "sso.identity_unlink", "user_identity", identityID, fmt.Sprintf("provider=%s", target.Provider))
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
func (s *Service) InitiateLinkIdentity(ctx context.Context, userID, configID, callbackURL string) (string, error) {
	config, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return "", fmt.Errorf("InitiateLinkIdentity: %w", err)
	}
	if !config.IsEnabled {
		return "", ErrSSONotEnabled
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
		CallbackURL: callbackURL,
		Mode:        entity.SSOModeLink,
		ExpiresAt:   time.Now().Add(s.stateTTL),
	}

	var idpURL string
	switch config.Provider {
	case entity.SSOProviderSAML:
		var requestID string
		idpURL, requestID, err = s.samlProvider.GenerateAuthnRequest(config)
		if err != nil {
			return "", fmt.Errorf("InitiateLinkIdentity: generating SAML request: %w", err)
		}
		ssoState.SAMLRequestID = requestID
		if strings.Contains(idpURL, "?") {
			idpURL += "&RelayState=" + stateToken
		} else {
			idpURL += "?RelayState=" + stateToken
		}

	case entity.SSOProviderGoogle, entity.SSOProviderGitHub:
		var codeChallenge string
		if config.Provider == entity.SSOProviderGoogle {
			codeVerifier, cc, genErr := generatePKCE()
			if genErr != nil {
				return "", fmt.Errorf("InitiateLinkIdentity: generating PKCE: %w", genErr)
			}
			ssoState.CodeVerifier = codeVerifier
			codeChallenge = cc
		}
		idpURL = buildOAuthURL(config, stateToken, codeChallenge, s.baseURL)

	default:
		return "", ErrUnsupportedProvider
	}

	if err := ssoState.Validate(); err != nil {
		return "", fmt.Errorf("InitiateLinkIdentity: validating state: %w", err)
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
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("CreateSSOConfig: %w", err)
	}

	if err := s.configRepo.Create(ctx, config); err != nil {
		return nil, fmt.Errorf("CreateSSOConfig: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.config_create", "sso_config", config.ID, fmt.Sprintf("provider=%s display_name=%s", config.Provider, config.DisplayName))
	}

	return config, nil
}

// UpdateSSOConfig updates an SSO configuration.
func (s *Service) UpdateSSOConfig(ctx context.Context, id string, config *entity.SSOConfig) (*entity.SSOConfig, error) {
	config.ID = id
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("UpdateSSOConfig: %w", err)
	}

	if err := s.configRepo.Update(ctx, config); err != nil {
		return nil, fmt.Errorf("UpdateSSOConfig: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.config_update", "sso_config", id, fmt.Sprintf("provider=%s display_name=%s", config.Provider, config.DisplayName))
	}

	return config, nil
}

// DeleteSSOConfig removes an SSO configuration.
func (s *Service) DeleteSSOConfig(ctx context.Context, id string) error {
	// Fetch config before deletion so we can log details.
	config, err := s.configRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("DeleteSSOConfig: finding config: %w", err)
	}

	if err := s.configRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("DeleteSSOConfig: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.config_delete", "sso_config", id, fmt.Sprintf("provider=%s display_name=%s", config.Provider, config.DisplayName))
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
func (s *Service) GetSAMLEntityID(_ context.Context) string {
	return s.baseURL + "/api/auth/saml/metadata"
}

// EnsureSPSigningKey loads or generates the platform SP signing key pair.
// It loads from the settings table; if not found, generates a new RSA 2048-bit key
// and self-signed certificate (10-year validity) and stores them.
// Returns the private key and PEM-encoded certificate.
// Uses sync.Once to ensure concurrent calls only generate/load once.
func (s *Service) EnsureSPSigningKey(ctx context.Context) (*rsa.PrivateKey, string, error) {
	s.spKeyOnce.Do(func() {
		s.spKeyErr = s.loadOrGenerateSPKey(ctx)
	})
	if s.spKeyErr != nil {
		return nil, "", s.spKeyErr
	}
	return s.spKey, s.spCertPEM, nil
}

// loadOrGenerateSPKey does the actual work of loading or generating the SP key pair.
func (s *Service) loadOrGenerateSPKey(ctx context.Context) error {
	// Try to load from platform settings.
	keys := []string{"saml.sp_private_key", "saml.sp_certificate"}
	settings, err := s.settingRepo.FindPlatformSettings(ctx, keys)
	if err != nil {
		return fmt.Errorf("EnsureSPSigningKey: loading settings: %w", err)
	}

	var keyPEM, certPEM string
	for _, setting := range settings {
		switch setting.Key {
		case "saml.sp_private_key":
			keyPEM = setting.Value
		case "saml.sp_certificate":
			certPEM = setting.Value
		}
	}

	if keyPEM != "" && certPEM != "" {
		// Parse and cache.
		block, _ := pem.Decode([]byte(keyPEM))
		if block == nil {
			return fmt.Errorf("EnsureSPSigningKey: failed to decode SP private key PEM")
		}
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			// Try PKCS1 as fallback.
			rsaKey, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if pkcs1Err != nil {
				return fmt.Errorf("EnsureSPSigningKey: parsing SP private key: %w", parseErr)
			}
			parsed = rsaKey
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("EnsureSPSigningKey: SP key is not RSA")
		}
		s.spKey = rsaKey
		s.spCertPEM = certPEM
		return nil
	}

	// Generate new key pair.
	privKey, genErr := rsa.GenerateKey(rand.Reader, 2048)
	if genErr != nil {
		return fmt.Errorf("EnsureSPSigningKey: generating RSA key: %w", genErr)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Veilence-MX SAML SP"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certDER, certErr := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if certErr != nil {
		return fmt.Errorf("EnsureSPSigningKey: creating certificate: %w", certErr)
	}

	// Encode to PEM.
	keyBytes, marshalErr := x509.MarshalPKCS8PrivateKey(privKey)
	if marshalErr != nil {
		return fmt.Errorf("EnsureSPSigningKey: marshaling private key: %w", marshalErr)
	}
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}))
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))

	// Store in platform settings.
	if err := s.settingRepo.UpsertPlatformSetting(ctx, "saml.sp_private_key", keyPEM); err != nil {
		return fmt.Errorf("EnsureSPSigningKey: storing private key: %w", err)
	}
	if err := s.settingRepo.UpsertPlatformSetting(ctx, "saml.sp_certificate", certPEM); err != nil {
		return fmt.Errorf("EnsureSPSigningKey: storing certificate: %w", err)
	}

	s.spKey = privKey
	s.spCertPEM = certPEM
	return nil
}

// GetSPCertificate returns the SP signing certificate in PEM format.
// Returns empty string if no SP key has been generated yet.
func (s *Service) GetSPCertificate(ctx context.Context) (string, error) {
	_, certPEM, err := s.EnsureSPSigningKey(ctx)
	if err != nil {
		return "", fmt.Errorf("GetSPCertificate: %w", err)
	}
	return certPEM, nil
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
			return fmt.Errorf("cannot disable password login without at least one enabled SSO provider: %w", entity.ErrValidation)
		}
	}

	passwordVal := "true"
	if !config.PasswordLoginEnabled {
		passwordVal = "false"
	}

	if err := s.settingRepo.UpsertPlatformSetting(ctx, "auth.password_login_enabled", passwordVal); err != nil {
		return fmt.Errorf("UpdateAuthSettings: %w", err)
	}

	if s.auditLogger != nil {
		s.auditLogger.LogAction(ctx, "sso.auth_settings_update", "auth_settings", "", fmt.Sprintf("password_login_enabled=%s", passwordVal))
	}

	return nil
}

// --- Cleanup ---

// CleanupExpired removes expired SSOState records.
func (s *Service) CleanupExpired(ctx context.Context) error {
	_, err := s.stateRepo.DeleteExpired(ctx)
	if err != nil {
		return fmt.Errorf("CleanupExpired: deleting expired states: %w", err)
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
	b := make([]byte, pkceVerifierLength)
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
