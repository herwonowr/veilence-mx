package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

type releaseDetail struct {
	models.Release
	Diff       *models.Diff     `json:"diff,omitempty"`
	Analysis   *models.Analysis `json:"analysis,omitempty"`
	Package    *models.Package  `json:"package,omitempty"`
	IsBaseline bool             `json:"isBaseline,omitempty"`
}

// ListPackageReleases returns releases for a specific package scoped to the current org.
func (h *PackageHandlers) ListPackageReleases(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	packageID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid package ID"))
		return
	}

	// Verify package belongs to the requesting org
	var pkg models.Package
	if err := h.DB.Where("id = ? AND org_id = ?", packageID, orgID).First(&pkg).Error; err != nil {
		respondAppError(w, apperror.NotFound("package"))
		return
	}

	page, limit := parsePagination(r)

	var total int64
	if err := h.DB.Model(&models.Release{}).Where("package_id = ?", packageID).Count(&total).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to count releases"))
		return
	}

	var releases []models.Release
	if err := h.DB.Where("package_id = ?", packageID).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&releases).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to list releases"))
		return
	}

	respondJSON(w, http.StatusOK, releases, &Meta{Page: page, Limit: limit, Total: total})
}

// GetRelease returns a single release with its diff and analysis, scoped to the current org.
func (h *PackageHandlers) GetRelease(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid release ID"))
		return
	}

	var release models.Release
	if err := h.DB.Preload("Package").First(&release, id).Error; err != nil {
		respondAppError(w, apperror.NotFound("release"))
		return
	}

	// Verify the release's package belongs to the requesting org
	if release.Package.OrgID != orgID {
		respondAppError(w, apperror.NotFound("release"))
		return
	}

	detail := releaseDetail{
		Release: release,
		Package: &release.Package,
	}

	var diff models.Diff
	if tx := h.DB.Where("release_id = ?", release.ID).Limit(1).Find(&diff); tx.RowsAffected > 0 {
		detail.Diff = &diff
		var analysis models.Analysis
		if tx := h.DB.Where("diff_id = ?", diff.ID).Limit(1).Find(&analysis); tx.RowsAffected > 0 {
			detail.Analysis = &analysis
		}
	} else if release.Status == models.ReleaseStatusCompleted {
		// Completed release with no diff is a baseline (first tracked version)
		detail.IsBaseline = true
	}

	respondJSON(w, http.StatusOK, detail, nil)
}
