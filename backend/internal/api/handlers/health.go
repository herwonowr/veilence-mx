package handlers

import (
	"context"
	"net/http"
	"time"
)

// healthStatus represents the status of a dependency.
type healthStatus struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

// healthResponse is the response body for the health check endpoint.
type healthResponse struct {
	Status   string       `json:"status"`
	Database healthStatus `json:"database"`
	Redis    healthStatus `json:"redis"`
}

// HealthCheck handles GET /api/health — checks database and Redis connectivity.
func (h *HealthHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{Status: "ok"}

	// Check database
	dbStart := time.Now()
	sqlDB, err := h.DB.DB()
	if err != nil {
		resp.Database = healthStatus{Status: "error", Error: "failed to get database connection"}
		resp.Status = "degraded"
	} else {
		dbCtx, dbCancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer dbCancel()
		if err := sqlDB.PingContext(dbCtx); err != nil {
			resp.Database = healthStatus{Status: "error", Error: "database ping failed"}
			resp.Status = "degraded"
		} else {
			resp.Database = healthStatus{
				Status:  "ok",
				Latency: time.Since(dbStart).String(),
			}
		}
	}

	// Check Redis
	redisStart := time.Now()
	redisCtx, redisCancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer redisCancel()
	if err := h.Queue.Ping(redisCtx); err != nil {
		resp.Redis = healthStatus{Status: "error", Error: "redis ping failed"}
		resp.Status = "degraded"
	} else {
		resp.Redis = healthStatus{
			Status:  "ok",
			Latency: time.Since(redisStart).String(),
		}
	}

	status := http.StatusOK
	if resp.Status == "degraded" {
		status = http.StatusServiceUnavailable
	}

	respondJSON(w, status, resp, nil)
}

// ReadinessCheck handles GET /api/ready — returns 200 only when all dependencies
// (database and Redis) are reachable. Designed for Kubernetes readiness probes
// and load balancer health checks. Returns 503 if any dependency is unreachable.
func (h *HealthHandlers) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	ready := true

	resp := struct {
		Ready    bool         `json:"ready"`
		Database healthStatus `json:"database"`
		Redis    healthStatus `json:"redis"`
	}{Ready: true}

	// Check database with a tight timeout
	dbStart := time.Now()
	sqlDB, err := h.DB.DB()
	if err != nil {
		resp.Database = healthStatus{Status: "error", Error: "failed to get database connection"}
		ready = false
	} else {
		dbCtx, dbCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer dbCancel()
		if err := sqlDB.PingContext(dbCtx); err != nil {
			resp.Database = healthStatus{Status: "error", Error: "database ping failed"}
			ready = false
		} else {
			resp.Database = healthStatus{
				Status:  "ok",
				Latency: time.Since(dbStart).String(),
			}
		}
	}

	// Check Redis with a tight timeout
	redisStart := time.Now()
	redisCtx, redisCancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer redisCancel()
	if err := h.Queue.Ping(redisCtx); err != nil {
		resp.Redis = healthStatus{Status: "error", Error: "redis ping failed"}
		ready = false
	} else {
		resp.Redis = healthStatus{
			Status:  "ok",
			Latency: time.Since(redisStart).String(),
		}
	}

	resp.Ready = ready
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	respondJSON(w, status, resp, nil)
}
