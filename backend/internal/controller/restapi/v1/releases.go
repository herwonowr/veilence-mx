package v1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// ListPackageReleases returns releases for a specific package scoped to the current workspace.
func (h *PackageHandlers) ListPackageReleases(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	packageID, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	page, limit := parsePagination(r)

	releases, total, err := h.ReleaseSvc.ListByPackage(r.Context(), workspaceID, packageID, page, limit)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to list releases"))
		return
	}

	respondJSON(w, http.StatusOK, response.ReleasesFromEntities(releases), &Meta{Page: page, Limit: limit, Total: total})
}

// GetRelease returns a single release with its diff and analysis, scoped to the current workspace.
func (h *PackageHandlers) GetRelease(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid release ID"))
		return
	}

	detail, err := h.ReleaseSvc.GetRelease(r.Context(), workspaceID, id)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("release"))
			return
		}
		respondAppError(w, Internal("failed to get release"))
		return
	}

	respondJSON(w, http.StatusOK, response.ReleaseDetailFromEntity(detail), nil)
}

// ReanalyzeRelease re-queues a single release for analysis.
// POST /api/releases/{id}/reanalyze
func (h *PackageHandlers) ReanalyzeRelease(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid release ID"))
		return
	}

	message, jobID, err := h.ReleaseSvc.ReanalyzeRelease(r.Context(), workspaceID, id)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("release"))
			return
		}
		respondAppError(w, Internal("failed to enqueue reanalysis"))
		return
	}

	h.Audit.LogAction(r.Context(), "reanalyze", "release", id, fmt.Sprintf("triggered re-analysis for release %s", id))

	respondJSON(w, http.StatusOK, response.ReanalyzeResponse{
		Message: message,
		JobID:   jobID,
	}, nil)
}

// GetAnalysisHistory returns the analysis history for a package across all its releases.
// GET /api/packages/{id}/analysis-history
func (h *PackageHandlers) GetAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	packageID, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	entries, err := h.ReleaseSvc.GetAnalysisHistory(r.Context(), workspaceID, packageID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to load analysis history"))
		return
	}

	respondJSON(w, http.StatusOK, response.AnalysisHistoryFromEntities(entries), nil)
}

// GetPipelineStatus returns counts of releases by processing status for the current workspace.
// GET /api/releases/pipeline-status
func (h *PackageHandlers) GetPipelineStatus(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	status, err := h.ReleaseSvc.GetPipelineStatus(r.Context(), workspaceID)
	if err != nil {
		respondAppError(w, Internal("failed to get pipeline status"))
		return
	}

	respondJSON(w, http.StatusOK, response.PipelineStatusFromEntity(status), nil)
}
