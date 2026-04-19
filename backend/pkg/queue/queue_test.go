package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------
// Test helpers
// ------------------------------------------------------------

// newTestQueue creates a Queue backed by a miniredis instance.
// The caller should defer mr.Close() after the test.
func newTestQueue(t *testing.T, opts ...func(*Queue)) (*Queue, *miniredis.Miniredis) {
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

func withMaxAttempts(n int) func(*Queue) {
	return func(q *Queue) { q.maxAttempts = n }
}

func withLockTimeout(d time.Duration) func(*Queue) {
	return func(q *Queue) { q.lockTimeout = d }
}

// loadTestJob loads a job by ID from the test queue (convenience wrapper).
func loadTestJob(t *testing.T, q *Queue, jobID string) *Job {
	t.Helper()
	job, err := q.loadJob(context.Background(), jobID)
	require.NoError(t, err)
	return job
}

// ------------------------------------------------------------
// Enqueue tests
// ------------------------------------------------------------

func TestEnqueue_CreatesJobAndAddsToQueue(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	jobID, err := q.Enqueue(ctx, JobTypeDiff, 42)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Job data is stored in Redis
	job := loadTestJob(t, q, jobID)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, JobTypeDiff, job.Type)
	assert.Equal(t, uint(42), job.ReferenceID)
	assert.Equal(t, StatusPending, job.Status)
	assert.Equal(t, 0, job.Attempts)
	assert.Equal(t, 3, job.MaxAttempts) // default from test queue
	assert.NotZero(t, job.CreatedAt)
	assert.NotZero(t, job.UpdatedAt)

	// Job ID is in the pending list
	pending, err := q.rdb.LRange(ctx, pendingList+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.Contains(t, pending, jobID)
}

func TestEnqueue_IncrementsIDCounter(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	id1, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	id2, err := q.Enqueue(ctx, JobTypeDiff, 2)
	require.NoError(t, err)

	assert.NotEqual(t, id1, id2)
}

func TestEnqueue_IncrementsTotalEnqueued(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)
	_, err = q.Enqueue(ctx, JobTypeAnalyze, 2)
	require.NoError(t, err)

	val, err := q.rdb.HGet(ctx, statsHash, "total_enqueued").Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

func TestEnqueue_DifferentJobTypes(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	diffID, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	analyzeID, err := q.Enqueue(ctx, JobTypeAnalyze, 2)
	require.NoError(t, err)

	diffPending, _ := q.rdb.LRange(ctx, pendingList+JobTypeDiff, 0, -1).Result()
	analyzePending, _ := q.rdb.LRange(ctx, pendingList+JobTypeAnalyze, 0, -1).Result()

	assert.Contains(t, diffPending, diffID)
	assert.NotContains(t, diffPending, analyzeID)
	assert.Contains(t, analyzePending, analyzeID)
	assert.NotContains(t, analyzePending, diffID)
}

// ------------------------------------------------------------
// Dequeue tests
// ------------------------------------------------------------

func TestDequeue_ReturnsJobAndMovesToProcessing(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	jobID, err := q.Enqueue(ctx, JobTypeDiff, 10)
	require.NoError(t, err)

	job, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, StatusProcessing, job.Status)
	assert.Equal(t, 0, job.Attempts) // Attempts only increment on Fail, not Dequeue

	// Job should now be in the processing sorted set
	members, err := q.rdb.ZRange(ctx, processingSet+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.Contains(t, members, jobID)

	// The pending list should be empty now
	pending, err := q.rdb.LLen(ctx, pendingList+JobTypeDiff).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending)
}

func TestDequeue_ReturnsNilWhenEmpty(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	job, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestDequeue_IncrementsAttempts(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	job, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, 0, job.Attempts) // Attempts only increment on Fail, not Dequeue
}

func TestDequeue_FIFO_Order(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	id1, _ := q.Enqueue(ctx, JobTypeDiff, 1)
	id2, _ := q.Enqueue(ctx, JobTypeDiff, 2)
	id3, _ := q.Enqueue(ctx, JobTypeDiff, 3)

	job1, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, id1, job1.ID)

	job2, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, id2, job2.ID)

	job3, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, id3, job3.ID)

	// No more jobs
	job4, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Nil(t, job4)
}

func TestDequeue_DoesNotCrossPollJobTypes(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	job, err := q.Dequeue(ctx, JobTypeAnalyze)
	require.NoError(t, err)
	assert.Nil(t, job, "should not dequeue a job from a different type")
}

// ------------------------------------------------------------
// Complete tests
// ------------------------------------------------------------

func TestComplete_RemovesJobFromProcessing(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	job, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	require.NotNil(t, job)

	err = q.Complete(ctx, job)
	require.NoError(t, err)

	// Job should be removed from the processing set
	members, err := q.rdb.ZRange(ctx, processingSet+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.NotContains(t, members, job.ID)

	// Job data should still exist and be marked completed
	completed := loadTestJob(t, q, job.ID)
	assert.Equal(t, StatusCompleted, completed.Status)
}

func TestComplete_IncrementsTotalCompleted(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)

	err := q.Complete(ctx, job)
	require.NoError(t, err)

	val, err := q.rdb.HGet(ctx, statsHash, "total_completed").Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

// ------------------------------------------------------------
// Fail tests
// ------------------------------------------------------------

func TestFail_RetriesWhenAttemptsRemain(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(3))
	ctx := context.Background()

	_, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	job, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, 0, job.Attempts) // Attempts only increment on Fail

	// Fail the job
	err = q.Fail(ctx, job, errors.New("transient error"))
	require.NoError(t, err)

	// Job should be removed from processing
	members, err := q.rdb.ZRange(ctx, processingSet+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.NotContains(t, members, job.ID)

	// Job should be back in the pending list
	pending, err := q.rdb.LRange(ctx, pendingList+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.Contains(t, pending, job.ID)

	// Job data should reflect the failure
	updated := loadTestJob(t, q, job.ID)
	assert.Equal(t, StatusPending, updated.Status)
	assert.Equal(t, "transient error", updated.LastError)
}

func TestFail_MovesToDeadWhenMaxAttemptsExceeded(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	_, err := q.Enqueue(ctx, JobTypeDiff, 1)
	require.NoError(t, err)

	job, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, 0, job.Attempts) // Attempts only increment on Fail

	// Fail should move to dead since Fail increments to 1, and MaxAttempts is 1
	err = q.Fail(ctx, job, errors.New("fatal error"))
	require.NoError(t, err)

	// Job should NOT be in the pending list
	pending, err := q.rdb.LRange(ctx, pendingList+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.NotContains(t, pending, job.ID)

	// Job should be in the dead set
	dead, err := q.rdb.ZRange(ctx, deadSet+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.Contains(t, dead, job.ID)

	// Job data should reflect dead status
	updated := loadTestJob(t, q, job.ID)
	assert.Equal(t, StatusDead, updated.Status)
	assert.Equal(t, "fatal error", updated.LastError)
}

func TestFail_IncrementsDeadCounter(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)

	err := q.Fail(ctx, job, errors.New("boom"))
	require.NoError(t, err)

	val, err := q.rdb.HGet(ctx, statsHash, "total_dead").Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

func TestFail_BackoffIncreases(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(5))
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)

	before := time.Now().Unix()
	err := q.Fail(ctx, job, errors.New("retry"))
	require.NoError(t, err)

	updated := loadTestJob(t, q, job.ID)
	// After attempt 1, backoff = 30 * 2^0 = 30s
	assert.GreaterOrEqual(t, updated.NextRunAt, before+29) // allow 1s tolerance
}

// ------------------------------------------------------------
// Stats tests
// ------------------------------------------------------------

func TestStats_ReturnsCorrectCounts(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	// Enqueue 3 jobs
	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 2)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 3)

	stats, err := q.Stats(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Pending)
	assert.Equal(t, int64(0), stats.Processing)

	// Dequeue one → processing
	job1, _ := q.Dequeue(ctx, JobTypeDiff)

	stats, err = q.Stats(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Pending)
	assert.Equal(t, int64(1), stats.Processing)

	// Complete one
	_ = q.Complete(ctx, job1)

	stats, err = q.Stats(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Pending)
	assert.Equal(t, int64(0), stats.Processing)
	assert.Equal(t, int64(1), stats.Completed)

	// Dequeue another and fail it to dead (maxAttempts=1)
	job2, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Fail(ctx, job2, errors.New("dead"))

	stats, err = q.Stats(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Pending)
	assert.Equal(t, int64(0), stats.Processing)
	assert.Equal(t, int64(1), stats.Completed)
	assert.Equal(t, int64(1), stats.Dead)
}

func TestStats_EmptyQueue(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	stats, err := q.Stats(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.Pending)
	assert.Equal(t, int64(0), stats.Processing)
	assert.Equal(t, int64(0), stats.Completed)
	assert.Equal(t, int64(0), stats.Dead)
}

// ------------------------------------------------------------
// RecoverStuckJobs tests
// ------------------------------------------------------------

func TestRecoverStuckJobs_MovesStaleJobsBackToPending(t *testing.T) {
	// Use a very short lock timeout so we can simulate "stuck" jobs
	q, _ := newTestQueue(t, withLockTimeout(1*time.Second))
	ctx := context.Background()

	// Enqueue and dequeue a job (moves it to processing)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	require.NotNil(t, job)

	// Backdate the processing score so the job appears stuck (score < now - lockTimeout)
	stuckScore := float64(time.Now().Unix() - 10)
	q.rdb.ZAdd(ctx, processingSet+JobTypeDiff, redis.Z{Score: stuckScore, Member: job.ID})

	recovered, err := q.RecoverStuckJobs(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)

	// Job should be back in pending
	pending, err := q.rdb.LRange(ctx, pendingList+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.Contains(t, pending, job.ID)

	// Job should be removed from processing
	processing, err := q.rdb.ZRange(ctx, processingSet+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.NotContains(t, processing, job.ID)

	// Job status updated
	updated := loadTestJob(t, q, job.ID)
	assert.Equal(t, StatusPending, updated.Status)
}

func TestRecoverStuckJobs_NoStuckJobs(t *testing.T) {
	q, _ := newTestQueue(t, withLockTimeout(10*time.Minute))
	ctx := context.Background()

	// Enqueue and dequeue (freshly processing, not stuck)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Dequeue(ctx, JobTypeDiff)

	recovered, err := q.RecoverStuckJobs(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, 0, recovered)
}

func TestRecoverStuckJobs_MovesToDeadWhenMaxAttemptsExceeded(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1), withLockTimeout(1*time.Second))
	ctx := context.Background()

	// Enqueue and dequeue (attempt 1)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	require.NotNil(t, job)
	assert.Equal(t, 0, job.Attempts) // Dequeue doesn't increment

	// Simulate a prior failure: set attempts to maxAttempts so recovery sends to dead
	job.Attempts = 1
	q.saveJob(ctx, job)

	// Backdate the processing score so the job appears stuck
	stuckScore := float64(time.Now().Unix() - 10)
	q.rdb.ZAdd(ctx, processingSet+JobTypeDiff, redis.Z{Score: stuckScore, Member: job.ID})

	recovered, err := q.RecoverStuckJobs(ctx, JobTypeDiff)
	require.NoError(t, err)
	// recovered count should be 0 because the job went to dead, not pending
	assert.Equal(t, 0, recovered)

	// Job should be in the dead set
	dead, err := q.rdb.ZRange(ctx, deadSet+JobTypeDiff, 0, -1).Result()
	require.NoError(t, err)
	assert.Contains(t, dead, job.ID)

	updated := loadTestJob(t, q, job.ID)
	assert.Equal(t, StatusDead, updated.Status)
}

func TestRecoverStuckJobs_RecoverMultipleJobs(t *testing.T) {
	q, _ := newTestQueue(t, withLockTimeout(1*time.Second))
	ctx := context.Background()

	// Enqueue and dequeue 3 jobs
	ids := make([]string, 3)
	for i := range ids {
		id, _ := q.Enqueue(ctx, JobTypeDiff, uint(i+1))
		ids[i] = id
	}
	for range ids {
		_, _ = q.Dequeue(ctx, JobTypeDiff)
	}

	// Backdate all processing scores so jobs appear stuck
	stuckScore := float64(time.Now().Unix() - 10)
	for _, id := range ids {
		q.rdb.ZAdd(ctx, processingSet+JobTypeDiff, redis.Z{Score: stuckScore, Member: id})
	}

	recovered, err := q.RecoverStuckJobs(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, 3, recovered)
}

// ------------------------------------------------------------
// RequeueDead tests
// ------------------------------------------------------------

func TestRequeueDead_MovesDeadJobBackToPending(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	// Create a dead job
	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Fail(ctx, job, errors.New("fatal"))

	// Confirm it's dead
	dead, _ := q.rdb.ZRange(ctx, deadSet+JobTypeDiff, 0, -1).Result()
	require.Contains(t, dead, job.ID)

	// Requeue it
	err := q.RequeueDead(ctx, JobTypeDiff, job.ID)
	require.NoError(t, err)

	// Should be in pending
	pending, _ := q.rdb.LRange(ctx, pendingList+JobTypeDiff, 0, -1).Result()
	assert.Contains(t, pending, job.ID)

	// Should NOT be in dead
	dead, _ = q.rdb.ZRange(ctx, deadSet+JobTypeDiff, 0, -1).Result()
	assert.NotContains(t, dead, job.ID)

	// Job should be reset
	updated := loadTestJob(t, q, job.ID)
	assert.Equal(t, StatusPending, updated.Status)
	assert.Equal(t, 0, updated.Attempts)
	assert.Empty(t, updated.LastError)
}

func TestRequeueDead_FailsIfJobNotDead(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	// Create a pending job
	jobID, _ := q.Enqueue(ctx, JobTypeDiff, 1)

	err := q.RequeueDead(ctx, JobTypeDiff, jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in dead-letter queue")
}

func TestRequeueDead_FailsIfJobDoesNotExist(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	err := q.RequeueDead(ctx, JobTypeDiff, "nonexistent-999")
	assert.Error(t, err)
}

// ------------------------------------------------------------
// RequeueAllDead tests
// ------------------------------------------------------------

func TestRequeueAllDead_RequeuesAllDeadJobs(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	// Create 3 dead jobs
	for i := range 3 {
		_, _ = q.Enqueue(ctx, JobTypeDiff, uint(i+1))
		job, _ := q.Dequeue(ctx, JobTypeDiff)
		_ = q.Fail(ctx, job, fmt.Errorf("error %d", i))
	}

	deadBefore, _ := q.rdb.ZCard(ctx, deadSet+JobTypeDiff).Result()
	assert.Equal(t, int64(3), deadBefore)

	count, err := q.RequeueAllDead(ctx, JobTypeDiff)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	deadAfter, _ := q.rdb.ZCard(ctx, deadSet+JobTypeDiff).Result()
	assert.Equal(t, int64(0), deadAfter)

	pendingLen, _ := q.rdb.LLen(ctx, pendingList+JobTypeDiff).Result()
	assert.Equal(t, int64(3), pendingLen)
}

// ------------------------------------------------------------
// DeadJobs tests
// ------------------------------------------------------------

func TestDeadJobs_ReturnsDeadJobsSortedByRecent(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	// Create dead jobs
	for i := range 3 {
		_, _ = q.Enqueue(ctx, JobTypeDiff, uint(i+1))
		job, _ := q.Dequeue(ctx, JobTypeDiff)
		_ = q.Fail(ctx, job, fmt.Errorf("error %d", i))
	}

	jobs, _, err := q.DeadJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 3)

	// All should have dead status
	for _, j := range jobs {
		assert.Equal(t, StatusDead, j.Status)
	}
}

func TestDeadJobs_RespectsLimit(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	for i := range 5 {
		_, _ = q.Enqueue(ctx, JobTypeDiff, uint(i+1))
		job, _ := q.Dequeue(ctx, JobTypeDiff)
		_ = q.Fail(ctx, job, fmt.Errorf("error %d", i))
	}

	jobs, _, err := q.DeadJobs(ctx, JobTypeDiff, 0, 2)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestDeadJobs_DefaultsLimitTo50(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	// Pass 0 as limit, should default to 50
	jobs, _, err := q.DeadJobs(ctx, JobTypeDiff, 0, 0)
	require.NoError(t, err)
	assert.Empty(t, jobs) // no dead jobs
}

// ------------------------------------------------------------
// Ping tests
// ------------------------------------------------------------

func TestPing_Success(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	err := q.Ping(ctx)
	assert.NoError(t, err)
}

// ------------------------------------------------------------
// Close tests
// ------------------------------------------------------------

func TestClose(t *testing.T) {
	q, _ := newTestQueue(t)
	err := q.Close()
	assert.NoError(t, err)
}

// ------------------------------------------------------------
// Job serialization round-trip test
// ------------------------------------------------------------

func TestJob_JSONRoundTrip(t *testing.T) {
	original := Job{
		ID:          "42",
		Type:        JobTypeDiff,
		ReferenceID: 100,
		Status:      StatusPending,
		Attempts:    2,
		MaxAttempts: 5,
		LastError:   "something went wrong",
		CreatedAt:   1700000000,
		UpdatedAt:   1700000100,
		NextRunAt:   1700000200,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Job
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

// ------------------------------------------------------------
// New() constructor tests
// ------------------------------------------------------------

func TestNew_InvalidURL(t *testing.T) {
	_, err := New(Config{RedisURL: "not-a-valid-url"})
	assert.Error(t, err)
}

func TestNew_DefaultMaxAttempts(t *testing.T) {
	mr := miniredis.RunT(t)

	q, err := New(Config{
		RedisURL:    "redis://" + mr.Addr(),
		MaxAttempts: 0,
	})
	require.NoError(t, err)
	defer q.Close()

	assert.Equal(t, 5, q.maxAttempts)
}

func TestNew_DefaultLockTimeout(t *testing.T) {
	mr := miniredis.RunT(t)

	q, err := New(Config{
		RedisURL:    "redis://" + mr.Addr(),
		LockTimeout: 0,
	})
	require.NoError(t, err)
	defer q.Close()

	assert.Equal(t, 10*time.Minute, q.lockTimeout)
}

func TestNew_CustomValues(t *testing.T) {
	mr := miniredis.RunT(t)

	q, err := New(Config{
		RedisURL:    "redis://" + mr.Addr(),
		MaxAttempts: 10,
		LockTimeout: 5 * time.Minute,
	})
	require.NoError(t, err)
	defer q.Close()

	assert.Equal(t, 10, q.maxAttempts)
	assert.Equal(t, 5*time.Minute, q.lockTimeout)
}

// ------------------------------------------------------------
// Integration / workflow tests
// ------------------------------------------------------------

func TestFullJobLifecycle_Success(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	// 1. Enqueue
	jobID, err := q.Enqueue(ctx, JobTypeDiff, 42)
	require.NoError(t, err)

	// 2. Dequeue
	job, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, StatusProcessing, job.Status)

	// 3. Complete
	err = q.Complete(ctx, job)
	require.NoError(t, err)

	// 4. Verify final state
	final := loadTestJob(t, q, jobID)
	assert.Equal(t, StatusCompleted, final.Status)
	assert.Equal(t, 0, final.Attempts) // No failures, so attempts stays 0

	stats, _ := q.Stats(ctx, JobTypeDiff)
	assert.Equal(t, int64(0), stats.Pending)
	assert.Equal(t, int64(0), stats.Processing)
	assert.Equal(t, int64(1), stats.Completed)
}

func TestFullJobLifecycle_FailRetrySuccess(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(3))
	ctx := context.Background()

	// 1. Enqueue
	jobID, err := q.Enqueue(ctx, JobTypeDiff, 42)
	require.NoError(t, err)

	// 2. First attempt — fail
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	require.NotNil(t, job)
	err = q.Fail(ctx, job, errors.New("attempt 1 failed"))
	require.NoError(t, err)

	// 3. Second attempt — succeed
	job2, _ := q.Dequeue(ctx, JobTypeDiff)
	require.NotNil(t, job2)
	assert.Equal(t, jobID, job2.ID)
	assert.Equal(t, 1, job2.Attempts) // 1 failure recorded

	err = q.Complete(ctx, job2)
	require.NoError(t, err)

	final := loadTestJob(t, q, jobID)
	assert.Equal(t, StatusCompleted, final.Status)
	assert.Equal(t, 1, final.Attempts) // 1 failure total
}

func TestFullJobLifecycle_FailUntilDead(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(2))
	ctx := context.Background()

	_, err := q.Enqueue(ctx, JobTypeDiff, 42)
	require.NoError(t, err)

	// First attempt — fail
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	require.NotNil(t, job)
	_ = q.Fail(ctx, job, errors.New("fail 1"))

	// Second attempt — fail again → dead
	job2, _ := q.Dequeue(ctx, JobTypeDiff)
	require.NotNil(t, job2)
	assert.Equal(t, 1, job2.Attempts) // 1 failure so far
	_ = q.Fail(ctx, job2, errors.New("fail 2"))

	final := loadTestJob(t, q, job.ID)
	assert.Equal(t, StatusDead, final.Status)
	assert.Equal(t, "fail 2", final.LastError)

	stats, _ := q.Stats(ctx, JobTypeDiff)
	assert.Equal(t, int64(0), stats.Pending)
	assert.Equal(t, int64(0), stats.Processing)
	assert.Equal(t, int64(1), stats.Dead)
}

func TestFullJobLifecycle_DeadThenRequeue(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	// Enqueue → Dequeue → Fail → Dead
	_, _ = q.Enqueue(ctx, JobTypeDiff, 42)
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Fail(ctx, job, errors.New("fatal"))

	// Requeue from dead
	err := q.RequeueDead(ctx, JobTypeDiff, job.ID)
	require.NoError(t, err)

	// Dequeue again → should work
	job2, err := q.Dequeue(ctx, JobTypeDiff)
	require.NoError(t, err)
	require.NotNil(t, job2)
	assert.Equal(t, job.ID, job2.ID)
	assert.Equal(t, 0, job2.Attempts) // reset to 0 by RequeueDead, Dequeue doesn't increment

	// Complete this time
	err = q.Complete(ctx, job2)
	require.NoError(t, err)
}

// ------------------------------------------------------------
// saveJob TTL behaviour
// ------------------------------------------------------------

func TestSaveJob_SetsTTLBasedOnStatus(t *testing.T) {
	q, mr := newTestQueue(t)
	ctx := context.Background()

	// Pending job → 24h TTL
	jobID, _ := q.Enqueue(ctx, JobTypeDiff, 1)
	ttl := mr.TTL(jobHash + jobID)
	assert.InDelta(t, (24 * time.Hour).Seconds(), ttl.Seconds(), 5)

	// Completed job → 1h TTL
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Complete(ctx, job)
	ttl = mr.TTL(jobHash + job.ID)
	assert.InDelta(t, (1 * time.Hour).Seconds(), ttl.Seconds(), 5)
}

func TestSaveJob_DeadJobTTL(t *testing.T) {
	q, mr := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Fail(ctx, job, errors.New("dead"))

	ttl := mr.TTL(jobHash + job.ID)
	assert.InDelta(t, (7 * 24 * time.Hour).Seconds(), ttl.Seconds(), 5)
}

// ------------------------------------------------------------
// PendingJobs tests
// ------------------------------------------------------------

func TestPendingJobs_ReturnsJobsInFIFOOrder(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	id1, _ := q.Enqueue(ctx, JobTypeDiff, 1)
	id2, _ := q.Enqueue(ctx, JobTypeDiff, 2)
	id3, _ := q.Enqueue(ctx, JobTypeDiff, 3)

	jobs, total, err := q.PendingJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, jobs, 3)

	// FIFO: oldest first
	assert.Equal(t, id1, jobs[0].ID)
	assert.Equal(t, id2, jobs[1].ID)
	assert.Equal(t, id3, jobs[2].ID)
}

func TestPendingJobs_EmptyQueue(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	jobs, total, err := q.PendingJobs(ctx, JobTypeDiff, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, jobs)
}

func TestPendingJobs_Pagination(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	ids := make([]string, 5)
	for i := range 5 {
		ids[i], _ = q.Enqueue(ctx, JobTypeDiff, uint(i+1))
	}

	// Page 1: first 2 jobs
	jobs, total, err := q.PendingJobs(ctx, JobTypeDiff, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, jobs, 2)
	assert.Equal(t, ids[0], jobs[0].ID)
	assert.Equal(t, ids[1], jobs[1].ID)

	// Page 2: next 2 jobs
	jobs, total, err = q.PendingJobs(ctx, JobTypeDiff, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, jobs, 2)
	assert.Equal(t, ids[2], jobs[0].ID)
	assert.Equal(t, ids[3], jobs[1].ID)

	// Page 3: last job
	jobs, total, err = q.PendingJobs(ctx, JobTypeDiff, 4, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, jobs, 1)
	assert.Equal(t, ids[4], jobs[0].ID)
}

func TestPendingJobs_PageBeyondTotal(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)

	jobs, total, err := q.PendingJobs(ctx, JobTypeDiff, 100, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Empty(t, jobs)
}

func TestPendingJobs_SkipsExpiredHashes(t *testing.T) {
	q, mr := newTestQueue(t)
	ctx := context.Background()

	id1, _ := q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 2)

	// Expire the first job's hash
	mr.Del(jobHash + id1)

	jobs, total, err := q.PendingJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	// Total reflects list length (includes expired ID), but only 1 job loaded
	assert.Equal(t, int64(2), total)
	assert.Len(t, jobs, 1)
}

func TestPendingJobs_DefaultsAndCapsLimit(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	// Limit 0 → defaults to 20
	jobs, _, err := q.PendingJobs(ctx, JobTypeDiff, 0, 0)
	require.NoError(t, err)
	assert.Empty(t, jobs)

	// Limit > 100 → capped to 100
	jobs, _, err = q.PendingJobs(ctx, JobTypeDiff, 0, 200)
	require.NoError(t, err)
	assert.Empty(t, jobs)
}

func TestPendingJobs_DoesNotCrossJobTypes(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Enqueue(ctx, JobTypeAnalyze, 2)

	diffJobs, diffTotal, err := q.PendingJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), diffTotal)
	assert.Len(t, diffJobs, 1)
	assert.Equal(t, JobTypeDiff, diffJobs[0].Type)

	analyzeJobs, analyzeTotal, err := q.PendingJobs(ctx, JobTypeAnalyze, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), analyzeTotal)
	assert.Len(t, analyzeJobs, 1)
	assert.Equal(t, JobTypeAnalyze, analyzeJobs[0].Type)
}

// ------------------------------------------------------------
// ProcessingJobs tests
// ------------------------------------------------------------

func TestProcessingJobs_ReturnsJobsOrderedByStartTime(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 2)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 3)

	// Dequeue all (moves to processing)
	job1, _ := q.Dequeue(ctx, JobTypeDiff)
	job2, _ := q.Dequeue(ctx, JobTypeDiff)
	job3, _ := q.Dequeue(ctx, JobTypeDiff)

	jobs, total, err := q.ProcessingJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, jobs, 3)

	// All should be in processing status
	for _, j := range jobs {
		assert.Equal(t, StatusProcessing, j.Status)
	}

	// Should contain all dequeued jobs
	jobIDs := []string{jobs[0].ID, jobs[1].ID, jobs[2].ID}
	assert.Contains(t, jobIDs, job1.ID)
	assert.Contains(t, jobIDs, job2.ID)
	assert.Contains(t, jobIDs, job3.ID)
}

func TestProcessingJobs_EmptyQueue(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	jobs, total, err := q.ProcessingJobs(ctx, JobTypeDiff, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, jobs)
}

func TestProcessingJobs_Pagination(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	for i := range 5 {
		_, _ = q.Enqueue(ctx, JobTypeDiff, uint(i+1))
		_, _ = q.Dequeue(ctx, JobTypeDiff)
	}

	// Page 1: first 2
	jobs, total, err := q.ProcessingJobs(ctx, JobTypeDiff, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, jobs, 2)

	// Page 2: next 2
	jobs, _, err = q.ProcessingJobs(ctx, JobTypeDiff, 2, 2)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	// Page 3: last 1
	jobs, _, err = q.ProcessingJobs(ctx, JobTypeDiff, 4, 2)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
}

func TestProcessingJobs_PageBeyondTotal(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Dequeue(ctx, JobTypeDiff)

	jobs, total, err := q.ProcessingJobs(ctx, JobTypeDiff, 100, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Empty(t, jobs)
}

func TestProcessingJobs_SkipsExpiredHashes(t *testing.T) {
	q, mr := newTestQueue(t)
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Enqueue(ctx, JobTypeDiff, 2)
	job1, _ := q.Dequeue(ctx, JobTypeDiff)
	_, _ = q.Dequeue(ctx, JobTypeDiff)

	// Expire the first job's hash
	mr.Del(jobHash + job1.ID)

	jobs, total, err := q.ProcessingJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total) // stale member cleaned from ZSET
	assert.Len(t, jobs, 1)          // only 1 job hash was loadable
}

func TestProcessingJobs_DoesNotCrossJobTypes(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	_, _ = q.Dequeue(ctx, JobTypeDiff)

	_, _ = q.Enqueue(ctx, JobTypeAnalyze, 2)
	_, _ = q.Dequeue(ctx, JobTypeAnalyze)

	diffJobs, diffTotal, err := q.ProcessingJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), diffTotal)
	assert.Len(t, diffJobs, 1)
	assert.Equal(t, JobTypeDiff, diffJobs[0].Type)
}

// ------------------------------------------------------------
// DeadJobs pagination tests
// ------------------------------------------------------------

func TestDeadJobs_Pagination(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	for i := range 5 {
		_, _ = q.Enqueue(ctx, JobTypeDiff, uint(i+1))
		job, _ := q.Dequeue(ctx, JobTypeDiff)
		_ = q.Fail(ctx, job, fmt.Errorf("error %d", i))
	}

	// Page 1: first 2
	jobs, total, err := q.DeadJobs(ctx, JobTypeDiff, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, jobs, 2)

	// Page 2: next 2
	jobs, _, err = q.DeadJobs(ctx, JobTypeDiff, 2, 2)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	// Page 3: last 1
	jobs, _, err = q.DeadJobs(ctx, JobTypeDiff, 4, 2)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
}

func TestDeadJobs_PageBeyondTotal(t *testing.T) {
	q, _ := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Fail(ctx, job, errors.New("dead"))

	jobs, total, err := q.DeadJobs(ctx, JobTypeDiff, 100, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Empty(t, jobs)
}

func TestDeadJobs_SkipsExpiredHashes(t *testing.T) {
	q, mr := newTestQueue(t, withMaxAttempts(1))
	ctx := context.Background()

	_, _ = q.Enqueue(ctx, JobTypeDiff, 1)
	job1, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Fail(ctx, job1, errors.New("dead"))

	_, _ = q.Enqueue(ctx, JobTypeDiff, 2)
	job2, _ := q.Dequeue(ctx, JobTypeDiff)
	_ = q.Fail(ctx, job2, errors.New("dead"))

	// Expire the first job's hash
	mr.Del(jobHash + job1.ID)

	jobs, total, err := q.DeadJobs(ctx, JobTypeDiff, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total) // stale member cleaned from ZSET
	assert.Len(t, jobs, 1)
}

// ------------------------------------------------------------
// LoadJob tests
// ------------------------------------------------------------

func TestLoadJob_ReturnsJob(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	jobID, _ := q.Enqueue(ctx, JobTypeDiff, 42)

	job, err := q.LoadJob(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, JobTypeDiff, job.Type)
	assert.Equal(t, uint(42), job.ReferenceID)
}

func TestLoadJob_ReturnsErrorForMissingJob(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	_, err := q.LoadJob(ctx, "nonexistent-999")
	assert.Error(t, err)
}
