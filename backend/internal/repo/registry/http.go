package registry

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// newSSRFSafeClient creates an HTTP client with SSRF protections:
// - Rejects redirects to private/internal IPs
// - Rejects HTTPS to HTTP downgrades
// - Limits redirect count to 10
func newSSRFSafeClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme == "http" && len(via) > 0 && via[len(via)-1].URL.Scheme == "https" {
				return fmt.Errorf("redirect from HTTPS to HTTP is not allowed")
			}
			host := req.URL.Hostname()
			if isPrivateHost(host) {
				return fmt.Errorf("redirect to private/internal IP is not allowed")
			}
			return nil
		},
	}
}

// isPrivateHost checks if a hostname resolves to a private/internal IP address.
func isPrivateHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		addrs, err := net.LookupHost(host)
		if err != nil || len(addrs) == 0 {
			return false
		}
		ip = net.ParseIP(addrs[0])
		if ip == nil {
			return false
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// validateTarballURL checks that a tarball URL starts with the expected base URL.
func validateTarballURL(tarballURL, baseURL string) error {
	if !strings.HasPrefix(tarballURL, baseURL) {
		return fmt.Errorf("tarball URL does not match expected registry origin")
	}
	return nil
}
