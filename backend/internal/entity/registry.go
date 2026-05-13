package entity

import "time"

// RegistryFileHash holds a single file hash from a registry API response.
type RegistryFileHash struct {
	Filename  string
	Algorithm string
	Hash      string
}

// RegistryVersionInfo holds metadata about a specific version of a package.
type RegistryVersionInfo struct {
	Version     string
	PublishedAt time.Time
	TarballURL  string
	Hashes      []RegistryFileHash
}

// RegistryPackageInfo holds metadata about a package from a registry.
type RegistryPackageInfo struct {
	Name        string
	Version     string
	Description string
	Versions    []RegistryVersionInfo
}
