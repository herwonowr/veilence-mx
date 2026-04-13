package handlers

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/veilence/veilence-mx/backend/internal/queue"
)

type queueStatsResponse struct {
	Diff    *queue.QueueStats `json:"diff"`
	Analyze *queue.QueueStats `json:"analyze"`
}

// GetQueueStats returns current queue statistics.
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireOrg middleware for access control only.
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
// requires org membership via RequireOrg middleware for access control only.
func (h *QueueHandlers) GetDeadJobs(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	jobType := r.URL.Query().Get("type")

	if jobType != "" {
		jobs, err := h.Queue.DeadJobs(r.Context(), jobType, 50)
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
		jobs, err := h.Queue.DeadJobs(r.Context(), jt, 50)
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
// requires org membership via RequireOrg middleware for access control only.
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
