import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { QueueStatsResponse, QueueJob, QueueJobsParams } from "@/domains/queue/types/queue.types"

export const getQueueStats = async (): Promise<
  ApiResponse<QueueStatsResponse>
> => fetchApi<QueueStatsResponse>("/api/queue/stats")

export const getQueueJobs = async (
  params: QueueJobsParams
): Promise<ApiResponse<QueueJob[]>> => {
  const searchParams = new URLSearchParams({
    type: params.type,
    status: params.status,
    ...(params.page ? { page: String(params.page) } : {}),
    ...(params.limit ? { limit: String(params.limit) } : {}),
  })
  return fetchApi<QueueJob[]>(`/api/queue/jobs?${searchParams}`)
}

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
