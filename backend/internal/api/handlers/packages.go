package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// packageNamePattern validates package name format.
// Allows alphanumeric characters, dots, underscores, hyphens, forward slashes (for npm scoped packages like @scope/pkg),
// and @ (for npm scoped packages).
var packageNamePattern = regexp.MustCompile(`^[@a-zA-Z0-9._/-]+$`)

// maxPackageNameLength is the maximum allowed length for a package name.
const maxPackageNameLength = 200

// validatePackageName checks that a package name matches the allowed pattern and length.
func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > maxPackageNameLength {
		return fmt.Errorf("name must be at most %d characters", maxPackageNameLength)
	}
	if !packageNamePattern.MatchString(name) {
		return fmt.Errorf("name contains invalid characters (allowed: letters, digits, dots, underscores, hyphens, @, /)")
	}
	return nil
}

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
	if err := validatePackageName(req.Name); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
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

// importPackageEntry represents a single package in a bulk import request.
type importPackageEntry struct {
	Name     string `json:"name"`
	Registry string `json:"registry"`
}

// importPackagesRequest is the request body for bulk package import.
// Supports two modes:
//   - Format-based: {format: "requirements_txt"|"package_json"|"list", content: "..."}
//   - Legacy array: {packages: [{name, registry}, ...]}
type importPackagesRequest struct {
	Format   string               `json:"format"`
	Content  string               `json:"content"`
	Packages []importPackageEntry `json:"packages"`
}

// importErrorEntry represents a single error in the import result.
type importErrorEntry struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// importResult summarizes the outcome of a bulk import.
type importResult struct {
	Imported int                `json:"imported"`
	Skipped  int                `json:"skipped"`
	Errors   []importErrorEntry `json:"errors,omitempty"`
}

// parseRequirementsTxt parses a requirements.txt file content into package entries.
// Strips comments (#), empty lines, pip options (-i, --index-url, etc.),
// and version specifiers (==, >=, ~=, !=, <, >).
func parseRequirementsTxt(content string) ([]importPackageEntry, error) {
	var entries []importPackageEntry
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Skip empty lines, comments, and pip options
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		// Strip inline comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		// Extract package name by stripping version specifiers
		// Handles: requests==2.31.0, flask>=2.0, numpy, django~=4.0, pkg[extra]>=1.0
		name := line
		for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">", "<", ";"} {
			if idx := strings.Index(name, sep); idx >= 0 {
				name = name[:idx]
			}
		}
		// Strip extras like [security,socks]
		if idx := strings.Index(name, "["); idx >= 0 {
			name = name[:idx]
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		entries = append(entries, importPackageEntry{Name: name, Registry: "pypi"})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no packages found in requirements.txt content")
	}
	return entries, nil
}

// parsePackageJSON parses a package.json file content into package entries.
// Extracts package names from "dependencies" and "devDependencies" fields.
func parsePackageJSON(content string) ([]importPackageEntry, error) {
	var pkgJSON struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(content), &pkgJSON); err != nil {
		return nil, fmt.Errorf("invalid package.json: %w", err)
	}

	seen := make(map[string]bool)
	var entries []importPackageEntry
	for name := range pkgJSON.Dependencies {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			entries = append(entries, importPackageEntry{Name: name, Registry: "npm"})
			seen[name] = true
		}
	}
	for name := range pkgJSON.DevDependencies {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			entries = append(entries, importPackageEntry{Name: name, Registry: "npm"})
			seen[name] = true
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no packages found in package.json dependencies")
	}
	return entries, nil
}

// parseListFormat parses newline-separated "registry:name" pairs.
// Example: "pypi:requests\nnpm:express\npypi:flask"
func parseListFormat(content string) ([]importPackageEntry, error) {
	var entries []importPackageEntry
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		registry := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if registry == "" || name == "" {
			continue
		}
		entries = append(entries, importPackageEntry{Name: name, Registry: registry})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no packages found in list content")
	}
	return entries, nil
}

// ImportPackages handles POST /api/packages/bulk-import — bulk adds packages to monitoring.
// Supports format-based parsing (requirements_txt, package_json, list) or a legacy
// pre-parsed packages array.
func (h *PackageHandlers) ImportPackages(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	var req importPackagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, apperror.BadRequest("invalid request body"))
		return
	}

	var entries []importPackageEntry

	// Format-based parsing takes priority over legacy packages array.
	if req.Format != "" {
		if req.Content == "" {
			respondAppError(w, apperror.Validation("content is required when format is specified"))
			return
		}
		var err error
		switch req.Format {
		case "requirements_txt":
			entries, err = parseRequirementsTxt(req.Content)
		case "package_json":
			entries, err = parsePackageJSON(req.Content)
		case "list":
			entries, err = parseListFormat(req.Content)
		default:
			respondAppError(w, apperror.Validation("format must be 'requirements_txt', 'package_json', or 'list'"))
			return
		}
		if err != nil {
			respondAppError(w, apperror.Validation(err.Error()))
			return
		}
	} else if len(req.Packages) > 0 {
		// Legacy mode: pre-parsed packages array.
		entries = req.Packages
	} else {
		respondAppError(w, apperror.Validation("either 'format'+'content' or 'packages' array is required"))
		return
	}

	const maxImport = 500
	if len(entries) > maxImport {
		respondAppError(w, apperror.Validation(fmt.Sprintf("too many packages (max %d, got %d)", maxImport, len(entries))))
		return
	}

	result := importResult{}
	validRegistries := map[string]bool{"pypi": true, "npm": true}

	for _, entry := range entries {
		if entry.Name == "" {
			result.Errors = append(result.Errors, importErrorEntry{Name: "(empty)", Error: "name is required"})
			continue
		}
		if err := validatePackageName(entry.Name); err != nil {
			result.Errors = append(result.Errors, importErrorEntry{Name: entry.Name, Error: err.Error()})
			continue
		}
		if !validRegistries[entry.Registry] {
			result.Errors = append(result.Errors, importErrorEntry{Name: entry.Name, Error: fmt.Sprintf("registry must be 'pypi' or 'npm', got %q", entry.Registry)})
			continue
		}

		// Check for existing package
		var existing models.Package
		if tx := h.DB.Where("org_id = ? AND name = ? AND registry = ?", orgID, entry.Name, entry.Registry).Limit(1).Find(&existing); tx.RowsAffected > 0 {
			result.Skipped++
			continue
		}

		pkg := models.Package{
			OrgID:    orgID,
			Name:     entry.Name,
			Registry: models.Registry(entry.Registry),
			IsCustom: true,
		}

		if err := h.DB.Create(&pkg).Error; err != nil {
			slog.Error("failed to import package", "name", entry.Name, "registry", entry.Registry, "error", err)
			result.Errors = append(result.Errors, importErrorEntry{Name: entry.Name, Error: "failed to create package"})
			continue
		}
		result.Imported++
	}

	h.Audit.LogAction(r.Context(), "import", "package", 0,
		fmt.Sprintf("bulk imported %d packages (%d created, %d skipped, %d errors, format=%s)",
			len(entries), result.Imported, result.Skipped, len(result.Errors), req.Format))

	respondJSON(w, http.StatusOK, result, nil)
}
