import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { QueueStatsResponse, QueueJob } from "@/domains/queue/types/queue.types"

export const getQueueStats = async (): Promise<
  ApiResponse<QueueStatsResponse>
> => fetchApi<QueueStatsResponse>("/api/queue/stats")

export const getDeadJobs = async (
  type?: string
): Promise<ApiResponse<QueueJob[]>> => {
  const query = type ? `?type=${type}` : ""
  return fetchApi<QueueJob[]>(`/api/queue/dead${query}`)
}

export const retryDeadJobs = async (
  type?: string
): Promise<ApiResponse<{ message: string; count: number }>> => {
  const query = type ? `?type=${type}` : ""
  return fetchApi<{ message: string; count: number }>(
    `/api/queue/retry-dead${query}`,
    { method: "POST" }
  )
}

export const retryDeadJob = async (
  jobId: string
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>(
    `/api/queue/dead/${jobId}/retry`,
    { method: "POST" }
  )
