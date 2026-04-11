package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// ListPackages returns a paginated list of packages scoped to the current org.
func (h *PackageHandlers) ListPackages(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"name":          "name",
		"registry":      "registry",
		"latestVersion": "latest_version",
		"rank":          "rank",
		"isCustom":      "is_custom",
		"createdAt":     "created_at",
	}, "rank ASC NULLS LAST, name ASC")
	registryFilter := r.URL.Query().Get("registry")
	isCustomFilter := r.URL.Query().Get("is_custom")
	search := r.URL.Query().Get("search")

	query := h.DB.Model(&models.Package{}).Where("org_id = ?", orgID)
	if registryFilter != "" {
		query = query.Where("registry = ?", registryFilter)
	}
	if isCustomFilter != "" {
		query = query.Where("is_custom = ?", isCustomFilter == "true")
	}
	if search != "" {
		search = escapeLike(search)
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to count packages"))
		return
	}

	var packages []models.Package
	if err := query.Order(sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&packages).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to list packages"))
		return
	}

	respondJSON(w, http.StatusOK, packages, &Meta{Page: page, Limit: limit, Total: total})
}

// GetPackage returns a single package scoped to the current org.
func (h *PackageHandlers) GetPackage(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid package ID"))
		return
	}

	var pkg models.Package
	if err := h.DB.Where("id = ? AND org_id = ?", id, orgID).First(&pkg).Error; err != nil {
		respondAppError(w, apperror.NotFound("package"))
		return
	}

	respondJSON(w, http.StatusOK, pkg, nil)
}

type createPackageRequest struct {
	Name     string `json:"name"`
	Registry string `json:"registry"`
}

// CreatePackage adds a custom package to monitor within the current org.
func (h *PackageHandlers) CreatePackage(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	var req createPackageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, apperror.BadRequest("invalid request body"))
		return
	}

	if req.Name == "" {
		respondAppError(w, apperror.Validation("name is required"))
		return
	}
	if req.Registry != "pypi" && req.Registry != "npm" {
		respondAppError(w, apperror.Validation("registry must be 'pypi' or 'npm'"))
		return
	}

	var existing models.Package
	if tx := h.DB.Where("org_id = ? AND name = ? AND registry = ?", orgID, req.Name, req.Registry).Limit(1).Find(&existing); tx.RowsAffected > 0 {
		respondAppError(w, apperror.Conflict("package already monitored"))
		return
	}

	pkg := models.Package{
		OrgID:    orgID,
		Name:     req.Name,
		Registry: models.Registry(req.Registry),
		IsCustom: true,
	}

	if err := h.DB.Create(&pkg).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to create package"))
		return
	}

	h.Audit.LogAction(r.Context(), "create", "package", pkg.ID, fmt.Sprintf("added %s package %q to monitoring", req.Registry, req.Name))

	respondJSON(w, http.StatusCreated, pkg, nil)
}

// DeletePackage removes a package from monitoring within the current org.
func (h *PackageHandlers) DeletePackage(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid package ID"))
		return
	}

	result := h.DB.Where("id = ? AND org_id = ?", id, orgID).Delete(&models.Package{})
	if result.Error != nil {
		respondAppError(w, apperror.Internal("failed to delete package"))
		return
	}
	if result.RowsAffected == 0 {
		respondAppError(w, apperror.NotFound("package"))
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "package", uint(id), fmt.Sprintf("removed package %d from monitoring", id))

	respondJSON(w, http.StatusOK, nil, nil)
}
