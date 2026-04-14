package response

import "github.com/veilence/veilence-mx/backend/internal/entity"

// HealthStatusResponse represents the status of a dependency.
type HealthStatusResponse struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

// HealthResponse is the response body for the health check endpoint.
type HealthResponse struct {
	Status   string               `json:"status"`
	Database HealthStatusResponse `json:"database"`
	Redis    HealthStatusResponse `json:"redis"`
}

// HealthFromEntity maps a domain HealthResponse to a response DTO.
func HealthFromEntity(h *entity.HealthResponse) HealthResponse {
	return HealthResponse{
		Status: h.Status,
		Database: HealthStatusResponse{
			Status:  h.Database.Status,
			Latency: h.Database.Latency,
			Error:   h.Database.Error,
		},
		Redis: HealthStatusResponse{
			Status:  h.Redis.Status,
			Latency: h.Redis.Latency,
			Error:   h.Redis.Error,
		},
	}
}

// ReadinessResponse is the response body for the readiness endpoint.
type ReadinessResponse struct {
	Ready    bool                 `json:"ready"`
	Database HealthStatusResponse `json:"database"`
	Redis    HealthStatusResponse `json:"redis"`
}

// ReadinessFromEntity maps a domain ReadinessResponse to a response DTO.
func ReadinessFromEntity(r *entity.ReadinessResponse) ReadinessResponse {
	return ReadinessResponse{
		Ready: r.Ready,
		Database: HealthStatusResponse{
			Status:  r.Database.Status,
			Latency: r.Database.Latency,
			Error:   r.Database.Error,
		},
		Redis: HealthStatusResponse{
			Status:  r.Redis.Status,
			Latency: r.Redis.Latency,
			Error:   r.Redis.Error,
		},
	}
}
