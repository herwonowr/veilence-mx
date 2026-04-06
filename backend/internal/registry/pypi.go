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
	Digests     struct {
		SHA256 string `json:"sha256"`
	} `json:"digests"`
	UploadTime string `json:"upload_time_iso_8601"`
}

type pypiTopPackagesResponse struct {
	Rows []struct {
		Project string `json:"project"`
	} `json:"rows"`
}

// PyPIClient implements the Registry interface for PyPI.
type PyPIClient struct {
	httpClient *http.Client
	baseURL    string
	topURL     string
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

// NewPyPIClient creates a new PyPI registry client.
func NewPyPIClient(opts ...PyPIOption) *PyPIClient {
	c := &PyPIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://pypi.org",
		topURL:     "https://hugovk.github.io/top-pypi-packages/top-pypi-packages-30-days.min.json",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *PyPIClient) Name() string { return "pypi" }

// GetPackage retrieves metadata for a PyPI package.
func (c *PyPIClient) GetPackage(ctx context.Context, name string) (*PackageInfo, error) {
	url := fmt.Sprintf("%s/pypi/%s/json", c.baseURL, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", name, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching package %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %s not found on PyPI", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PyPI returned status %d for package %s", resp.StatusCode, name)
	}

	var pypiResp pypiPackageResponse
	if err := json.NewDecoder(resp.Body).Decode(&pypiResp); err != nil {
		return nil, fmt.Errorf("decoding response for %s: %w", name, err)
	}

	versions := make([]VersionInfo, 0, len(pypiResp.Releases))
	for version, files := range pypiResp.Releases {
		for _, f := range files {
			if f.PackageType == "sdist" {
				publishedAt := time.Time{}
				if f.UploadTime != "" {
					if t, parseErr := time.Parse(time.RFC3339, f.UploadTime); parseErr == nil {
						publishedAt = t
					}
				}
				versions = append(versions, VersionInfo{
					Version:     version,
					PublishedAt: publishedAt,
					TarballURL:  f.URL,
					SHA256:      f.Digests.SHA256,
				})
				break
			}
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].PublishedAt.After(versions[j].PublishedAt)
	})

	return &PackageInfo{
		Name:        pypiResp.Info.Name,
		Version:     pypiResp.Info.Version,
		Description: pypiResp.Info.Summary,
		Versions:    versions,
	}, nil
}

// GetTopPackages returns the names of the top N PyPI packages.
func (c *PyPIClient) GetTopPackages(ctx context.Context, limit int) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.topURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating top packages request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching top PyPI packages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("top packages API returned status %d", resp.StatusCode)
	}

	var topResp pypiTopPackagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&topResp); err != nil {
		return nil, fmt.Errorf("decoding top packages response: %w", err)
	}

	names := make([]string, 0, limit)
	for i, row := range topResp.Rows {
		if i >= limit {
			break
		}
		names = append(names, row.Project)
	}

	slog.Info("fetched top PyPI packages", "count", len(names))
	return names, nil
}

// DownloadTarball downloads a tarball and returns the path to the temp file.
func (c *PyPIClient) DownloadTarball(ctx context.Context, tarballURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating tarball request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading tarball from %s: %w", tarballURL, err)
	}
	defer resp.Body.Close()

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
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("writing tarball: %w", err)
	}

	return tmpFile, nil
}
