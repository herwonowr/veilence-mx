package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

type pypiPackageResponse struct {
	Info struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Summary string `json:"summary"`
	} `json:"info"`
	Releases map[string][]pypiReleaseFile `json:"releases"`
}

type pypiReleaseFile struct {
	URL         string `json:"url"`
	PackageType string `json:"packagetype"`
	UploadTime  string `json:"upload_time_iso_8601"`
}

type pypiTopPackagesResponse struct {
	Rows []struct {
		Project       string `json:"project"`
		DownloadCount int64  `json:"download_count"`
	} `json:"rows"`
}

// PyPIClient implements the Registry interface for PyPI.
type PyPIClient struct {
	httpClient      *http.Client
	baseURL         string
	topURL          string
	tarballBaseURL  string
	maxDownloadSize int
}

// PyPIOption is a functional option for configuring the PyPI client.
type PyPIOption func(*PyPIClient)

// WithPyPIBaseURL sets a custom base URL for testing.
func WithPyPIBaseURL(url string) PyPIOption {
	return func(c *PyPIClient) { c.baseURL = url }
}

// WithPyPITopURL sets a custom top-packages URL for testing.
func WithPyPITopURL(url string) PyPIOption {
	return func(c *PyPIClient) { c.topURL = url }
}

// WithPyPITarballBaseURL sets a custom tarball base URL for testing.
func WithPyPITarballBaseURL(url string) PyPIOption {
	return func(c *PyPIClient) { c.tarballBaseURL = url }
}

// WithPyPIMaxDownloadSize sets the maximum tarball download size in bytes.
func WithPyPIMaxDownloadSize(size int) PyPIOption {
	return func(c *PyPIClient) { c.maxDownloadSize = size }
}

// NewPyPIClient creates a new PyPI registry client.
func NewPyPIClient(opts ...PyPIOption) *PyPIClient {
	c := &PyPIClient{
		httpClient:      newSSRFSafeClient(),
		baseURL:         "https://pypi.org",
		topURL:          "https://hugovk.github.io/top-pypi-packages/top-pypi-packages-30-days.min.json",
		tarballBaseURL:  "https://files.pythonhosted.org/",
		maxDownloadSize: 200 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *PyPIClient) Name() string { return "python" }

// GetPackage retrieves metadata for a PyPI package.
func (c *PyPIClient) GetPackage(ctx context.Context, name string) (*entity.RegistryPackageInfo, error) {
	url := fmt.Sprintf("%s/pypi/%s/json", c.baseURL, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", name, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Veilence-MX")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching package %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %s not found on PyPI", name)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("PyPI rate limited", "package", name)
		return nil, fmt.Errorf("PyPI rate limited for package %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PyPI returned status %d for package %s", resp.StatusCode, name)
	}

	var pypiResp pypiPackageResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&pypiResp); err != nil {
		return nil, fmt.Errorf("decoding response for %s: %w", name, err)
	}

	versions := make([]entity.RegistryVersionInfo, 0, len(pypiResp.Releases))
	for version, files := range pypiResp.Releases {
		for _, f := range files {
			if f.PackageType == "sdist" {
				publishedAt := time.Time{}
				if f.UploadTime != "" {
					if t, parseErr := time.Parse(time.RFC3339, f.UploadTime); parseErr == nil {
						publishedAt = t
					}
				}
				versions = append(versions, entity.RegistryVersionInfo{
					Version:     version,
					PublishedAt: publishedAt,
					TarballURL:  f.URL,
				})
				break
			}
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].PublishedAt.After(versions[j].PublishedAt)
	})

	return &entity.RegistryPackageInfo{
		Name:        pypiResp.Info.Name,
		Version:     pypiResp.Info.Version,
		Description: pypiResp.Info.Summary,
		Versions:    versions,
	}, nil
}

// GetTopPackages returns the top N PyPI packages with download counts.
func (c *PyPIClient) GetTopPackages(ctx context.Context, limit int) ([]entity.PackageRanking, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.topURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating top packages request: %w", err)
	}
	req.Header.Set("User-Agent", "Veilence-MX")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching top PyPI packages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("top packages API returned status %d", resp.StatusCode)
	}

	var topResp pypiTopPackagesResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&topResp); err != nil {
		return nil, fmt.Errorf("decoding top packages response: %w", err)
	}

	rankings := make([]entity.PackageRanking, 0, limit)
	for i, row := range topResp.Rows {
		if i >= limit {
			break
		}
		rankings = append(rankings, entity.PackageRanking{
			Name:          row.Project,
			DownloadCount: row.DownloadCount,
		})
	}

	slog.Info("fetched top PyPI packages", "count", len(rankings))
	return rankings, nil
}

// DownloadTarball downloads a tarball and returns the path to the temp file.
func (c *PyPIClient) DownloadTarball(ctx context.Context, tarballURL string) (string, error) {
	if err := validateTarballURL(tarballURL, c.tarballBaseURL); err != nil {
		return "", fmt.Errorf("validating tarball URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating tarball request: %w", err)
	}
	req.Header.Set("User-Agent", "Veilence-MX")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading tarball from %s: %w", tarballURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("PyPI rate limited during tarball download")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tarball download returned status %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "veilence-pypi-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, "package.tar.gz")
	f, err := os.Create(tmpFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, io.LimitReader(resp.Body, int64(c.maxDownloadSize)))
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("writing tarball: %w", err)
	}

	// Detect silent truncation
	var oneByte [1]byte
	if _, err := resp.Body.Read(oneByte[:]); err == nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("tarball exceeded maximum download size of %d bytes", c.maxDownloadSize)
	}

	_ = written
	return tmpFile, nil
}
