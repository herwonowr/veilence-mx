package healthuc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/veilence/veilence-mx/backend/internal/usecase/healthuc"
)

// ---------------------------------------------------------------------------
// Mock DBPinger
// ---------------------------------------------------------------------------

type mockDBPinger struct {
	err error
}

func (m *mockDBPinger) PingDB(_ context.Context) error { return m.err }

// ---------------------------------------------------------------------------
// Mock RedisPinger
// ---------------------------------------------------------------------------

type mockRedisPinger struct {
	err error
}

func (m *mockRedisPinger) Ping(_ context.Context) error { return m.err }

// ---------------------------------------------------------------------------
// HealthCheck
// ---------------------------------------------------------------------------

func TestHealthCheck_AllHealthy(t *testing.T) {
	uc := healthuc.New(&mockDBPinger{}, &mockRedisPinger{})
	resp := uc.HealthCheck(context.Background())

	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "ok", resp.Database.Status)
	assert.Equal(t, "ok", resp.Redis.Status)
	assert.NotEmpty(t, resp.Database.Latency)
	assert.NotEmpty(t, resp.Redis.Latency)
	assert.Empty(t, resp.Database.Error)
	assert.Empty(t, resp.Redis.Error)
}

func TestHealthCheck_DBDown(t *testing.T) {
	uc := healthuc.New(
		&mockDBPinger{err: errors.New("connection refused")},
		&mockRedisPinger{},
	)
	resp := uc.HealthCheck(context.Background())

	assert.Equal(t, "degraded", resp.Status)
	assert.Equal(t, "error", resp.Database.Status)
	assert.Equal(t, "database ping failed", resp.Database.Error)
	assert.Equal(t, "ok", resp.Redis.Status)
}

func TestHealthCheck_RedisDown(t *testing.T) {
	uc := healthuc.New(
		&mockDBPinger{},
		&mockRedisPinger{err: errors.New("connection refused")},
	)
	resp := uc.HealthCheck(context.Background())

	assert.Equal(t, "degraded", resp.Status)
	assert.Equal(t, "ok", resp.Database.Status)
	assert.Equal(t, "error", resp.Redis.Status)
	assert.Equal(t, "redis ping failed", resp.Redis.Error)
}

func TestHealthCheck_BothDown(t *testing.T) {
	uc := healthuc.New(
		&mockDBPinger{err: errors.New("db down")},
		&mockRedisPinger{err: errors.New("redis down")},
	)
	resp := uc.HealthCheck(context.Background())

	assert.Equal(t, "degraded", resp.Status)
	assert.Equal(t, "error", resp.Database.Status)
	assert.Equal(t, "error", resp.Redis.Status)
}

// ---------------------------------------------------------------------------
// ReadinessCheck
// ---------------------------------------------------------------------------

func TestReadinessCheck_AllReady(t *testing.T) {
	uc := healthuc.New(&mockDBPinger{}, &mockRedisPinger{})
	resp := uc.ReadinessCheck(context.Background())

	assert.True(t, resp.Ready)
	assert.Equal(t, "ok", resp.Database.Status)
	assert.Equal(t, "ok", resp.Redis.Status)
	assert.NotEmpty(t, resp.Database.Latency)
	assert.NotEmpty(t, resp.Redis.Latency)
}

func TestReadinessCheck_DBDown(t *testing.T) {
	uc := healthuc.New(
		&mockDBPinger{err: errors.New("connection refused")},
		&mockRedisPinger{},
	)
	resp := uc.ReadinessCheck(context.Background())

	assert.False(t, resp.Ready)
	assert.Equal(t, "error", resp.Database.Status)
	assert.Equal(t, "ok", resp.Redis.Status)
}

func TestReadinessCheck_RedisDown(t *testing.T) {
	uc := healthuc.New(
		&mockDBPinger{},
		&mockRedisPinger{err: errors.New("connection refused")},
	)
	resp := uc.ReadinessCheck(context.Background())

	assert.False(t, resp.Ready)
	assert.Equal(t, "ok", resp.Database.Status)
	assert.Equal(t, "error", resp.Redis.Status)
}

func TestReadinessCheck_BothDown(t *testing.T) {
	uc := healthuc.New(
		&mockDBPinger{err: errors.New("db down")},
		&mockRedisPinger{err: errors.New("redis down")},
	)
	resp := uc.ReadinessCheck(context.Background())

	assert.False(t, resp.Ready)
	assert.Equal(t, "error", resp.Database.Status)
	assert.Equal(t, "error", resp.Redis.Status)
}
