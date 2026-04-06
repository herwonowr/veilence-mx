package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestWorkerQueue creates a Queue with fast poll interval defaults.
func newTestWorkerQueue(t *testing.T, opts ...func(*Queue)) (*Queue, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	q := &Queue{
		rdb:         rdb,
		maxAttempts: 3,
		lockTimeout: 10 * time.Minute,
	}
	for _, o := range opts {
		o(q)
	}
	return q, mr
}

// waitFor polls a condition until it returns true or the timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, msg string, cond func() bool) {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for: %s", msg)
		case <-ticker.C:
			if cond() {
				return
			}
		}
	}
}

// ------------------------------------------------------------
// Worker — starts and processes a job
// ------------------------------------------------------------

func TestWorker_ProcessesJob(t *testing.T) {
	q, _ := newTestWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processed atomic.Int32

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		processed.Add(1)
		return nil
	}, WorkerConfig{
		PollInterval: 20 * time.Millisecond,
		Concurrency:  1,
	})

	// Enqueue a job
	jobID, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	go w.Start(ctx)

	waitFor(t, 2*time.Second, "job processed", func() bool {
		return processed.Load() >= 1
	})

	cancel() // stop worker

	assert.Equal(t, int32(1), processed.Load())

	// Job should be completed
	job := loadTestJob(t, q, jobID)
	assert.Equal(t, StatusCompleted, job.Status)
}

// ------------------------------------------------------------
// Worker — handles job failure
// ------------------------------------------------------------

func TestWorker_HandlesJobFailure(t *testing.T) {
	q, _ := newTestWorkerQueue(t, withMaxAttempts(1)) // maxAttempts=1 means first failure goes to dead
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var attempts atomic.Int32

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		attempts.Add(1)
		return errors.New("processing error")
	}, WorkerConfig{
		PollInterval: 20 * time.Millisecond,
		Concurrency:  1,
	})

	jobID, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	go w.Start(ctx)

	// Wait for the single attempt — with maxAttempts=1, it goes straight to dead
	waitFor(t, 2*time.Second, "job failed", func() bool {
		return attempts.Load() >= 1
	})

	cancel()

	// After maxAttempts=2 failures, job should be dead
	job := loadTestJob(t, q, jobID)
	assert.Equal(t, StatusDead, job.Status)
	assert.Contains(t, job.LastError, "processing error")
}

// ------------------------------------------------------------
// Worker — respects concurrency limits
// ------------------------------------------------------------

func TestWorker_RespectsConcurrencyLimit(t *testing.T) {
	q, _ := newTestWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxConcurrency := 2
	var (
		current   atomic.Int32
		maxSeen   atomic.Int32
		processed atomic.Int32
		mu        sync.Mutex
	)

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		c := current.Add(1)
		defer current.Add(-1)

		// Track the maximum concurrent executions we've observed
		mu.Lock()
		if c > maxSeen.Load() {
			maxSeen.Store(c)
		}
		mu.Unlock()

		// Simulate work
		time.Sleep(50 * time.Millisecond)
		processed.Add(1)
		return nil
	}, WorkerConfig{
		PollInterval: 10 * time.Millisecond,
		Concurrency:  maxConcurrency,
	})

	// Enqueue more jobs than the concurrency limit
	for i := range 6 {
		_, err := q.Enqueue(ctx, JobTypeDiff, uint(i+1))
		require.NoError(t, err)
	}

	go w.Start(ctx)

	// Wait for all jobs to complete
	waitFor(t, 5*time.Second, "all 6 jobs processed", func() bool {
		return processed.Load() >= 6
	})

	cancel()

	// The maximum observed concurrency should not exceed the limit
	assert.LessOrEqual(t, maxSeen.Load(), int32(maxConcurrency),
		"concurrency should not exceed configured limit of %d", maxConcurrency)
}

// ------------------------------------------------------------
// Worker — stops cleanly on context cancellation
// ------------------------------------------------------------

func TestWorker_StopsCleanly(t *testing.T) {
	q, _ := newTestWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		return nil
	}, WorkerConfig{
		PollInterval: 20 * time.Millisecond,
		Concurrency:  1,
	})

	go func() {
		w.Start(ctx)
		close(done)
	}()

	// Give the worker a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Worker should exit promptly
	select {
	case <-done:
		// success — worker exited
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop within 2 seconds of context cancellation")
	}
}

// ------------------------------------------------------------
// Worker — no-op when queue is empty
// ------------------------------------------------------------

func TestWorker_IdlesWhenQueueEmpty(t *testing.T) {
	q, _ := newTestWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls atomic.Int32

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		calls.Add(1)
		return nil
	}, WorkerConfig{
		PollInterval: 20 * time.Millisecond,
		Concurrency:  1,
	})

	go w.Start(ctx)

	// Let it run for a bit with no jobs
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(0), calls.Load(), "process function should not be called when queue is empty")
}

// ------------------------------------------------------------
// Worker — processes multiple job types independently
// ------------------------------------------------------------

func TestWorker_ProcessesOnlyItsJobType(t *testing.T) {
	q, _ := newTestWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var diffProcessed atomic.Int32

	// Worker only listens for "diff" jobs
	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		diffProcessed.Add(1)
		return nil
	}, WorkerConfig{
		PollInterval: 20 * time.Millisecond,
		Concurrency:  1,
	})

	// Enqueue jobs of both types
	_, _ = q.Enqueue(ctx, JobTypeAnalyze, 1) // should be ignored by this worker
	_, _ = q.Enqueue(ctx, JobTypeDiff, 2)

	go w.Start(ctx)

	waitFor(t, 2*time.Second, "diff job processed", func() bool {
		return diffProcessed.Load() >= 1
	})

	cancel()

	assert.Equal(t, int32(1), diffProcessed.Load())

	// The analyze job should still be pending — use fresh context
	pendingCtx := context.Background()
	pending, _ := q.rdb.LLen(pendingCtx, pendingList+JobTypeAnalyze).Result()
	assert.Equal(t, int64(1), pending)
}

// ------------------------------------------------------------
// NewWorker — default config values
// ------------------------------------------------------------

func TestNewWorker_DefaultConfig(t *testing.T) {
	q, _ := newTestWorkerQueue(t)

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		return nil
	}, WorkerConfig{})

	assert.Equal(t, 1*time.Second, w.pollInterval)
	assert.Equal(t, 1, w.concurrency)
	assert.Equal(t, JobTypeDiff, w.jobType)
}

func TestNewWorker_CustomConfig(t *testing.T) {
	q, _ := newTestWorkerQueue(t)

	w := NewWorker(q, JobTypeAnalyze, func(_ context.Context, job *Job) error {
		return nil
	}, WorkerConfig{
		PollInterval: 500 * time.Millisecond,
		Concurrency:  4,
	})

	assert.Equal(t, 500*time.Millisecond, w.pollInterval)
	assert.Equal(t, 4, w.concurrency)
	assert.Equal(t, JobTypeAnalyze, w.jobType)
}

// ------------------------------------------------------------
// Worker — processes job then calls Complete
// ------------------------------------------------------------

func TestWorker_CompletesSuccessfulJob(t *testing.T) {
	q, _ := newTestWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processed atomic.Int32

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		processed.Add(1)
		return nil
	}, WorkerConfig{
		PollInterval: 20 * time.Millisecond,
		Concurrency:  1,
	})

	jobID, _ := q.Enqueue(ctx, JobTypeDiff, 42)

	go w.Start(ctx)

	waitFor(t, 2*time.Second, "job completed", func() bool {
		return processed.Load() >= 1
	})

	// Give a moment for the Complete call to finish
	time.Sleep(50 * time.Millisecond)
	cancel()

	job := loadTestJob(t, q, jobID)
	assert.Equal(t, StatusCompleted, job.Status)

	// Stats should reflect completion — use fresh context since we cancelled
	statsCtx := context.Background()
	stats, err := q.Stats(statsCtx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Completed)
}

// ------------------------------------------------------------
// Worker — enqueue during processing
// ------------------------------------------------------------

func TestWorker_ProcessesJobsEnqueuedWhileRunning(t *testing.T) {
	q, _ := newTestWorkerQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processed atomic.Int32

	w := NewWorker(q, JobTypeDiff, func(_ context.Context, job *Job) error {
		processed.Add(1)
		return nil
	}, WorkerConfig{
		PollInterval: 20 * time.Millisecond,
		Concurrency:  1,
	})

	go w.Start(ctx)

	// Enqueue after worker started
	time.Sleep(50 * time.Millisecond)
	_, err := q.Enqueue(ctx, JobTypeDiff, 99)
	require.NoError(t, err)

	waitFor(t, 2*time.Second, "late-enqueued job processed", func() bool {
		return processed.Load() >= 1
	})

	cancel()
	assert.Equal(t, int32(1), processed.Load())
}
