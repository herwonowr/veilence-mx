package registry

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

const (
	goDefaultBaseURL    = "https://proxy.golang.org"
	goMaxVersionListLen = 1 * 1024 * 1024 // 1MB
	goMaxVersions       = 10000
)

var goSemverRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[a-zA-Z0-9._]+)?(\+[a-zA-Z0-9._]+)?$`)

// goTopPackagesResponse represents the JSON response from the top Go packages endpoint.
type goTopPackagesResponse struct {
	Rows []struct {
		Project   string `json:"project"`
		StarCount int64  `json:"star_count"`
	} `json:"rows"`
}

// GoModulesClient implements the Registry interface for Go modules via proxy.golang.org.
type GoModulesClient struct {
	httpClient      *http.Client
	baseURL         string
	topURL          string
	maxDownloadSize int
}

// GoModulesOption is a functional option for configuring the Go modules client.
type GoModulesOption func(*GoModulesClient)

// WithGoModulesBaseURL sets a custom base URL for testing.
func WithGoModulesBaseURL(url string) GoModulesOption {
	return func(c *GoModulesClient) { c.baseURL = url }
}

// WithGoModulesTopURL sets a custom top-packages URL for testing.
func WithGoModulesTopURL(url string) GoModulesOption {
	return func(c *GoModulesClient) { c.topURL = url }
}

// WithGoModulesMaxDownloadSize sets the maximum module archive download size in bytes.
func WithGoModulesMaxDownloadSize(size int) GoModulesOption {
	return func(c *GoModulesClient) { c.maxDownloadSize = size }
}

// NewGoModulesClient creates a new Go modules registry client.
func NewGoModulesClient(opts ...GoModulesOption) *GoModulesClient {
	c := &GoModulesClient{
		httpClient:      newSSRFSafeClient(),
		baseURL:         goDefaultBaseURL,
		topURL:          "https://raw.githubusercontent.com/herwonowr/top-go-packages/refs/heads/main/top-go-packages.min.json",
		maxDownloadSize: 200 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *GoModulesClient) Name() string { return "go" }

// GetPackage retrieves metadata for a Go module.
func (c *GoModulesClient) GetPackage(ctx context.Context, name string) (*entity.RegistryPackageInfo, error) {
	if err := validateModulePath(name); err != nil {
		return nil, fmt.Errorf("invalid module path: %w", err)
	}

	encoded := encodeModulePath(name)

	// Fetch version list
	listURL := fmt.Sprintf("%s/%s/@v/list", c.baseURL, encoded)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating version list request: %w", err)
	}
	req.Header.Set("User-Agent", "Veilence-MX")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching version list for module: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("module not found on Go proxy")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("Go proxy rate limited", "module", name)
		return nil, fmt.Errorf("go proxy rate limited")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("go proxy returned status %d for module", resp.StatusCode)
	}

	// Parse version list with limits
	limitedReader := io.LimitReader(resp.Body, goMaxVersionListLen)
	scanner := bufio.NewScanner(limitedReader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var versions []entity.RegistryVersionInfo
	versionCount := 0
	for scanner.Scan() {
		if versionCount >= goMaxVersions {
			slog.Warn("version list exceeded max count, truncating", "module", name, "max", goMaxVersions)
			break
		}
		version := strings.TrimSpace(scanner.Text())
		if version == "" {
			continue
		}
		if !goSemverRegex.MatchString(version) {
			slog.Debug("skipping non-semver version", "module", name, "version", version)
			continue
		}
		versionCount++

		// Fetch version info for publish time
		info, err := c.getVersionInfo(ctx, encoded, version)
		if err != nil {
			slog.Warn("failed to fetch version info, skipping", "module", name, "version", version, "error", err)
			continue
		}

		tarballURL := fmt.Sprintf("%s/%s/@v/%s.zip", c.baseURL, encoded, version)
		// Strip "v" prefix - all ecosystems store bare semver (e.g. "1.12.0")
		bareVersion := strings.TrimPrefix(version, "v")
		versions = append(versions, entity.RegistryVersionInfo{
			Version:     bareVersion,
			PublishedAt: info.Time,
			TarballURL:  tarballURL,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading version list: %w", err)
	}

	// Sort newest-first so poller's first-poll slice picks the most recent versions
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].PublishedAt.After(versions[j].PublishedAt)
	})

	return &entity.RegistryPackageInfo{
		Name:     name,
		Version:  latestVersion(versions),
		Versions: versions,
	}, nil
}

// goVersionInfo represents the JSON response from /@v/{version}.info.
type goVersionInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

func (c *GoModulesClient) getVersionInfo(ctx context.Context, encodedModule, version string) (*goVersionInfo, error) {
	infoURL := fmt.Sprintf("%s/%s/@v/%s.info", c.baseURL, encodedModule, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating version info request: %w", err)
	}
	req.Header.Set("User-Agent", "Veilence-MX")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching version info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version info returned status %d", resp.StatusCode)
	}

	var info goVersionInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1*1024*1024)).Decode(&info); err != nil {
		return nil, fmt.Errorf("decoding version info: %w", err)
	}
	return &info, nil
}

// GetTopPackages returns the top N Go modules with star counts.
func (c *GoModulesClient) GetTopPackages(ctx context.Context, limit int) ([]entity.PackageRanking, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.topURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating top packages request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching top Go packages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("top packages API returned status %d", resp.StatusCode)
	}

	var topResp goTopPackagesResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&topResp); err != nil {
		return nil, fmt.Errorf("decoding top packages response: %w", err)
	}

	rankings := make([]entity.PackageRanking, 0, limit)
	for _, row := range topResp.Rows {
		if len(rankings) >= limit {
			break
		}
		// Skip packages with no tagged versions (e.g. awesome-lists, non-library repos)
		if !c.hasVersions(ctx, row.Project) {
			slog.Debug("skipping package with no versions", "module", row.Project)
			continue
		}
		rankings = append(rankings, entity.PackageRanking{
			Name:          row.Project,
			DownloadCount: row.StarCount,
		})
	}

	slog.Info("fetched top Go packages", "count", len(rankings))
	return rankings, nil
}

// DownloadTarball downloads a Go module zip and returns the path to the temp file.
func (c *GoModulesClient) DownloadTarball(ctx context.Context, tarballURL string) (string, error) {
	// Validate tarball URL origin
	if !strings.HasPrefix(tarballURL, c.baseURL) {
		return "", fmt.Errorf("tarball URL does not match expected base URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "Veilence-MX")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading module archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("Go proxy rate limited during download", "url", tarballURL)
		return "", fmt.Errorf("go proxy rate limited during download")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("module archive download returned status %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "veilence-go-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, "module.zip")
	f, err := os.Create(tmpFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	limitedBody := io.LimitReader(resp.Body, int64(c.maxDownloadSize))
	written, err := io.Copy(f, limitedBody)
	f.Close()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("writing module archive: %w", err)
	}

	// Detect silent truncation - if we can read one more byte, the download was truncated
	var oneByte [1]byte
	if _, err := resp.Body.Read(oneByte[:]); err == nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("module archive exceeded maximum download size of %d bytes", c.maxDownloadSize)
	}

	slog.Info("Go module downloaded",
		"url", tarballURL,
		"size_bytes", written,
	)

	return tmpFile, nil
}

// encodeModulePath encodes a Go module path for the proxy protocol.
// Uppercase letters become '!' + lowercase per Go proxy spec.
func encodeModulePath(path string) string {
	var b strings.Builder
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if i > 0 {
			b.WriteByte('/')
		}
		// Apply !lowercase encoding
		var encoded strings.Builder
		for _, r := range segment {
			if unicode.IsUpper(r) {
				encoded.WriteByte('!')
				encoded.WriteRune(unicode.ToLower(r))
			} else {
				encoded.WriteRune(r)
			}
		}
		// URL path-escape the segment
		b.WriteString(url.PathEscape(encoded.String()))
	}
	return b.String()
}

// validateModulePath performs validation of a Go module path.
func validateModulePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty module path")
	}
	// Max length
	if len(path) > 500 {
		return fmt.Errorf("module path exceeds maximum length of 500 characters")
	}
	// Reject control characters and whitespace
	for _, r := range path {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("module path contains invalid characters")
		}
	}
	// Reject query/fragment characters
	if strings.ContainsAny(path, "?#") {
		return fmt.Errorf("module path contains invalid characters")
	}
	segments := strings.Split(path, "/")
	// First segment must look like a domain (contains a dot)
	if !strings.Contains(segments[0], ".") {
		return fmt.Errorf("module path must start with a domain")
	}
	// Reject "." and ".." segments
	for _, segment := range segments {
		if segment == ".." || segment == "." {
			return fmt.Errorf("module path contains invalid segment")
		}
	}
	return nil
}

func latestVersion(versions []entity.RegistryVersionInfo) string {
	if len(versions) == 0 {
		return ""
	}
	return versions[0].Version
}

// hasVersions does a lightweight check against the Go proxy /@v/list endpoint
// to see if a module has any tagged versions. Returns false for non-library
// repos (e.g. awesome-lists) that have no releases.
func (c *GoModulesClient) hasVersions(ctx context.Context, name string) bool {
	encoded := encodeModulePath(name)

	listURL := fmt.Sprintf("%s/%s/@v/list", c.baseURL, encoded)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Veilence-MX")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Read just the first few bytes - if there's any content, versions exist
	buf := make([]byte, 16)
	n, _ := resp.Body.Read(buf)
	return n > 0 && len(strings.TrimSpace(string(buf[:n]))) > 0
}
