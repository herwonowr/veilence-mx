package v1

import (
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/setup"
)

// ConfigHandlers handles public config endpoints.
type ConfigHandlers struct {
	Setup *setup.Service
	Auth  *auth.Service
}

// GetPublicConfig handles GET /api/config/public - returns platform configuration
// the frontend needs before authentication.
func (h *ConfigHandlers) GetPublicConfig(w http.ResponseWriter, r *http.Request) {
	setupRequired, err := h.Setup.IsSetupRequired(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check setup status")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"registrationEnabled":       h.Auth.RegistrationEnabled(),
		"setupRequired":             setupRequired,
		"hasEmailDomainRestriction": h.Auth.HasEmailDomainRestriction(),
	}, nil)
}
