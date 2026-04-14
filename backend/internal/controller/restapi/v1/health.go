package v1

import (
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
)

// HealthCheck handles GET /api/health — checks database and Redis connectivity.
func (h *HealthHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := h.HealthSvc.HealthCheck(r.Context())

	status := http.StatusOK
	if resp.Status == "degraded" {
		status = http.StatusServiceUnavailable
	}

	respondJSON(w, status, response.HealthFromEntity(resp), nil)
}

// ReadinessCheck handles GET /api/ready — returns 200 only when all dependencies
// (database and Redis) are reachable. Designed for Kubernetes readiness probes
// and load balancer health checks. Returns 503 if any dependency is unreachable.
func (h *HealthHandlers) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	resp := h.HealthSvc.ReadinessCheck(r.Context())

	status := http.StatusOK
	if !resp.Ready {
		status = http.StatusServiceUnavailable
	}

	respondJSON(w, status, response.ReadinessFromEntity(resp), nil)
}
