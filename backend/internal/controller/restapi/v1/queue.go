package v1

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/veilence/veilence-mx/backend/pkg/queue"
)

// validJobTypes lists the accepted queue type parameter values.
var validJobTypes = map[string]bool{
	queue.JobTypeDiff:    true,
	queue.JobTypeAnalyze: true,
}

// validJobStatuses lists the browsable status parameter values.
var validJobStatuses = map[string]bool{
	queue.StatusPending:    true,
	queue.StatusProcessing: true,
	queue.StatusDead:       true,
}

type queueStatsResponse struct {
	Diff    *queue.QueueStats `json:"diff"`
	Analyze *queue.QueueStats `json:"analyze"`
}

// GetQueueStats returns current queue statistics.
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireWorkspace middleware for access control only.
func (h *QueueHandlers) GetQueueStats(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	diffStats, err := h.Queue.Stats(r.Context(), queue.JobTypeDiff)
	if err != nil {
		slog.Error("failed to get diff queue stats", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get diff queue stats")
		return
	}

	analyzeStats, err := h.Queue.Stats(r.Context(), queue.JobTypeAnalyze)
	if err != nil {
		slog.Error("failed to get analyze queue stats", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get analyze queue stats")
		return
	}

	respondJSON(w, http.StatusOK, queueStatsResponse{
		Diff:    diffStats,
		Analyze: analyzeStats,
	}, nil)
}

// GetDeadJobs returns dead-letter jobs for a given queue type.
// When no type is specified, returns dead jobs from all queue types.
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireWorkspace middleware for access control only.
//
// Deprecated: Use GET /api/queue/jobs?status=dead instead. This endpoint
// is maintained for backward compatibility and will be removed in v1.1.0.
func (h *QueueHandlers) GetDeadJobs(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	jobType := r.URL.Query().Get("type")

	if jobType != "" {
		jobs, _, err := h.Queue.DeadJobs(r.Context(), jobType, 0, 50)
		if err != nil {
			slog.Error("failed to get dead jobs", "type", jobType, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to get dead jobs")
			return
		}
		respondJSON(w, http.StatusOK, jobs, nil)
		return
	}

	// No type filter — fetch from all queue types and merge
	var allJobs []queue.Job
	for _, jt := range []string{queue.JobTypeDiff, queue.JobTypeAnalyze} {
		jobs, _, err := h.Queue.DeadJobs(r.Context(), jt, 0, 50)
		if err != nil {
			slog.Error("failed to get dead jobs", "type", jt, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to get dead jobs")
			return
		}
		allJobs = append(allJobs, jobs...)
	}

	// Sort by most recent first (updatedAt descending)
	sort.Slice(allJobs, func(i, j int) bool {
		return allJobs[i].UpdatedAt > allJobs[j].UpdatedAt
	})

	respondJSON(w, http.StatusOK, allJobs, nil)
}

// RetryDeadJobs re-queues all dead-letter jobs for a given type.
// When no type is specified, retries dead jobs from all queue types.
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireWorkspace middleware for access control only.
func (h *QueueHandlers) RetryDeadJobs(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	jobType := r.URL.Query().Get("type")

	if jobType != "" {
		count, err := h.Queue.RequeueAllDead(r.Context(), jobType)
		if err != nil {
			slog.Error("failed to retry dead jobs", "type", jobType, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to retry dead jobs")
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"message": "dead jobs re-queued",
			"count":   count,
		}, nil)
		return
	}

	// No type filter — retry dead jobs from all queue types
	totalCount := 0
	for _, jt := range []string{queue.JobTypeDiff, queue.JobTypeAnalyze} {
		count, err := h.Queue.RequeueAllDead(r.Context(), jt)
		if err != nil {
			slog.Error("failed to retry dead jobs", "type", jt, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to retry dead jobs")
			return
		}
		totalCount += count
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"message": "dead jobs re-queued",
		"count":   totalCount,
	}, nil)
}

// GetQueueJobs returns a paginated list of jobs filtered by queue type and status.
// Query params: type (required: diff|analyze), status (required: pending|processing|dead),
// page (default 1), limit (default 20, max 100).
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireWorkspace middleware for access control only.
func (h *QueueHandlers) GetQueueJobs(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	jobType := r.URL.Query().Get("type")
	if jobType == "" {
		respondError(w, http.StatusBadRequest, "missing required parameter: type (diff or analyze)")
		return
	}
	if !validJobTypes[jobType] {
		respondError(w, http.StatusBadRequest, "invalid type parameter: must be 'diff' or 'analyze'")
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		respondError(w, http.StatusBadRequest, "missing required parameter: status (pending, processing, or dead)")
		return
	}
	if !validJobStatuses[status] {
		// Provide helpful message for completed/failed
		if status == "completed" {
			respondError(w, http.StatusBadRequest, "completed jobs are counter-only and cannot be listed — use GET /api/queue/stats for the completed count")
			return
		}
		if status == "failed" {
			respondError(w, http.StatusBadRequest, "failed is not a browsable status — retrying jobs appear as 'pending' with lastError set, exhausted jobs appear as 'dead'")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid status parameter: must be 'pending', 'processing', or 'dead'")
		return
	}

	page, limit := parsePagination(r)
	offset := (page - 1) * limit

	var (
		jobs  []queue.Job
		total int64
		err   error
	)

	switch status {
	case queue.StatusPending:
		jobs, total, err = h.Queue.PendingJobs(r.Context(), jobType, offset, limit)
	case queue.StatusProcessing:
		jobs, total, err = h.Queue.ProcessingJobs(r.Context(), jobType, offset, limit)
	case queue.StatusDead:
		jobs, total, err = h.Queue.DeadJobs(r.Context(), jobType, offset, limit)
	}

	if err != nil {
		slog.Error("failed to get queue jobs", "type", jobType, "status", status, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get queue jobs")
		return
	}

	respondJSON(w, http.StatusOK, jobs, &Meta{
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

// RetryDeadJob re-queues a single dead-letter job by its ID.
// The job's type is looked up from its stored data to route to the correct queue.
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireWorkspace middleware for access control only.
func (h *QueueHandlers) RetryDeadJob(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		respondError(w, http.StatusBadRequest, "missing job ID")
		return
	}

	// Load the job to determine its type and verify it's dead
	job, err := h.Queue.LoadJob(r.Context(), jobID)
	if err != nil {
		slog.Error("failed to load job for retry", "job_id", jobID, "error", err)
		respondError(w, http.StatusNotFound, "job not found or expired")
		return
	}

	if job.Status != queue.StatusDead {
		respondError(w, http.StatusBadRequest, "job is not in dead status (current status: "+job.Status+")")
		return
	}

	if err := h.Queue.RequeueDead(r.Context(), job.Type, jobID); err != nil {
		slog.Error("failed to retry dead job", "job_id", jobID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to retry job")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"message": "Job queued for retry",
	}, nil)
}
