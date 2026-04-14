package entity

import "time"

// RegistryVersionInfo holds metadata about a specific version of a package.
type RegistryVersionInfo struct {
	Version     string
	PublishedAt time.Time
	TarballURL  string
	SHA256      string
}

// RegistryPackageInfo holds metadata about a package from a registry.
type RegistryPackageInfo struct {
	Name        string
	Version     string
	Description string
	Versions    []RegistryVersionInfo
}
