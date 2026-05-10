package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

type npmPackageResponse struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	DistTags    map[string]string            `json:"dist-tags"`
	Time        map[string]string            `json:"time"`
	Versions    map[string]npmVersionDetails `json:"versions"`
}

type npmVersionDetails struct {
	Version string `json:"version"`
	Dist    struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

type npmSearchResponse struct {
	Objects []struct {
		Package struct {
			Name string `json:"name"`
		} `json:"package"`
		Downloads struct {
			Monthly int64 `json:"monthly"`
			Weekly  int64 `json:"weekly"`
		} `json:"downloads"`
		Score struct {
			Detail struct {
				Popularity float64 `json:"popularity"`
			} `json:"detail"`
		} `json:"score"`
	} `json:"objects"`
	Total int `json:"total"`
}

// NPMClient implements the Registry interface for npm.
type NPMClient struct {
	httpClient *http.Client
	baseURL    string
}

// NPMOption is a functional option for configuring the npm client.
type NPMOption func(*NPMClient)

// WithNPMBaseURL sets a custom base URL for testing.
func WithNPMBaseURL(baseURL string) NPMOption {
	return func(c *NPMClient) { c.baseURL = baseURL }
}

// NewNPMClient creates a new npm registry client.
func NewNPMClient(opts ...NPMOption) *NPMClient {
	c := &NPMClient{
		httpClient: newSSRFSafeClient(),
		baseURL:    "https://registry.npmjs.org",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *NPMClient) Name() string { return "npm" }

func encodeScopedPackage(name string) string {
	if after, ok := strings.CutPrefix(name, "@"); ok {
		return "@" + url.PathEscape(after)
	}
	return url.PathEscape(name)
}

// GetPackage retrieves metadata for an npm package.
func (c *NPMClient) GetPackage(ctx context.Context, name string) (*entity.RegistryPackageInfo, error) {
	encodedName := encodeScopedPackage(name)
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, encodedName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
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
		return nil, fmt.Errorf("package %s not found on npm", name)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("npm rate limited", "package", name)
		return nil, fmt.Errorf("npm rate limited for package %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm returned status %d for package %s", resp.StatusCode, name)
	}

	var npmResp npmPackageResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&npmResp); err != nil {
		return nil, fmt.Errorf("decoding response for %s: %w", name, err)
	}

	latestVersion := ""
	if latest, ok := npmResp.DistTags["latest"]; ok {
		latestVersion = latest
	}

	versions := make([]entity.RegistryVersionInfo, 0, len(npmResp.Versions))
	for version, details := range npmResp.Versions {
		publishedAt := time.Time{}
		if timeStr, ok := npmResp.Time[version]; ok {
			if t, parseErr := time.Parse(time.RFC3339, timeStr); parseErr == nil {
				publishedAt = t
			}
		}
		versions = append(versions, entity.RegistryVersionInfo{
			Version:     version,
			PublishedAt: publishedAt,
			TarballURL:  details.Dist.Tarball,
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].PublishedAt.After(versions[j].PublishedAt)
	})

	return &entity.RegistryPackageInfo{
		Name:        npmResp.Name,
		Version:     latestVersion,
		Description: npmResp.Description,
		Versions:    versions,
	}, nil
}

// GetTopPackages returns the top N npm packages by popularity with scores.
func (c *NPMClient) GetTopPackages(ctx context.Context, limit int) ([]entity.PackageRanking, error) {
	var allRankings []entity.PackageRanking
	fetched := 0
	batchSize := 250
	if limit < batchSize {
		batchSize = limit
	}

	for fetched < limit {
		remaining := limit - fetched
		size := batchSize
		if remaining < size {
			size = remaining
		}

		reqURL := fmt.Sprintf("%s/-/v1/search?text=keywords:&popularity=1.0&quality=0.0&maintenance=0.0&size=%d&from=%d",
			c.baseURL, size, fetched)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating search request: %w", err)
		}
		req.Header.Set("User-Agent", "Veilence-MX")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching top npm packages: %w", err)
		}

		var searchResp npmSearchResponse
		if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&searchResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding search response: %w", err)
		}
		resp.Body.Close()

		for _, obj := range searchResp.Objects {
			allRankings = append(allRankings, entity.PackageRanking{
				Name:          obj.Package.Name,
				DownloadCount: obj.Downloads.Monthly,
			})
		}

		fetched += len(searchResp.Objects)
		if len(searchResp.Objects) < size {
			break
		}
	}

	// Sort by download count descending
	sort.Slice(allRankings, func(i, j int) bool {
		return allRankings[i].DownloadCount > allRankings[j].DownloadCount
	})

	slog.Info("fetched top npm packages", "count", len(allRankings))
	return allRankings, nil
}

// DownloadTarball downloads a tarball and returns the path to the temp file.
func (c *NPMClient) DownloadTarball(ctx context.Context, tarballURL string) (string, error) {
	if err := validateTarballURL(tarballURL, c.baseURL); err != nil {
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
		return "", fmt.Errorf("npm rate limited during tarball download")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tarball download returned status %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "veilence-npm-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, "package.tgz")
	f, err := os.Create(tmpFile)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	const maxDownloadSize = 200 * 1024 * 1024 // 200MB
	written, err := io.Copy(f, io.LimitReader(resp.Body, maxDownloadSize))
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("writing tarball: %w", err)
	}

	// Detect silent truncation
	var oneByte [1]byte
	if _, err := resp.Body.Read(oneByte[:]); err == nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("tarball exceeded maximum download size of %d bytes", maxDownloadSize)
	}

	_ = written
	return tmpFile, nil
}
