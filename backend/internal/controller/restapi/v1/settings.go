package v1

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// GetSettings returns all settings as a key-value map scoped to the current org.
func (h *SettingsHandlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	settings, err := h.SettingSvc.GetSettings(r.Context(), orgID)
	if err != nil {
		respondAppError(w, Internal("failed to load settings"))
		return
	}

	respondJSON(w, http.StatusOK, settings, nil)
}

// UpdateSettings updates settings from a key-value map scoped to the current org.
func (h *SettingsHandlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate keys before delegating to usecase
	for key := range req {
		if !entity.ValidSettingKeys[key] {
			respondAppError(w, Validation("invalid setting key: "+key))
			return
		}
	}

	updated, err := h.SettingSvc.UpdateSettings(r.Context(), orgID, req)
	if err != nil {
		respondAppError(w, Internal("failed to update settings"))
		return
	}

	updatedKeys := make([]string, 0, len(req))
	for key := range req {
		updatedKeys = append(updatedKeys, key)
	}
	h.Audit.LogAction(r.Context(), "update", "setting", 0, fmt.Sprintf("updated settings: %s", strings.Join(updatedKeys, ", ")))

	// Invalidate the poller settings cache so changes take effect immediately
	if h.Poller != nil {
		h.Poller.InvalidateSettingsCache()

		// If discovery_scan_depth changed, trigger an immediate discovery cycle
		if _, changed := req[entity.SettingDiscoveryScanDepth]; changed {
			h.Poller.TriggerDiscovery(orgID)
		}
	}

	respondJSON(w, http.StatusOK, updated, nil)
}

// DiscoverPackages triggers an immediate discovery cycle for the current org.
func (h *SettingsHandlers) DiscoverPackages(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	// Read discovery_scan_depth setting (default 50)
	scanDepth := 50
	settings, err := h.SettingSvc.GetSettings(r.Context(), orgID)
	if err == nil {
		if v, ok := settings[entity.SettingDiscoveryScanDepth]; ok {
			if n, err := json.Number(v).Int64(); err == nil && n > 0 {
				scanDepth = int(n)
			}
		}
	}

	// Discover both ecosystems at the same scan depth
	if err := h.Poller.SyncTopPackages(r.Context(), h.Python, scanDepth, orgID); err != nil {
		slog.Error("failed to discover Python packages", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to discover Python packages")
		return
	}

	if err := h.Poller.SyncTopPackages(r.Context(), h.NPM, scanDepth, orgID); err != nil {
		slog.Error("failed to discover npm packages", "org_id", orgID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to discover npm packages")
		return
	}

	h.Audit.LogAction(r.Context(), "discover", "package", 0,
		fmt.Sprintf("triggered discovery for org (scan_depth=%d)", scanDepth))

	respondJSON(w, http.StatusOK, map[string]string{"message": "discovery triggered"}, nil)
}

// SyncTopPackages triggers an immediate discovery cycle for the current org.
// Deprecated: Use DiscoverPackages (POST /api/sync/discover) instead.
func (h *SettingsHandlers) SyncTopPackages(w http.ResponseWriter, r *http.Request) {
	// Set deprecation header
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Sunset", "2026-07-01")
	w.Header().Set("Link", `</api/sync/discover>; rel="successor-version"`)

	h.DiscoverPackages(w, r)
}
