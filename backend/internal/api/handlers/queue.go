package handlers

import (
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/queue"
)

type queueStatsResponse struct {
	Diff    *queue.QueueStats `json:"diff"`
	Analyze *queue.QueueStats `json:"analyze"`
}

// GetQueueStats returns current queue statistics.
func (h *Handlers) GetQueueStats(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		respondError(w, http.StatusInternalServerError, "queue not configured")
		return
	}

	diffStats, err := h.Queue.Stats(r.Context(), queue.JobTypeDiff)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get diff queue stats: "+err.Error())
		return
	}

	analyzeStats, err := h.Queue.Stats(r.Context(), queue.JobTypeAnalyze)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get analyze queue stats: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, queueStatsResponse{
		Diff:    diffStats,
		Analyze: analyzeStats,
	}, nil)
}

// GetDeadJobs returns dead-letter jobs for a given queue type.
func (h *Handlers) GetDeadJobs(w http.ResponseWriter, r *http.Request) {
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
		respondError(w, http.StatusInternalServerError, "failed to get dead jobs: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, jobs, nil)
}

// RetryDeadJobs re-queues all dead-letter jobs for a given type.
func (h *Handlers) RetryDeadJobs(w http.ResponseWriter, r *http.Request) {
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
		respondError(w, http.StatusInternalServerError, "failed to retry dead jobs: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"message": "dead jobs re-queued",
		"count":   count,
	}, nil)
}
