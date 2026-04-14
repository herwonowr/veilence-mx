// Package healthuc implements the business logic for health checks.
package healthuc

import (
	"context"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// DBPinger pings the database to check connectivity.
type DBPinger interface {
	PingDB(ctx context.Context) error
}

// RedisPinger pings Redis to check connectivity.
type RedisPinger interface {
	Ping(ctx context.Context) error
}

// UseCase implements usecase.HealthService.
type UseCase struct {
	db    DBPinger
	redis RedisPinger
}

// New creates a new health UseCase.
func New(db DBPinger, redis RedisPinger) *UseCase {
	return &UseCase{db: db, redis: redis}
}

// HealthCheck checks database and Redis connectivity.
func (uc *UseCase) HealthCheck(ctx context.Context) *entity.HealthResponse {
	resp := &entity.HealthResponse{Status: "ok"}

	// Check database
	dbStart := time.Now()
	dbCtx, dbCancel := context.WithTimeout(ctx, 3*time.Second)
	defer dbCancel()
	if err := uc.db.PingDB(dbCtx); err != nil {
		resp.Database = entity.HealthStatus{Status: "error", Error: "database ping failed"}
		resp.Status = "degraded"
	} else {
		resp.Database = entity.HealthStatus{
			Status:  "ok",
			Latency: time.Since(dbStart).String(),
		}
	}

	// Check Redis
	redisStart := time.Now()
	redisCtx, redisCancel := context.WithTimeout(ctx, 3*time.Second)
	defer redisCancel()
	if err := uc.redis.Ping(redisCtx); err != nil {
		resp.Redis = entity.HealthStatus{Status: "error", Error: "redis ping failed"}
		resp.Status = "degraded"
	} else {
		resp.Redis = entity.HealthStatus{
			Status:  "ok",
			Latency: time.Since(redisStart).String(),
		}
	}

	return resp
}

// ReadinessCheck checks if all dependencies are reachable.
func (uc *UseCase) ReadinessCheck(ctx context.Context) *entity.ReadinessResponse {
	resp := &entity.ReadinessResponse{Ready: true}

	// Check database
	dbStart := time.Now()
	dbCtx, dbCancel := context.WithTimeout(ctx, 2*time.Second)
	defer dbCancel()
	if err := uc.db.PingDB(dbCtx); err != nil {
		resp.Database = entity.HealthStatus{Status: "error", Error: "database ping failed"}
		resp.Ready = false
	} else {
		resp.Database = entity.HealthStatus{
			Status:  "ok",
			Latency: time.Since(dbStart).String(),
		}
	}

	// Check Redis
	redisStart := time.Now()
	redisCtx, redisCancel := context.WithTimeout(ctx, 2*time.Second)
	defer redisCancel()
	if err := uc.redis.Ping(redisCtx); err != nil {
		resp.Redis = entity.HealthStatus{Status: "error", Error: "redis ping failed"}
		resp.Ready = false
	} else {
		resp.Redis = entity.HealthStatus{
			Status:  "ok",
			Latency: time.Since(redisStart).String(),
		}
	}

	return resp
}
