package registry

import (
	"context"
	"time"
)

// VersionInfo holds metadata about a specific version of a package.
type VersionInfo struct {
	Version     string
	PublishedAt time.Time
	TarballURL  string
	SHA256      string
}

// PackageInfo holds metadata about a package from a registry.
type PackageInfo struct {
	Name        string
	Version     string
	Description string
	Versions    []VersionInfo
}

// Registry defines the interface for interacting with a package registry.
type Registry interface {
	GetPackage(ctx context.Context, name string) (*PackageInfo, error)
	GetTopPackages(ctx context.Context, limit int) ([]string, error)
	DownloadTarball(ctx context.Context, url string) (string, error)
	Name() string
}
