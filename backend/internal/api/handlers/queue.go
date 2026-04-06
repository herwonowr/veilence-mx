package handlers

import (
	"log/slog"
	"net/http"

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
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireOrg middleware for access control only.
func (h *QueueHandlers) GetDeadJobs(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	jobType := r.URL.Query().Get("type")
	if jobType == "" {
		jobType = queue.JobTypeAnalyze
	}

	jobs, err := h.Queue.DeadJobs(r.Context(), jobType, 50)
	if err != nil {
		slog.Error("failed to get dead jobs", "type", jobType, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get dead jobs")
		return
	}

	respondJSON(w, http.StatusOK, jobs, nil)
}

// RetryDeadJobs re-queues all dead-letter jobs for a given type.
// NOTE: Queue data is global (Redis-backed, not org-scoped). The endpoint
// requires org membership via RequireOrg middleware for access control only.
func (h *QueueHandlers) RetryDeadJobs(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	jobType := r.URL.Query().Get("type")
	if jobType == "" {
		jobType = queue.JobTypeAnalyze
	}

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
}
