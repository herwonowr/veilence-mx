package saml

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

const (
	metadataFetchTimeout = 10 * time.Second
	metadataMaxRedirects = 3
	metadataMaxBodySize  = 5 * 1024 * 1024 // 5 MB
)

// MetadataFetcher fetches and parses SAML IdP metadata from a URL.
type MetadataFetcher struct{}

// NewMetadataFetcher creates a new MetadataFetcher.
func NewMetadataFetcher() *MetadataFetcher {
	return &MetadataFetcher{}
}

// FetchAndParse fetches SAML metadata XML from the given URL and extracts
// the entity ID, SSO URL, and signing certificate.
func (f *MetadataFetcher) FetchAndParse(ctx context.Context, metadataURL string) (*usecase.SAMLMetadataInfo, error) {
	// Validate URL.
	parsed, err := url.Parse(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("metadata URL must use http or https scheme")
	}

	// Create HTTP client with timeout and redirect limit.
	redirectCount := 0
	client := &http.Client{
		Timeout: metadataFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount++
			if redirectCount > metadataMaxRedirects {
				return fmt.Errorf("too many redirects (max %d)", metadataMaxRedirects)
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/xml, text/xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata URL returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, metadataMaxBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading metadata response: %w", err)
	}

	return parseMetadataXML(body)
}

// samlEntityDescriptor is a minimal representation for parsing SAML metadata XML.
type samlEntityDescriptor struct {
	XMLName          xml.Name         `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	EntityID         string           `xml:"entityID,attr"`
	IDPSSODescriptor []samlIDPSSODesc `xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
}

type samlIDPSSODesc struct {
	KeyDescriptors       []samlKeyDescriptor `xml:"urn:oasis:names:tc:SAML:2.0:metadata KeyDescriptor"`
	SingleSignOnServices []samlSSOS          `xml:"urn:oasis:names:tc:SAML:2.0:metadata SingleSignOnService"`
	SingleLogoutServices []samlSSOS          `xml:"urn:oasis:names:tc:SAML:2.0:metadata SingleLogoutService"`
}

type samlKeyDescriptor struct {
	Use     string      `xml:"use,attr"`
	KeyInfo samlKeyInfo `xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
}

type samlKeyInfo struct {
	X509Data samlX509Data `xml:"http://www.w3.org/2000/09/xmldsig# X509Data"`
}

type samlX509Data struct {
	Certificate string `xml:"http://www.w3.org/2000/09/xmldsig# X509Certificate"`
}

type samlSSOS struct {
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
}

func parseMetadataXML(data []byte) (*usecase.SAMLMetadataInfo, error) {
	var desc samlEntityDescriptor
	if err := xml.Unmarshal(data, &desc); err != nil {
		return nil, fmt.Errorf("parsing metadata XML: %w", err)
	}

	if desc.EntityID == "" {
		return nil, fmt.Errorf("no entityID found in metadata")
	}

	if len(desc.IDPSSODescriptor) == 0 {
		return nil, fmt.Errorf("no IDPSSODescriptor found in metadata")
	}

	idp := desc.IDPSSODescriptor[0]

	// Extract SSO URL - prefer HTTP-POST, fall back to HTTP-Redirect.
	var ssoURL string
	for _, svc := range idp.SingleSignOnServices {
		if svc.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" {
			ssoURL = svc.Location
			break
		}
	}
	if ssoURL == "" {
		for _, svc := range idp.SingleSignOnServices {
			if svc.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
				ssoURL = svc.Location
				break
			}
		}
	}
	if ssoURL == "" {
		return nil, fmt.Errorf("no SingleSignOnService with HTTP-POST or HTTP-Redirect binding found")
	}

	// Extract signing certificate.
	var certData string
	for _, kd := range idp.KeyDescriptors {
		if kd.Use == "signing" || kd.Use == "" {
			raw := strings.TrimSpace(kd.KeyInfo.X509Data.Certificate)
			if raw != "" {
				certData = raw
				break
			}
		}
	}
	if certData == "" {
		return nil, fmt.Errorf("no X509Certificate found in signing key descriptor")
	}

	// Format as PEM.
	pemCert := formatAsPEM(certData)

	// Extract SLO URL (optional) - prefer HTTP-POST, fall back to HTTP-Redirect.
	var sloURL string
	for _, svc := range idp.SingleLogoutServices {
		if svc.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" {
			sloURL = svc.Location
			break
		}
	}
	if sloURL == "" {
		for _, svc := range idp.SingleLogoutServices {
			if svc.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
				sloURL = svc.Location
				break
			}
		}
	}

	return &usecase.SAMLMetadataInfo{
		EntityID:    desc.EntityID,
		SSOURL:      ssoURL,
		SloURL:      sloURL,
		Certificate: pemCert,
	}, nil
}

// formatAsPEM wraps a base64-encoded certificate in PEM headers.
func formatAsPEM(base64Cert string) string {
	// Remove any existing whitespace/newlines from the base64 data.
	clean := strings.Join(strings.Fields(base64Cert), "")

	var b strings.Builder
	b.WriteString("-----BEGIN CERTIFICATE-----\n")
	for i := 0; i < len(clean); i += 64 {
		end := i + 64
		if end > len(clean) {
			end = len(clean)
		}
		b.WriteString(clean[i:end])
		b.WriteString("\n")
	}
	b.WriteString("-----END CERTIFICATE-----")
	return b.String()
}

// Compile-time check that MetadataFetcher implements usecase.SAMLMetadataFetcher.
var _ usecase.SAMLMetadataFetcher = (*MetadataFetcher)(nil)
