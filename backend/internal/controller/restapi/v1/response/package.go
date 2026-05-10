// Package response defines HTTP response DTOs for the v1 REST API.
// These DTOs own the JSON tags; entity types should remain tag-free.
package response

import (
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// PackageResponse is the JSON representation of a monitored package.
type PackageResponse struct {
	ID                     string     `json:"id"`
	WorkspaceID            string     `json:"workspaceId"`
	Name                   string     `json:"name"`
	Ecosystem              string     `json:"ecosystem"`
	LatestVersion          string     `json:"latestVersion"`
	Description            string     `json:"description"`
	Source                 string     `json:"source"`
	Status                 string     `json:"status"`
	DownloadCount          int64      `json:"downloadCount"`
	DownloadCountUpdatedAt *time.Time `json:"downloadCountUpdatedAt,omitempty"`
	BlockedAt              *time.Time `json:"blockedAt,omitempty"`
	BlockedReason          string     `json:"blockedReason,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

// PackageFromEntity maps a domain Package to a response DTO.
func PackageFromEntity(p *entity.Package) PackageResponse {
	return PackageResponse{
		ID:                     p.ID,
		WorkspaceID:            p.WorkspaceID,
		Name:                   p.Name,
		Ecosystem:              string(p.Ecosystem),
		LatestVersion:          p.LatestVersion,
		Description:            p.Description,
		Source:                 string(p.Source),
		Status:                 string(p.Status),
		DownloadCount:          p.DownloadCount,
		DownloadCountUpdatedAt: p.DownloadCountUpdatedAt,
		BlockedAt:              p.BlockedAt,
		BlockedReason:          p.BlockedReason,
		CreatedAt:              p.CreatedAt,
		UpdatedAt:              p.UpdatedAt,
	}
}

// PackagesFromEntities maps a slice of domain Packages to response DTOs.
func PackagesFromEntities(ps []entity.Package) []PackageResponse {
	result := make([]PackageResponse, len(ps))
	for i := range ps {
		result[i] = PackageFromEntity(&ps[i])
	}
	return result
}

// StalePackageResponse extends PackageResponse with stale-specific fields.
type StalePackageResponse struct {
	PackageResponse
	LastReleaseAt        *time.Time `json:"lastReleaseAt"`
	DaysSinceLastRelease *int       `json:"daysSinceLastRelease"`
}

// StalePackageFromEntity maps a domain Package to a stale package response DTO.
func StalePackageFromEntity(p *entity.Package) StalePackageResponse {
	resp := StalePackageResponse{
		PackageResponse: PackageFromEntity(p),
	}
	if p.LastReleaseAt != nil {
		resp.LastReleaseAt = p.LastReleaseAt
		days := int(time.Since(*p.LastReleaseAt).Hours() / 24)
		resp.DaysSinceLastRelease = &days
	}
	return resp
}

// StalePackagesFromEntities maps a slice of domain Packages to stale package response DTOs.
func StalePackagesFromEntities(ps []entity.Package) []StalePackageResponse {
	result := make([]StalePackageResponse, len(ps))
	for i := range ps {
		result[i] = StalePackageFromEntity(&ps[i])
	}
	return result
}

// ImportErrorResponse is the JSON representation of a single import error.
type ImportErrorResponse struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// ImportResultResponse summarizes the outcome of a bulk import.
type ImportResultResponse struct {
	Imported int                   `json:"imported"`
	Skipped  int                   `json:"skipped"`
	Errors   []ImportErrorResponse `json:"errors,omitempty"`
}

// ImportResultFromEntity maps an entity ImportResult to a response DTO.
func ImportResultFromEntity(r *entity.ImportResult) ImportResultResponse {
	resp := ImportResultResponse{
		Imported: r.Imported,
		Skipped:  r.Skipped,
	}
	for _, e := range r.Errors {
		resp.Errors = append(resp.Errors, ImportErrorResponse{
			Name:  e.Name,
			Error: e.Error,
		})
	}
	return resp
}
