package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
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

// ListPackages returns a paginated list of packages scoped to the current workspace.
func (h *PackageHandlers) ListPackages(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"name":          "name",
		"ecosystem":     "ecosystem",
		"latestVersion": "latest_version",
		"downloadCount": "download_count",
		"source":        "source",
		"status":        "status",
		"createdAt":     "created_at",
	}, "download_count DESC, name ASC")

	var filters entity.PackageFilters
	if eco := r.URL.Query().Get("ecosystem"); eco != "" {
		e := entity.Ecosystem(eco)
		filters.Ecosystem = &e
	}
	if src := r.URL.Query().Get("source"); src != "" {
		s := entity.PackageSource(src)
		filters.Source = &s
	}
	if st := r.URL.Query().Get("status"); st != "" {
		s := entity.PackageStatus(st)
		filters.Status = &s
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}

	packages, total, err := h.PkgSvc.ListPackages(r.Context(), workspaceID, page, limit, sortOrder, filters)
	if err != nil {
		respondAppError(w, Internal("failed to list packages"))
		return
	}

	respondJSON(w, http.StatusOK, response.PackagesFromEntities(packages), &Meta{Page: page, Limit: limit, Total: total})
}

// GetPackage returns a single package scoped to the current workspace.
func (h *PackageHandlers) GetPackage(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	pkg, err := h.PkgSvc.GetPackage(r.Context(), workspaceID, id)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to get package"))
		return
	}

	respondJSON(w, http.StatusOK, response.PackageFromEntity(pkg), nil)
}

type createPackageRequest struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

// CreatePackage adds a custom package to monitor within the current workspace.
func (h *PackageHandlers) CreatePackage(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	var req createPackageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	if req.Name == "" {
		respondAppError(w, Validation("name is required"))
		return
	}
	if err := validatePackageName(req.Name); err != nil {
		respondAppError(w, ValidationFromErr(err))
		return
	}
	if req.Ecosystem != "python" && req.Ecosystem != "npm" && req.Ecosystem != "go" {
		respondAppError(w, Validation("ecosystem must be 'python', 'npm', or 'go'"))
		return
	}

	pkg, err := h.PkgSvc.CreatePackage(r.Context(), workspaceID, req.Name, entity.Ecosystem(req.Ecosystem))
	if err != nil {
		if errors.Is(err, entity.ErrConflict) {
			respondAppError(w, Conflict("package already monitored"))
			return
		}
		if errors.Is(err, entity.ErrValidation) {
			respondAppError(w, ValidationFromErr(err))
			return
		}
		respondAppError(w, Internal("failed to create package"))
		return
	}

	respondJSON(w, http.StatusCreated, response.PackageFromEntity(pkg), nil)
}

// DeletePackage removes a package from monitoring within the current workspace.
func (h *PackageHandlers) DeletePackage(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	if err := h.PkgSvc.RemovePackage(r.Context(), workspaceID, id); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to remove package"))
		return
	}

	respondJSON(w, http.StatusOK, nil, nil)
}

// blockPackageRequest is the request body for blocking a package.
type blockPackageRequest struct {
	Reason string `json:"reason"`
}

// BlockPackage sets a package's status to 'blocked'.
func (h *PackageHandlers) BlockPackage(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	var req blockPackageRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondAppError(w, BadRequest("invalid request body"))
			return
		}
	}

	pkg, err := h.PkgSvc.BlockPackage(r.Context(), workspaceID, id, req.Reason)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to block package"))
		return
	}

	respondJSON(w, http.StatusOK, response.PackageFromEntity(pkg), nil)
}

// UnblockPackage sets a package's status back to 'active'.
func (h *PackageHandlers) UnblockPackage(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	pkg, err := h.PkgSvc.UnblockPackage(r.Context(), workspaceID, id)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to unblock package"))
		return
	}

	respondJSON(w, http.StatusOK, response.PackageFromEntity(pkg), nil)
}

// importPackageEntry represents a single package in a bulk import request.
type importPackageEntry struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
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
		entries = append(entries, importPackageEntry{Name: name, Ecosystem: "python"})
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
			entries = append(entries, importPackageEntry{Name: name, Ecosystem: "npm"})
			seen[name] = true
		}
	}
	for name := range pkgJSON.DevDependencies {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			entries = append(entries, importPackageEntry{Name: name, Ecosystem: "npm"})
			seen[name] = true
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no packages found in package.json dependencies")
	}
	return entries, nil
}

// parseListFormat parses newline-separated "ecosystem:name" pairs.
// Example: "python:requests\nnpm:express\npython:flask"
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
		ecosystem := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if ecosystem == "" || name == "" {
			continue
		}
		entries = append(entries, importPackageEntry{Name: name, Ecosystem: ecosystem})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no packages found in list content")
	}
	return entries, nil
}

// ImportPackages handles POST /api/packages/bulk-import - bulk adds packages to monitoring.
// Supports format-based parsing (requirements_txt, package_json, list) or a legacy
// pre-parsed packages array.
func (h *PackageHandlers) ImportPackages(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	var req importPackagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	var entries []importPackageEntry

	// Format-based parsing takes priority over legacy packages array.
	if req.Format != "" {
		if req.Content == "" {
			respondAppError(w, Validation("content is required when format is specified"))
			return
		}
		// Limit content size to prevent expensive processing (Finding 10)
		const maxContentSize = 100 * 1024 // 100KB
		if len(req.Content) > maxContentSize {
			respondAppError(w, Validation(fmt.Sprintf("content too large (max %dKB)", maxContentSize/1024)))
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
			respondAppError(w, Validation("format must be 'requirements_txt', 'package_json', or 'list'"))
			return
		}
		if err != nil {
			respondAppError(w, ValidationFromErr(err))
			return
		}
	} else if len(req.Packages) > 0 {
		// Legacy mode: pre-parsed packages array.
		entries = req.Packages
	} else {
		respondAppError(w, Validation("either 'format'+'content' or 'packages' array is required"))
		return
	}

	const maxImport = 500
	if len(entries) > maxImport {
		respondAppError(w, Validation(fmt.Sprintf("too many packages (max %d, got %d)", maxImport, len(entries))))
		return
	}

	// Convert to entity.ImportEntry, validating each entry.
	// Invalid entries are collected as errors and excluded from the import.
	var importEntries []entity.ImportEntry
	var importErrors []entity.ImportErrorEntry
	for _, e := range entries {
		if e.Ecosystem != "python" && e.Ecosystem != "npm" && e.Ecosystem != "go" {
			importErrors = append(importErrors, entity.ImportErrorEntry{
				Name:  e.Name,
				Error: fmt.Sprintf("unsupported ecosystem %q (must be 'python', 'npm', or 'go')", e.Ecosystem),
			})
			continue
		}
		if err := validatePackageName(e.Name); err != nil {
			importErrors = append(importErrors, entity.ImportErrorEntry{
				Name:  e.Name,
				Error: err.Error(),
			})
			continue
		}
		importEntries = append(importEntries, entity.ImportEntry{Name: e.Name, Ecosystem: entity.Ecosystem(e.Ecosystem)})
	}

	result, err := h.PkgSvc.ImportPackages(r.Context(), workspaceID, importEntries)
	if err != nil {
		respondAppError(w, Internal("failed to import packages"))
		return
	}

	// Merge handler-level validation errors with usecase-level errors
	result.Errors = append(importErrors, result.Errors...)

	h.Audit.LogAction(r.Context(), "import", "package", "",
		fmt.Sprintf("bulk imported %d packages (%d created, %d skipped, %d errors, format=%s)",
			len(entries), result.Imported, result.Skipped, len(result.Errors), req.Format))

	respondJSON(w, http.StatusOK, response.ImportResultFromEntity(result), nil)
}

// ListSuggestions returns a paginated list of suggested packages for the current workspace.
// GET /api/packages/suggestions
func (h *PackageHandlers) ListSuggestions(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"name":          "name",
		"ecosystem":     "ecosystem",
		"downloadCount": "download_count",
	}, "download_count DESC, name ASC")

	var filters entity.PackageFilters
	if eco := r.URL.Query().Get("ecosystem"); eco != "" {
		e := entity.Ecosystem(eco)
		filters.Ecosystem = &e
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}

	packages, total, err := h.PkgSvc.ListSuggestions(r.Context(), workspaceID, page, limit, sortOrder, filters)
	if err != nil {
		respondAppError(w, Internal("failed to list suggestions"))
		return
	}

	respondJSON(w, http.StatusOK, response.PackagesFromEntities(packages), &Meta{Page: page, Limit: limit, Total: total})
}

// ApprovePackage promotes a suggested package to active monitoring.
// POST /api/packages/{id}/approve
func (h *PackageHandlers) ApprovePackage(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	pkg, err := h.PkgSvc.ApprovePackage(r.Context(), workspaceID, id)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to approve package"))
		return
	}

	respondJSON(w, http.StatusOK, response.PackageFromEntity(pkg), nil)
}

// RejectPackage rejects a suggested package.
// POST /api/packages/{id}/reject
func (h *PackageHandlers) RejectPackage(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid package ID"))
		return
	}

	if err := h.PkgSvc.RejectPackage(r.Context(), workspaceID, id); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("package"))
			return
		}
		respondAppError(w, Internal("failed to reject package"))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// bulkApproveRequest is the request body for bulk-approving suggested packages.
type bulkApproveRequest struct {
	PackageIDs []string `json:"packageIds"`
	Ecosystem  string   `json:"ecosystem"`
	ApproveAll bool     `json:"approveAll"`
}

// BulkApprovePackages approves multiple suggested packages at once.
// POST /api/packages/bulk-approve
func (h *PackageHandlers) BulkApprovePackages(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	var req bulkApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	// Approve ALL pending suggestions for this workspace (no IDs needed).
	if req.ApproveAll {
		count, err := h.PkgSvc.BulkApproveAllSuggestions(r.Context(), workspaceID)
		if err != nil {
			respondAppError(w, Internal("failed to bulk approve packages"))
			return
		}
		respondJSON(w, http.StatusOK, map[string]int{"approved": count}, nil)
		return
	}

	var pkgIDs []string

	if len(req.PackageIDs) > 0 {
		const maxBulkApprove = 1000
		if len(req.PackageIDs) > maxBulkApprove {
			respondAppError(w, Validation(fmt.Sprintf("cannot approve more than %d packages at once", maxBulkApprove)))
			return
		}
		for _, id := range req.PackageIDs {
			if _, err := uuid.Parse(id); err != nil {
				respondAppError(w, BadRequest(fmt.Sprintf("invalid package ID: %s", id)))
				return
			}
		}
		pkgIDs = req.PackageIDs
	} else if req.Ecosystem != "" {
		// Approve all suggestions for the given ecosystem: fetch all suggested packages
		// and filter by ecosystem.
		if req.Ecosystem != "python" && req.Ecosystem != "npm" && req.Ecosystem != "go" {
			respondAppError(w, Validation("ecosystem must be 'python', 'npm', or 'go'"))
			return
		}
		packages, _, err := h.PkgSvc.ListSuggestions(r.Context(), workspaceID, 1, 10000, "", entity.PackageFilters{})
		if err != nil {
			respondAppError(w, Internal("failed to list suggestions"))
			return
		}
		for _, pkg := range packages {
			if string(pkg.Ecosystem) == req.Ecosystem {
				pkgIDs = append(pkgIDs, pkg.ID)
			}
		}
		if len(pkgIDs) == 0 {
			respondJSON(w, http.StatusOK, map[string]int{"approved": 0}, nil)
			return
		}
	} else {
		respondAppError(w, Validation("either 'packageIds' or 'ecosystem' is required"))
		return
	}

	count, err := h.PkgSvc.BulkApprovePackages(r.Context(), workspaceID, pkgIDs)
	if err != nil {
		respondAppError(w, Internal("failed to bulk approve packages"))
		return
	}

	respondJSON(w, http.StatusOK, map[string]int{"approved": count}, nil)
}

// ListStalePackages returns packages that haven't had a release in a configurable number of months.
// GET /api/packages/stale
func (h *PackageHandlers) ListStalePackages(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	months := 6
	if m := r.URL.Query().Get("months"); m != "" {
		parsed, err := strconv.Atoi(m)
		if err != nil || parsed < 1 || parsed > 24 {
			respondAppError(w, Validation("months must be between 1 and 24"))
			return
		}
		months = parsed
	}

	staleBefore := time.Now().AddDate(0, -months, 0)

	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"name":          "name",
		"ecosystem":     "ecosystem",
		"latestVersion": "latest_version",
		"downloadCount": "download_count",
		"source":        "source",
		"createdAt":     "created_at",
	}, "download_count DESC, name ASC")

	var filters entity.PackageFilters
	if eco := r.URL.Query().Get("ecosystem"); eco != "" {
		e := entity.Ecosystem(eco)
		filters.Ecosystem = &e
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}

	packages, total, err := h.PkgSvc.ListStalePackages(r.Context(), workspaceID, staleBefore, page, limit, sortOrder, filters)
	if err != nil {
		respondAppError(w, Internal("failed to list stale packages"))
		return
	}

	respondJSON(w, http.StatusOK, response.PackagesFromEntities(packages), &Meta{Page: page, Limit: limit, Total: total})
}
