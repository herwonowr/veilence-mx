// Package saml provides a SAML 2.0 Service Provider implementation.
package saml

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// Provider implements usecase.SAMLProvider using the crewjam/saml library.
type Provider struct {
	// baseURL is the application's base URL used for ACS and metadata endpoints.
	baseURL string
	// clockSkew is the maximum allowed clock skew when validating SAML assertions.
	clockSkew time.Duration
}

// NewProvider creates a new SAML provider.
// baseURL is the application base URL used for ACS and metadata endpoints.
// clockSkew is the maximum allowed clock difference with the IdP.
func NewProvider(baseURL string, clockSkew time.Duration) *Provider {
	return &Provider{baseURL: baseURL, clockSkew: clockSkew}
}

// GenerateAuthnRequest creates a SAML AuthnRequest and returns the IdP redirect URL.
func (p *Provider) GenerateAuthnRequest(config *entity.SSOConfig) (string, error) {
	sp, err := p.buildServiceProvider(config)
	if err != nil {
		return "", fmt.Errorf("saml.GenerateAuthnRequest: %w", err)
	}

	redirectURL, err := sp.MakeRedirectAuthenticationRequest("")
	if err != nil {
		return "", fmt.Errorf("saml.GenerateAuthnRequest: creating redirect URL: %w", err)
	}

	return redirectURL.String(), nil
}

// ValidateResponse validates a SAML response and extracts the assertion data.
// samlResponse is the base64-encoded SAMLResponse from the IdP POST to our ACS.
func (p *Provider) ValidateResponse(config *entity.SSOConfig, samlResponse string) (*usecase.SAMLAssertion, error) {
	sp, err := p.buildServiceProvider(config)
	if err != nil {
		return nil, fmt.Errorf("saml.ValidateResponse: %w", err)
	}

	// Build a fake http.Request with the SAMLResponse form value,
	// since crewjam/saml.ParseResponse expects *http.Request.
	form := url.Values{}
	form.Set("SAMLResponse", samlResponse)
	body := form.Encode()
	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/api/auth/saml/acs", bytes.NewBufferString(body))
	if err != nil {
		return nil, fmt.Errorf("saml.ValidateResponse: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if parseErr := req.ParseForm(); parseErr != nil {
		return nil, fmt.Errorf("saml.ValidateResponse: parsing form: %w", parseErr)
	}

	assertion, err := sp.ParseResponse(req, []string{""})
	if err != nil {
		return nil, fmt.Errorf("saml.ValidateResponse: validating assertion: %w", err)
	}

	if assertion == nil {
		return nil, errors.New("saml.ValidateResponse: no assertion in response")
	}

	result := &usecase.SAMLAssertion{
		NameID: assertion.Subject.NameID.Value,
	}

	// Extract attributes using configured attribute names.
	attrEmail := config.SAMLAttrEmail
	attrFirstName := config.SAMLAttrFirstName
	attrLastName := config.SAMLAttrLastName

	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if len(attr.Values) == 0 {
				continue
			}
			switch attr.Name {
			case attrEmail:
				result.Email = attr.Values[0].Value
			case attrFirstName:
				result.FirstName = attr.Values[0].Value
			case attrLastName:
				result.LastName = attr.Values[0].Value
			case "groups", "memberOf":
				for _, v := range attr.Values {
					result.Groups = append(result.Groups, v.Value)
				}
			}
		}
	}

	// Extract session index.
	for _, stmt := range assertion.AuthnStatements {
		if stmt.SessionIndex != "" {
			result.SessionIndex = stmt.SessionIndex
			break
		}
	}

	return result, nil
}

// GenerateMetadata generates SP metadata XML for this configuration.
func (p *Provider) GenerateMetadata(config *entity.SSOConfig) ([]byte, error) {
	sp, err := p.buildServiceProvider(config)
	if err != nil {
		return nil, fmt.Errorf("saml.GenerateMetadata: %w", err)
	}

	metadata := sp.Metadata()
	buf, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("saml.GenerateMetadata: marshaling XML: %w", err)
	}

	return buf, nil
}

// buildServiceProvider constructs a crewjam/saml.ServiceProvider from the entity config.
func (p *Provider) buildServiceProvider(config *entity.SSOConfig) (saml.ServiceProvider, error) {
	cert, err := parsePEMCertificate(config.SAMLCertificate)
	if err != nil {
		return saml.ServiceProvider{}, fmt.Errorf("parsing IdP certificate: %w", err)
	}

	acsURL, err := url.Parse(p.baseURL + "/api/auth/saml/acs")
	if err != nil {
		return saml.ServiceProvider{}, fmt.Errorf("parsing ACS URL: %w", err)
	}

	metadataPath := "/api/auth/saml/" + config.ID + "/metadata"
	metadataURL, err := url.Parse(p.baseURL + metadataPath)
	if err != nil {
		return saml.ServiceProvider{}, fmt.Errorf("parsing metadata URL: %w", err)
	}

	idpDescriptor := &saml.EntityDescriptor{
		EntityID: config.SAMLEntityID,
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						KeyDescriptors: []saml.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: saml.KeyInfo{
									X509Data: saml.X509Data{
										X509Certificates: []saml.X509Certificate{
											{Data: base64.StdEncoding.EncodeToString(cert.Raw)},
										},
									},
								},
							},
						},
					},
				},
				SingleSignOnServices: []saml.Endpoint{
					{
						Binding:  saml.HTTPRedirectBinding,
						Location: config.SAMLSsoURL,
					},
					{
						Binding:  saml.HTTPPostBinding,
						Location: config.SAMLSsoURL,
					},
				},
			},
		},
	}

	sp := saml.ServiceProvider{
		EntityID:          p.baseURL + metadataPath,
		AcsURL:            *acsURL,
		MetadataURL:       *metadataURL,
		IDPMetadata:       idpDescriptor,
		AuthnNameIDFormat: saml.EmailAddressNameIDFormat,
	}

	// Apply clock skew tolerance if configured.
	if p.clockSkew > 0 {
		saml.MaxClockSkew = p.clockSkew
	}

	return sp, nil
}

// parsePEMCertificate parses a PEM-encoded X.509 certificate.
func parsePEMCertificate(pemData string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("no PEM block found in certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing X.509 certificate: %w", err)
	}
	return cert, nil
}

// ParseLogoutRequest parses an IdP-initiated SAML LogoutRequest and extracts the NameID and Issuer.
func (p *Provider) ParseLogoutRequest(samlRequest string) (string, string, error) {
	raw, err := base64.StdEncoding.DecodeString(samlRequest)
	if err != nil {
		return "", "", fmt.Errorf("saml.ParseLogoutRequest: decoding base64: %w", err)
	}

	var logoutReq struct {
		XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol LogoutRequest"`
		Issuer  struct {
			Value string `xml:",chardata"`
		} `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
		NameID struct {
			Value string `xml:",chardata"`
		} `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`
	}
	if err := xml.Unmarshal(raw, &logoutReq); err != nil {
		return "", "", fmt.Errorf("saml.ParseLogoutRequest: parsing XML: %w", err)
	}

	if logoutReq.NameID.Value == "" {
		return "", "", errors.New("saml.ParseLogoutRequest: no NameID found in LogoutRequest")
	}

	return logoutReq.NameID.Value, logoutReq.Issuer.Value, nil
}

// VerifyLogoutSignature verifies the XML signature on a SAML LogoutRequest.
// samlRequest is the base64-encoded LogoutRequest, signature is the base64-encoded
// signature value, sigAlg is the signature algorithm URI, and pemCertificate is the
// PEM-encoded IdP signing certificate.
func (p *Provider) VerifyLogoutSignature(samlRequest, signature, sigAlg, pemCertificate string) error {
	if signature == "" {
		return errors.New("saml.VerifyLogoutSignature: no signature provided")
	}
	if sigAlg == "" {
		return errors.New("saml.VerifyLogoutSignature: no signature algorithm provided")
	}

	cert, err := parsePEMCertificate(pemCertificate)
	if err != nil {
		return fmt.Errorf("saml.VerifyLogoutSignature: %w", err)
	}

	rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("saml.VerifyLogoutSignature: certificate does not contain an RSA public key")
	}

	// Determine the hash algorithm from the signature algorithm URI.
	var hashAlg crypto.Hash
	switch sigAlg {
	case "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256":
		hashAlg = crypto.SHA256
	case "http://www.w3.org/2000/09/xmldsig#rsa-sha1":
		hashAlg = crypto.SHA1
	case "http://www.w3.org/2001/04/xmldsig-more#rsa-sha384":
		hashAlg = crypto.SHA384
	case "http://www.w3.org/2001/04/xmldsig-more#rsa-sha512":
		hashAlg = crypto.SHA512
	default:
		return fmt.Errorf("saml.VerifyLogoutSignature: unsupported signature algorithm: %s", sigAlg)
	}

	// For HTTP-Redirect binding, the signature is computed over the query string
	// components: SAMLRequest=...&SigAlg=...
	// We reconstruct the signed content.
	signedContent := "SAMLRequest=" + url.QueryEscape(samlRequest) + "&SigAlg=" + url.QueryEscape(sigAlg)

	h := hashAlg.New()
	h.Write([]byte(signedContent))
	digest := h.Sum(nil)

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("saml.VerifyLogoutSignature: decoding signature: %w", err)
	}

	if err := rsa.VerifyPKCS1v15(rsaKey, hashAlg, digest, sigBytes); err != nil {
		return fmt.Errorf("saml.VerifyLogoutSignature: signature verification failed: %w", err)
	}

	return nil
}

// Compile-time check that Provider implements usecase.SAMLProvider.
var _ usecase.SAMLProvider = (*Provider)(nil)
