package app

import (
	"context"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/pkg/oauth"
	pkgsaml "github.com/veilence/veilence-mx/backend/pkg/saml"
)

// ---------------------------------------------------------------------------
// SAML adapter - converts between pkg/saml types and internal types
// ---------------------------------------------------------------------------

// samlProviderAdapter wraps pkg/saml.Provider to implement usecase.SAMLProvider,
// converting between internal entity types and pkg-level types.
type samlProviderAdapter struct {
	provider *pkgsaml.Provider
}

func newSAMLProviderAdapter(provider *pkgsaml.Provider) *samlProviderAdapter {
	return &samlProviderAdapter{provider: provider}
}

func (a *samlProviderAdapter) GenerateAuthnRequest(config *entity.SSOConfig) (string, string, error) {
	return a.provider.GenerateAuthnRequest(toSAMLConfig(config))
}

func (a *samlProviderAdapter) ValidateResponse(config *entity.SSOConfig, samlResponse string, requestID string) (*usecase.SAMLAssertion, error) {
	result, err := a.provider.ValidateResponse(toSAMLConfig(config), samlResponse, requestID)
	if err != nil {
		return nil, err
	}
	return &usecase.SAMLAssertion{
		NameID:       result.NameID,
		Email:        result.Email,
		FirstName:    result.FirstName,
		LastName:     result.LastName,
		Groups:       result.Groups,
		SessionIndex: result.SessionIndex,
	}, nil
}

func (a *samlProviderAdapter) GenerateMetadata(config *entity.SSOConfig) ([]byte, error) {
	return a.provider.GenerateMetadata(toSAMLConfig(config))
}

func (a *samlProviderAdapter) ParseLogoutRequest(samlRequest string) (string, string, error) {
	return a.provider.ParseLogoutRequest(samlRequest)
}

func (a *samlProviderAdapter) VerifyLogoutSignature(samlRequest, signature, sigAlg, pemCertificate string) error {
	return a.provider.VerifyLogoutSignature(samlRequest, signature, sigAlg, pemCertificate)
}

func toSAMLConfig(config *entity.SSOConfig) *pkgsaml.SAMLConfig {
	return &pkgsaml.SAMLConfig{
		SAMLEntityID:      config.SAMLEntityID,
		SAMLSsoURL:        config.SAMLSsoURL,
		SAMLCertificate:   config.SAMLCertificate,
		SAMLAttrEmail:     config.SAMLAttrEmail,
		SAMLAttrFirstName: config.SAMLAttrFirstName,
		SAMLAttrLastName:  config.SAMLAttrLastName,
	}
}

// Compile-time check.
var _ usecase.SAMLProvider = (*samlProviderAdapter)(nil)

// ---------------------------------------------------------------------------
// OAuth adapter - converts between pkg/oauth types and internal types
// ---------------------------------------------------------------------------

// oauthExchangerAdapter adapts pkg/oauth.Exchanger to usecase.OAuthTokenExchanger.
// This exists because pkg/ must not import internal/ - the adapter converts types at the boundary.
type oauthExchangerAdapter struct {
	exchanger *oauth.Exchanger
}

func newOAuthExchangerAdapter(exchanger *oauth.Exchanger) usecase.OAuthTokenExchanger {
	return &oauthExchangerAdapter{exchanger: exchanger}
}

func (a *oauthExchangerAdapter) ExchangeGoogle(ctx context.Context, config *entity.SSOConfig, code, codeVerifier string) (*usecase.OAuthUserInfo, error) {
	cfg := &oauth.OAuthConfig{
		OAuthClientID:      config.OAuthClientID,
		OAuthClientSecret:  config.OAuthClientSecret,
		GoogleHostedDomain: config.GoogleHostedDomain,
	}
	info, err := a.exchanger.ExchangeGoogle(ctx, cfg, code, codeVerifier)
	if err != nil {
		return nil, err
	}
	return &usecase.OAuthUserInfo{
		ProviderUserID: info.ProviderUserID,
		Email:          info.Email,
		FirstName:      info.FirstName,
		LastName:       info.LastName,
		AvatarURL:      info.AvatarURL,
		Organizations:  info.Organizations,
		HostedDomain:   info.HostedDomain,
	}, nil
}

func (a *oauthExchangerAdapter) ExchangeGitHub(ctx context.Context, config *entity.SSOConfig, code, codeVerifier string) (*usecase.OAuthUserInfo, error) {
	cfg := &oauth.OAuthConfig{
		OAuthClientID:     config.OAuthClientID,
		OAuthClientSecret: config.OAuthClientSecret,
	}
	info, err := a.exchanger.ExchangeGitHub(ctx, cfg, code, codeVerifier)
	if err != nil {
		return nil, err
	}
	return &usecase.OAuthUserInfo{
		ProviderUserID: info.ProviderUserID,
		Email:          info.Email,
		FirstName:      info.FirstName,
		LastName:       info.LastName,
		AvatarURL:      info.AvatarURL,
		Organizations:  info.Organizations,
		HostedDomain:   info.HostedDomain,
	}, nil
}
