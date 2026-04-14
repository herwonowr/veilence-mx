package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// GetSettings returns all settings as a key-value map scoped to the current org.
func (h *SettingsHandlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	var settings []models.Setting
	if err := h.DB.Where("org_id = ?", orgID).Find(&settings).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to load settings"))
		return
	}

	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	respondJSON(w, http.StatusOK, result, nil)
}

// UpdateSettings updates settings from a key-value map scoped to the current org.
func (h *SettingsHandlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	for key, value := range req {
		if !domain.ValidSettingKeys[key] {
			respondAppError(w, apperror.Validation("invalid setting key: "+key))
			return
		}

		var setting models.Setting
		result := h.DB.Where("org_id = ? AND key = ?", orgID, key).First(&setting)
		if result.Error != nil {
			// Create new setting
			if err := h.DB.Create(&models.Setting{OrgID: orgID, Key: key, Value: value}).Error; err != nil {
				respondAppError(w, apperror.Internal("failed to save setting"))
				return
			}
		} else {
			// Update existing
			if err := h.DB.Model(&setting).Update("value", value).Error; err != nil {
				respondAppError(w, apperror.Internal("failed to update setting"))
				return
			}
		}
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
		if _, changed := req[models.SettingDiscoveryScanDepth]; changed {
			h.Poller.TriggerDiscovery(orgID)
		}
	}

	// Return updated settings
	h.GetSettings(w, r)
}

// DiscoverPackages triggers an immediate discovery cycle for the current org.
func (h *SettingsHandlers) DiscoverPackages(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	// Read discovery_scan_depth setting (default 50)
	scanDepth := 50
	var setting models.Setting
	if err := h.DB.Where("org_id = ? AND key = ?", orgID, models.SettingDiscoveryScanDepth).First(&setting).Error; err == nil {
		if v, err := json.Number(setting.Value).Int64(); err == nil && v > 0 {
			scanDepth = int(v)
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
