package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// GetSettings returns all settings as a key-value map scoped to the current org.
func (h *SettingsHandlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	var settings []models.Setting
	h.DB.Where("org_id = ?", orgID).Find(&settings)

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

	validKeys := map[string]bool{
		models.SettingPyPIPollInterval:    true,
		models.SettingNPMPollInterval:     true,
		models.SettingPyPITopN:            true,
		models.SettingNPMTopN:             true,
		models.SettingAnalyzerMode:        true,
		models.SettingTopNRefreshInterval: true,
		models.SettingDiffSizeLimit:       true,
		models.SettingVersionDepthMode:    true,
		models.SettingVersionDepthCount:   true,
	}

	for key, value := range req {
		if !validKeys[key] {
			respondAppError(w, apperror.Validation("invalid setting key: "+key))
			return
		}

		var setting models.Setting
		result := h.DB.Where("org_id = ? AND key = ?", orgID, key).First(&setting)
		if result.Error != nil {
			// Create new setting
			h.DB.Create(&models.Setting{OrgID: orgID, Key: key, Value: value})
		} else {
			// Update existing
			h.DB.Model(&setting).Update("value", value)
		}
	}

	updatedKeys := make([]string, 0, len(req))
	for key := range req {
		updatedKeys = append(updatedKeys, key)
	}
	h.Audit.LogAction(r.Context(), "update", "setting", 0, fmt.Sprintf("updated settings: %s", strings.Join(updatedKeys, ", ")))

	// Return updated settings
	h.GetSettings(w, r)
}

// SyncTopPackages triggers an immediate top-N package sync for the current org.
func (h *SettingsHandlers) SyncTopPackages(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	registryParam := r.URL.Query().Get("registry")

	if registryParam == "" || registryParam == "pypi" {
		// Get limit from settings or use default
		limit := 100
		var setting models.Setting
		if err := h.DB.Where("org_id = ? AND key = ?", orgID, models.SettingPyPITopN).First(&setting).Error; err == nil {
			if v, err := json.Number(setting.Value).Int64(); err == nil {
				limit = int(v)
			}
		}

		if err := h.Poller.SyncTopPackages(r.Context(), h.PyPI, limit, orgID); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to sync PyPI top packages: "+err.Error())
			return
		}
	}

	if registryParam == "" || registryParam == "npm" {
		limit := 100
		var setting models.Setting
		if err := h.DB.Where("org_id = ? AND key = ?", orgID, models.SettingNPMTopN).First(&setting).Error; err == nil {
			if v, err := json.Number(setting.Value).Int64(); err == nil {
				limit = int(v)
			}
		}

		if err := h.Poller.SyncTopPackages(r.Context(), h.NPM, limit, orgID); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to sync npm top packages: "+err.Error())
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "sync triggered"}, nil)
}
