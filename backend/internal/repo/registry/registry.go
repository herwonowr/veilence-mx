package registry

import (
	"context"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// Registry defines the interface for interacting with a package registry.
// NOTE: The canonical interface lives in usecase/contracts.go (consumer defines the contract).
// This type alias is kept here for backward compatibility and to satisfy the interface locally.
type Registry interface {
	GetPackage(ctx context.Context, name string) (*entity.RegistryPackageInfo, error)
	GetTopPackages(ctx context.Context, limit int) ([]entity.PackageRanking, error)
	DownloadTarball(ctx context.Context, url string) (string, error)
	Name() string
}
