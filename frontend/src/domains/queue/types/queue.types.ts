export interface QueueStats {
  pending: number
  processing: number
  completed: number
  failed: number
  dead: number
}

export interface QueueStatsResponse {
  diff: QueueStats
  analyze: QueueStats
}

export interface QueueJob {
  id: string
  type: string
  referenceId: number
  status: string
  attempts: number
  maxAttempts: number
  lastError?: string
  createdAt: number
  updatedAt: number
  nextRunAt: number
}

export type QueueJobStatus = "pending" | "processing" | "dead"

export type QueueJobType = "diff" | "analyze"

export interface QueueJobsParams {
  type: QueueJobType
  status: QueueJobStatus
  page?: number
  limit?: number
}
