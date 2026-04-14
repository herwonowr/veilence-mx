package queue

import (
	"context"
	"log/slog"
	"time"
)

// ProcessFunc is a function that processes a job. Return an error to retry.
type ProcessFunc func(ctx context.Context, job *Job) error

// Worker polls a Redis queue and processes jobs.
type Worker struct {
	queue        *Queue
	jobType      string
	process      ProcessFunc
	pollInterval time.Duration
	concurrency  int
}

// WorkerConfig holds worker configuration.
type WorkerConfig struct {
	PollInterval time.Duration // How often to check for new jobs (default: 1s)
	Concurrency  int           // Number of concurrent workers (default: 1)
}

// NewWorker creates a new queue worker.
func NewWorker(q *Queue, jobType string, process ProcessFunc, cfg WorkerConfig) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1 * time.Second
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	return &Worker{
		queue:        q,
		jobType:      jobType,
		process:      process,
		pollInterval: cfg.PollInterval,
		concurrency:  cfg.Concurrency,
	}
}

// Start begins the worker loop. Blocks until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	slog.Info("queue worker started", "type", w.jobType, "concurrency", w.concurrency)

	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, w.concurrency)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("queue worker shutting down", "type", w.jobType)
			return
		case <-ticker.C:
			// Try to acquire a semaphore slot
			select {
			case sem <- struct{}{}:
			default:
				continue // All workers busy
			}

			job, err := w.queue.Dequeue(ctx, w.jobType)
			if err != nil {
				slog.Error("failed to dequeue job", "type", w.jobType, "error", err)
				<-sem
				continue
			}

			if job == nil {
				<-sem // No jobs available
				continue
			}

			// Check if job should wait (backoff delay)
			if job.NextRunAt > time.Now().Unix() {
				// Put it back and wait
				job.Status = StatusPending
				w.queue.saveJob(ctx, job)
				w.queue.rdb.LPush(ctx, pendingList+w.jobType, job.ID)
				w.queue.rdb.ZRem(ctx, processingSet+w.jobType, job.ID)
				<-sem
				continue
			}

			go func(j *Job) {
				defer func() { <-sem }()
				w.processJob(ctx, j)
			}(job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *Job) {
	slog.Info("processing job",
		"job_id", job.ID,
		"type", job.Type,
		"reference_id", job.ReferenceID,
		"attempt", job.Attempts,
	)

	err := w.process(ctx, job)
	if err != nil {
		slog.Error("job failed",
			"job_id", job.ID,
			"type", job.Type,
			"reference_id", job.ReferenceID,
			"attempt", job.Attempts,
			"error", err.Error(),
		)
		if failErr := w.queue.Fail(ctx, job, err); failErr != nil {
			slog.Error("failed to mark job as failed", "job_id", job.ID, "error", failErr)
		}
		return
	}

	if err := w.queue.Complete(ctx, job); err != nil {
		slog.Error("failed to mark job as completed", "job_id", job.ID, "error", err)
	}

	slog.Info("job completed",
		"job_id", job.ID,
		"type", job.Type,
		"reference_id", job.ReferenceID,
	)
}
