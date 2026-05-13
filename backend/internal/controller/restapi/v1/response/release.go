package response

import (
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// ReleaseResponse is the JSON representation of a package release.
type ReleaseResponse struct {
	ID           string    `json:"id"`
	PackageID    string    `json:"packageId"`
	Version      string    `json:"version"`
	PublishedAt  time.Time `json:"publishedAt"`
	TarballURL   string    `json:"tarballUrl"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	HashCount    int64     `json:"hashCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ReleaseFromEntity maps a domain Release to a response DTO.
func ReleaseFromEntity(r *entity.Release) ReleaseResponse {
	return ReleaseResponse{
		ID:           r.ID,
		PackageID:    r.PackageID,
		Version:      r.Version,
		PublishedAt:  r.PublishedAt,
		TarballURL:   r.TarballURL,
		Status:       string(r.Status),
		ErrorMessage: r.ErrorMessage,
		CreatedAt:    r.CreatedAt,
	}
}

// ReleasesFromEntities maps a slice of domain Releases to response DTOs.
func ReleasesFromEntities(rs []entity.Release) []ReleaseResponse {
	result := make([]ReleaseResponse, len(rs))
	for i := range rs {
		result[i] = ReleaseFromEntity(&rs[i])
	}
	return result
}

// DiffResponse is the JSON representation of a release diff.
type DiffResponse struct {
	ID               string    `json:"id"`
	ReleaseID        string    `json:"releaseId"`
	PrevReleaseID    string    `json:"prevReleaseId"`
	DiffContent      string    `json:"diffContent"`
	FileChangesCount int       `json:"fileChangesCount"`
	LinesAdded       int       `json:"linesAdded"`
	LinesRemoved     int       `json:"linesRemoved"`
	Truncated        bool      `json:"truncated"`
	OriginalSize     int       `json:"originalSize"`
	CreatedAt        time.Time `json:"createdAt"`
}

// DiffFromEntity maps a domain Diff to a response DTO.
func DiffFromEntity(d *entity.Diff) *DiffResponse {
	if d == nil {
		return nil
	}
	return &DiffResponse{
		ID:               d.ID,
		ReleaseID:        d.ReleaseID,
		PrevReleaseID:    d.PrevReleaseID,
		DiffContent:      d.DiffContent,
		FileChangesCount: d.FileChangesCount,
		LinesAdded:       d.LinesAdded,
		LinesRemoved:     d.LinesRemoved,
		Truncated:        d.Truncated,
		OriginalSize:     d.OriginalSize,
		CreatedAt:        d.CreatedAt,
	}
}

// AnalysisResponse is the JSON representation of an LLM analysis result.
type AnalysisResponse struct {
	ID             string    `json:"id"`
	DiffID         string    `json:"diffId"`
	Classification string    `json:"classification"`
	Confidence     float64   `json:"confidence"`
	Reasoning      string    `json:"reasoning"`
	ModelUsed      string    `json:"modelUsed"`
	AnalyzerType   string    `json:"analyzerType"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AnalysisFromEntity maps a domain Analysis to a response DTO.
func AnalysisFromEntity(a *entity.Analysis) *AnalysisResponse {
	if a == nil {
		return nil
	}
	return &AnalysisResponse{
		ID:             a.ID,
		DiffID:         a.DiffID,
		Classification: string(a.Classification),
		Confidence:     a.Confidence,
		Reasoning:      a.Reasoning,
		ModelUsed:      a.ModelUsed,
		AnalyzerType:   string(a.AnalyzerType),
		CreatedAt:      a.CreatedAt,
	}
}

// ReleaseDetailResponse is the JSON representation of a release with its diff, analysis, and package info.
type ReleaseDetailResponse struct {
	ReleaseResponse
	Diff       *DiffResponse     `json:"diff,omitempty"`
	Analysis   *AnalysisResponse `json:"analysis,omitempty"`
	Package    *PackageResponse  `json:"package,omitempty"`
	IsBaseline bool              `json:"isBaseline,omitempty"`
}

// ReleaseDetailFromEntity maps a domain ReleaseDetail to a response DTO.
func ReleaseDetailFromEntity(d *entity.ReleaseDetail) ReleaseDetailResponse {
	resp := ReleaseDetailResponse{
		ReleaseResponse: ReleaseFromEntity(&d.Release),
		Diff:            DiffFromEntity(d.Diff),
		Analysis:        AnalysisFromEntity(d.Analysis),
		IsBaseline:      d.IsBaseline,
	}
	if d.Package != nil {
		pkg := PackageFromEntity(d.Package)
		resp.Package = &pkg
	}
	return resp
}

// RecentReleaseResponse combines a release with its package info and classification for list views.
type RecentReleaseResponse struct {
	ReleaseResponse
	PackageName      string `json:"packageName"`
	PackageEcosystem string `json:"packageEcosystem"`
	Classification   string `json:"classification,omitempty"`
}

// RecentReleaseFromEntity maps a domain ReleaseWithDetails to a response DTO.
func RecentReleaseFromEntity(r *entity.ReleaseWithDetails) RecentReleaseResponse {
	return RecentReleaseResponse{
		ReleaseResponse:  ReleaseFromEntity(&r.Release),
		PackageName:      r.PackageName,
		PackageEcosystem: r.PackageEcosystem,
		Classification:   r.Classification,
	}
}

// RecentReleasesFromEntities maps a slice of domain ReleaseWithDetails to response DTOs.
func RecentReleasesFromEntities(rs []entity.ReleaseWithDetails) []RecentReleaseResponse {
	result := make([]RecentReleaseResponse, len(rs))
	for i := range rs {
		result[i] = RecentReleaseFromEntity(&rs[i])
	}
	return result
}

// AnalysisHistoryEntryResponse is the JSON representation of a single analysis history entry.
type AnalysisHistoryEntryResponse struct {
	ReleaseID      string  `json:"releaseId"`
	Version        string  `json:"version"`
	Classification string  `json:"classification"`
	Confidence     float64 `json:"confidence"`
	Reasoning      string  `json:"reasoning"`
	ModelUsed      string  `json:"modelUsed"`
	AnalyzerType   string  `json:"analyzerType"`
	AnalyzedAt     string  `json:"analyzedAt"`
	PublishedAt    string  `json:"publishedAt"`
}

// AnalysisHistoryFromEntity maps a domain AnalysisHistoryEntry to a response DTO.
func AnalysisHistoryFromEntity(e *entity.AnalysisHistoryEntry) AnalysisHistoryEntryResponse {
	return AnalysisHistoryEntryResponse{
		ReleaseID:      e.ReleaseID,
		Version:        e.Version,
		Classification: e.Classification,
		Confidence:     e.Confidence,
		Reasoning:      e.Reasoning,
		ModelUsed:      e.ModelUsed,
		AnalyzerType:   e.AnalyzerType,
		AnalyzedAt:     e.AnalyzedAt,
		PublishedAt:    e.PublishedAt,
	}
}

// AnalysisHistoryFromEntities maps a slice of domain AnalysisHistoryEntry to response DTOs.
func AnalysisHistoryFromEntities(entries []entity.AnalysisHistoryEntry) []AnalysisHistoryEntryResponse {
	result := make([]AnalysisHistoryEntryResponse, len(entries))
	for i := range entries {
		result[i] = AnalysisHistoryFromEntity(&entries[i])
	}
	return result
}

// ReanalyzeResponse is the JSON representation of a reanalysis request result.
type ReanalyzeResponse struct {
	Message string `json:"message"`
	JobID   string `json:"jobId"`
}

// ReleaseHashResponse is the JSON representation of a release file hash (IoC).
type ReleaseHashResponse struct {
	ID        string    `json:"id"`
	ReleaseID string    `json:"releaseId"`
	Filename  string    `json:"filename"`
	Algorithm string    `json:"algorithm"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"createdAt"`
}

// ReleaseHashFromEntity maps a domain ReleaseHash to a response DTO.
func ReleaseHashFromEntity(h *entity.ReleaseHash) ReleaseHashResponse {
	return ReleaseHashResponse{
		ID:        h.ID,
		ReleaseID: h.ReleaseID,
		Filename:  h.Filename,
		Algorithm: h.Algorithm,
		Hash:      h.Hash,
		CreatedAt: h.CreatedAt,
	}
}

// ReleaseHashesFromEntities maps a slice of domain ReleaseHash to response DTOs.
func ReleaseHashesFromEntities(hashes []entity.ReleaseHash) []ReleaseHashResponse {
	result := make([]ReleaseHashResponse, len(hashes))
	for i := range hashes {
		result[i] = ReleaseHashFromEntity(&hashes[i])
	}
	return result
}

// PipelineStatusResponse is the JSON representation of release pipeline status counts.
type PipelineStatusResponse struct {
	Pending   int64 `json:"pending"`
	Diffing   int64 `json:"diffing"`
	Analyzing int64 `json:"analyzing"`
	Completed int64 `json:"completed"`
	Error     int64 `json:"error"`
}

// PipelineStatusFromEntity maps a domain PipelineStatus to a response DTO.
func PipelineStatusFromEntity(s *entity.PipelineStatus) PipelineStatusResponse {
	return PipelineStatusResponse{
		Pending:   s.Pending,
		Diffing:   s.Diffing,
		Analyzing: s.Analyzing,
		Completed: s.Completed,
		Error:     s.Error,
	}
}
