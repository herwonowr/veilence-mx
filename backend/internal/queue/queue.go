package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	JobTypeDiff    = "diff"
	JobTypeAnalyze = "analyze"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusDead       = "dead"
)

const (
	keyPrefix     = "veilence:queue:"
	pendingList   = keyPrefix + "pending:"
	processingSet = keyPrefix + "processing:"
	deadSet       = keyPrefix + "dead:"
	jobHash       = keyPrefix + "job:"
	statsHash     = keyPrefix + "stats"
	jobIDCounter  = keyPrefix + "id_counter"
)

type Job struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	ReferenceID uint   `json:"referenceId"`
	Status      string `json:"status"`
	Attempts    int    `json:"attempts"`
	MaxAttempts int    `json:"maxAttempts"`
	LastError   string `json:"lastError,omitempty"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
	NextRunAt   int64  `json:"nextRunAt"`
}

type QueueStats struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Dead       int64 `json:"dead"`
}

// Enqueuer is the minimal interface needed by handlers that only enqueue jobs.
// This allows test code to supply a mock without requiring a live Redis connection.
type Enqueuer interface {
	Enqueue(ctx context.Context, jobType string, referenceID uint) (string, error)
}

type Queue struct {
	rdb         *redis.Client
	maxAttempts int
	lockTimeout time.Duration
}

type Config struct {
	RedisURL    string
	MaxAttempts int
	LockTimeout time.Duration
}

func New(cfg Config) (*Queue, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parsing redis URL: %w", err)
	}

	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	lockTimeout := cfg.LockTimeout
	if lockTimeout <= 0 {
		lockTimeout = 10 * time.Minute
	}

	return &Queue{
		rdb:         rdb,
		maxAttempts: maxAttempts,
		lockTimeout: lockTimeout,
	}, nil
}

func (q *Queue) Close() error {
	return q.rdb.Close()
}

// Ping checks the Redis connection by sending a PING command.
func (q *Queue) Ping(ctx context.Context) error {
	return q.rdb.Ping(ctx).Err()
}

func (q *Queue) Enqueue(ctx context.Context, jobType string, referenceID uint) (string, error) {
	id, err := q.rdb.Incr(ctx, jobIDCounter).Result()
	if err != nil {
		return "", fmt.Errorf("generating job ID: %w", err)
	}

	now := time.Now().Unix()
	job := Job{
		ID:          strconv.FormatInt(id, 10),
		Type:        jobType,
		ReferenceID: referenceID,
		Status:      StatusPending,
		Attempts:    0,
		MaxAttempts: q.maxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
		NextRunAt:   now,
	}

	data, err := json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("marshaling job: %w", err)
	}

	pipe := q.rdb.Pipeline()
	pipe.Set(ctx, jobHash+job.ID, data, 24*time.Hour)
	pipe.LPush(ctx, pendingList+jobType, job.ID)
	pipe.HIncrBy(ctx, statsHash, "total_enqueued", 1)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("enqueuing job: %w", err)
	}

	return job.ID, nil
}

func (q *Queue) Dequeue(ctx context.Context, jobType string) (*Job, error) {
	jobID, err := q.rdb.RPopLPush(ctx, pendingList+jobType, pendingList+jobType+"_temp").Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dequeuing job: %w", err)
	}

	q.rdb.LRem(ctx, pendingList+jobType+"_temp", 1, jobID)

	job, err := q.loadJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	job.Status = StatusProcessing
	job.Attempts++
	job.UpdatedAt = now

	if err := q.saveJob(ctx, job); err != nil {
		return nil, err
	}

	q.rdb.ZAdd(ctx, processingSet+jobType, redis.Z{
		Score:  float64(now),
		Member: job.ID,
	})

	return job, nil
}

func (q *Queue) Complete(ctx context.Context, job *Job) error {
	job.Status = StatusCompleted
	job.UpdatedAt = time.Now().Unix()

	pipe := q.rdb.Pipeline()
	data, _ := json.Marshal(job)
	pipe.Set(ctx, jobHash+job.ID, data, 1*time.Hour)
	pipe.ZRem(ctx, processingSet+job.Type, job.ID)
	pipe.HIncrBy(ctx, statsHash, "total_completed", 1)
	_, err := pipe.Exec(ctx)
	return err
}

func (q *Queue) Fail(ctx context.Context, job *Job, jobErr error) error {
	job.LastError = jobErr.Error()
	job.UpdatedAt = time.Now().Unix()

	q.rdb.ZRem(ctx, processingSet+job.Type, job.ID)

	if job.Attempts >= job.MaxAttempts {
		job.Status = StatusDead
		if err := q.saveJob(ctx, job); err != nil {
			return err
		}

		pipe := q.rdb.Pipeline()
		pipe.ZAdd(ctx, deadSet+job.Type, redis.Z{
			Score:  float64(job.UpdatedAt),
			Member: job.ID,
		})
		pipe.HIncrBy(ctx, statsHash, "total_dead", 1)
		_, err := pipe.Exec(ctx)

		slog.Warn("job moved to dead-letter queue",
			"job_id", job.ID,
			"type", job.Type,
			"reference_id", job.ReferenceID,
			"attempts", job.Attempts,
			"error", jobErr.Error(),
		)
		return err
	}

	backoff := time.Duration(30*math.Pow(2, float64(job.Attempts-1))) * time.Second
	job.Status = StatusPending
	job.NextRunAt = time.Now().Add(backoff).Unix()

	if err := q.saveJob(ctx, job); err != nil {
		return err
	}

	q.rdb.LPush(ctx, pendingList+job.Type, job.ID)

	slog.Info("job scheduled for retry",
		"job_id", job.ID,
		"type", job.Type,
		"reference_id", job.ReferenceID,
		"attempt", job.Attempts,
		"max_attempts", job.MaxAttempts,
		"backoff", backoff.String(),
		"error", jobErr.Error(),
	)

	return nil
}

func (q *Queue) RecoverStuckJobs(ctx context.Context, jobType string) (int, error) {
	cutoff := float64(time.Now().Add(-q.lockTimeout).Unix())

	stuckIDs, err := q.rdb.ZRangeByScore(ctx, processingSet+jobType, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(cutoff, 'f', 0, 64),
	}).Result()
	if err != nil {
		return 0, fmt.Errorf("finding stuck jobs: %w", err)
	}

	recovered := 0
	for _, jobID := range stuckIDs {
		job, err := q.loadJob(ctx, jobID)
		if err != nil {
			slog.Error("failed to load stuck job", "job_id", jobID, "error", err)
			continue
		}

		q.rdb.ZRem(ctx, processingSet+jobType, jobID)

		if job.Attempts >= job.MaxAttempts {
			job.Status = StatusDead
			job.LastError = "recovered from stuck state, max attempts exceeded"
			job.UpdatedAt = time.Now().Unix()
			q.saveJob(ctx, job)
			q.rdb.ZAdd(ctx, deadSet+jobType, redis.Z{
				Score:  float64(job.UpdatedAt),
				Member: job.ID,
			})
			slog.Warn("stuck job moved to dead-letter", "job_id", jobID, "type", jobType)
		} else {
			job.Status = StatusPending
			job.UpdatedAt = time.Now().Unix()
			q.saveJob(ctx, job)
			q.rdb.LPush(ctx, pendingList+jobType, jobID)
			recovered++
			slog.Info("recovered stuck job", "job_id", jobID, "type", jobType, "attempts", job.Attempts)
		}
	}

	return recovered, nil
}

func (q *Queue) RequeueDead(ctx context.Context, jobType string, jobID string) error {
	job, err := q.loadJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Status != StatusDead {
		return fmt.Errorf("job %s is not in dead-letter queue (status: %s)", jobID, job.Status)
	}

	job.Status = StatusPending
	job.Attempts = 0
	job.LastError = ""
	job.UpdatedAt = time.Now().Unix()
	job.NextRunAt = time.Now().Unix()

	if err := q.saveJob(ctx, job); err != nil {
		return err
	}

	pipe := q.rdb.Pipeline()
	pipe.ZRem(ctx, deadSet+jobType, jobID)
	pipe.LPush(ctx, pendingList+jobType, jobID)
	_, err = pipe.Exec(ctx)
	return err
}

func (q *Queue) RequeueAllDead(ctx context.Context, jobType string) (int, error) {
	deadIDs, err := q.rdb.ZRange(ctx, deadSet+jobType, 0, -1).Result()
	if err != nil {
		return 0, fmt.Errorf("listing dead jobs: %w", err)
	}

	count := 0
	for _, jobID := range deadIDs {
		if err := q.RequeueDead(ctx, jobType, jobID); err != nil {
			slog.Error("failed to requeue dead job", "job_id", jobID, "error", err)
			continue
		}
		count++
	}
	return count, nil
}

func (q *Queue) Stats(ctx context.Context, jobType string) (*QueueStats, error) {
	pending, err := q.rdb.LLen(ctx, pendingList+jobType).Result()
	if err != nil {
		return nil, err
	}

	processing, err := q.rdb.ZCard(ctx, processingSet+jobType).Result()
	if err != nil {
		return nil, err
	}

	dead, err := q.rdb.ZCard(ctx, deadSet+jobType).Result()
	if err != nil {
		return nil, err
	}

	completed, _ := q.rdb.HGet(ctx, statsHash, "total_completed").Int64()
	failed, _ := q.rdb.HGet(ctx, statsHash, "total_dead").Int64()

	return &QueueStats{
		Pending:    pending,
		Processing: processing,
		Completed:  completed,
		Failed:     failed,
		Dead:       dead,
	}, nil
}

// PendingJobs returns jobs from the pending list for a given type.
// Returns jobs in FIFO order (oldest first). Supports offset/limit pagination.
// Jobs whose hash key has expired are skipped gracefully.
func (q *Queue) PendingJobs(ctx context.Context, jobType string, offset, limit int) ([]Job, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	total, err := q.rdb.LLen(ctx, pendingList+jobType).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("counting pending jobs: %w", err)
	}

	if total == 0 || int64(offset) >= total {
		return []Job{}, total, nil
	}

	// LRange uses 0-based inclusive start/stop. We read from the right end (FIFO)
	// because LPush pushes to the left and RPop pops from the right.
	// To get FIFO order (oldest first), we reverse: oldest is at the tail.
	start := int64(offset)
	stop := start + int64(limit) - 1

	// Pending list is reversed (LPush adds at head, RPop reads at tail).
	// To get FIFO order, read from the tail backwards.
	tailStart := total - 1 - stop
	tailStop := total - 1 - start
	if tailStart < 0 {
		tailStart = 0
	}

	jobIDs, err := q.rdb.LRange(ctx, pendingList+jobType, tailStart, tailStop).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("listing pending jobs: %w", err)
	}

	// Reverse to get FIFO order (oldest first)
	for i, j := 0, len(jobIDs)-1; i < j; i, j = i+1, j-1 {
		jobIDs[i], jobIDs[j] = jobIDs[j], jobIDs[i]
	}

	jobs := make([]Job, 0, len(jobIDs))
	for _, id := range jobIDs {
		job, err := q.loadJob(ctx, id)
		if err != nil {
			continue // expired hash key — skip gracefully
		}
		jobs = append(jobs, *job)
	}
	return jobs, total, nil
}

// ProcessingJobs returns jobs from the processing sorted set for a given type.
// Returns jobs ordered by processing start time (oldest first). Supports offset/limit pagination.
// Jobs whose hash key has expired are skipped gracefully.
func (q *Queue) ProcessingJobs(ctx context.Context, jobType string, offset, limit int) ([]Job, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	total, err := q.rdb.ZCard(ctx, processingSet+jobType).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("counting processing jobs: %w", err)
	}

	if total == 0 || int64(offset) >= total {
		return []Job{}, total, nil
	}

	jobIDs, err := q.rdb.ZRange(ctx, processingSet+jobType, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("listing processing jobs: %w", err)
	}

	jobs := make([]Job, 0, len(jobIDs))
	for _, id := range jobIDs {
		job, err := q.loadJob(ctx, id)
		if err != nil {
			continue // expired hash key — skip gracefully
		}
		jobs = append(jobs, *job)
	}
	return jobs, total, nil
}

// DeadJobs returns jobs from the dead sorted set for a given type.
// Returns jobs ordered by most recent first. Supports offset/limit pagination.
// Jobs whose hash key has expired are skipped gracefully.
//
// Deprecated: For new code, use GetQueueJobs with status=dead. This method
// signature is maintained for backward compatibility with the GET /api/queue/dead endpoint.
func (q *Queue) DeadJobs(ctx context.Context, jobType string, offset, limit int) ([]Job, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	total, err := q.rdb.ZCard(ctx, deadSet+jobType).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("counting dead jobs: %w", err)
	}

	if total == 0 || int64(offset) >= total {
		return []Job{}, total, nil
	}

	deadIDs, err := q.rdb.ZRevRange(ctx, deadSet+jobType, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("listing dead jobs: %w", err)
	}

	jobs := make([]Job, 0, len(deadIDs))
	for _, id := range deadIDs {
		job, err := q.loadJob(ctx, id)
		if err != nil {
			continue // expired hash key — skip gracefully
		}
		jobs = append(jobs, *job)
	}
	return jobs, total, nil
}

// LoadJob loads a single job by ID. Returns nil and an error if the job hash
// has expired or does not exist. Exported for use by handlers that need to
// look up a job (e.g., single dead job retry).
func (q *Queue) LoadJob(ctx context.Context, jobID string) (*Job, error) {
	return q.loadJob(ctx, jobID)
}

func (q *Queue) loadJob(ctx context.Context, jobID string) (*Job, error) {
	data, err := q.rdb.Get(ctx, jobHash+jobID).Bytes()
	if err != nil {
		return nil, fmt.Errorf("loading job %s: %w", jobID, err)
	}

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("unmarshaling job %s: %w", jobID, err)
	}
	return &job, nil
}

func (q *Queue) saveJob(ctx context.Context, job *Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshaling job: %w", err)
	}

	ttl := 24 * time.Hour
	if job.Status == StatusCompleted {
		ttl = 1 * time.Hour
	}
	if job.Status == StatusDead {
		ttl = 7 * 24 * time.Hour
	}

	return q.rdb.Set(ctx, jobHash+job.ID, data, ttl).Err()
}
